package model

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTicketTestDB(t *testing.T) {
	t.Helper()
	originalDB, originalType := DB, common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "tickets.db")+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate"), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	connection.SetMaxOpenConns(4)
	DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	InitDBColumns()
	t.Cleanup(func() {
		require.NoError(t, connection.Close())
		DB = originalDB
		common.SetMainDatabaseType(originalType)
		InitDBColumns()
	})
	require.NoError(t, DB.AutoMigrate(&Ticket{}, &TicketMessage{}, &TicketActionLog{}, &Option{}, &NotificationBot{}, &NotificationTask{}, &NotificationTarget{}, &NotificationEventReceipt{}, &NotificationEvent{}, &NotificationDelivery{}))
}

func createTicketFixture(t *testing.T, owner int) *Ticket {
	t.Helper()
	ticket, err := CreateTicket(owner, TicketCreateInput{Title: "模型请求失败", Category: "technical", Priority: "normal", Content: "请帮忙确认请求状态"})
	require.NoError(t, err)
	return ticket
}

func subscribeTicketEvents(t *testing.T) {
	t.Helper()
	bot := &NotificationBot{Name: "工单通知", Token: "ticket-fixture-token", Enabled: true}
	require.NoError(t, CreateNotificationBot(bot))
	// 同时订阅已下线的旧事件，证明它们不再入队。
	for _, event := range []string{NotificationEventTypeTicketUserMessage, "ticket_created", "ticket_user_replied", "ticket_admin_replied", "ticket_status_changed"} {
		task := &NotificationTask{Name: event, EventType: event, BotId: bot.Id, Enabled: true}
		require.NoError(t, CreateNotificationTask(task))
		require.NoError(t, CreateNotificationTarget(&NotificationTarget{TaskId: task.Id, ChatId: "123", Enabled: true}))
	}
}

func TestTicketOwnerIsolationAndPersonalAdminScope(t *testing.T) {
	setupTicketTestDB(t)
	owned := createTicketFixture(t, 10)
	other := createTicketFixture(t, 20)
	page := &common.PageInfo{Page: 1, PageSize: 20}
	items, total, err := ListTickets(10, false, TicketFilter{UserID: 20}, page)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, owned.Id, items[0].Id)
	_, err = GetTicketDetail(other.Id, 10, false, 0)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = ReplyTicket(other.Id, 10, false, "不能越权回复")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	_, err = ChangeTicketStatus(other.Id, 10, false, TicketStatusClosed)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	items, total, err = ListTickets(10, true, TicketFilter{UserID: 20}, page)
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, items, 1)
	assert.Equal(t, other.Id, items[0].Id)
}

func TestTicketConversationLifecycleAndNotificationMetadata(t *testing.T) {
	setupTicketTestDB(t)
	subscribeTicketEvents(t)
	ticket := createTicketFixture(t, 10)
	assert.Equal(t, TicketStatusOpen, ticket.Status)
	detail, err := GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 1)
	assert.Equal(t, "user", detail.Messages[0].AuthorRole)
	message, err := ReplyTicket(ticket.Id, 99, true, "管理员正在排查")
	require.NoError(t, err)
	assert.Equal(t, "admin", message.AuthorRole)
	detail, err = GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	assert.Equal(t, TicketStatusProcessing, detail.Ticket.Status)
	_, err = ChangeTicketStatus(ticket.Id, 99, true, TicketStatusResolved)
	require.NoError(t, err)
	_, err = ReplyTicket(ticket.Id, 10, false, "secret-content-never-notify")
	require.NoError(t, err)
	detail, err = GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	assert.Equal(t, TicketStatusOpen, detail.Ticket.Status)
	_, err = ChangeTicketStatus(ticket.Id, 10, false, TicketStatusProcessing)
	assert.ErrorIs(t, err, ErrTicketInvalidInput)
	_, err = ChangeTicketStatus(ticket.Id, 10, false, TicketStatusClosed)
	require.NoError(t, err)
	_, err = ReplyTicket(ticket.Id, 10, false, "已关闭不能追问")
	assert.ErrorIs(t, err, ErrTicketClosed)
	_, err = ReplyTicket(ticket.Id, 99, true, "管理员也需要重开")
	assert.ErrorIs(t, err, ErrTicketClosed)
	_, err = ChangeTicketStatus(ticket.Id, 10, false, TicketStatusClosed)
	require.NoError(t, err)
	_, err = ChangeTicketStatus(ticket.Id, 10, false, TicketStatusOpen)
	require.NoError(t, err)
	_, err = ChangeTicketStatus(ticket.Id, 10, false, TicketStatusClosed)
	require.NoError(t, err)
	var events []NotificationEvent
	require.NoError(t, DB.Order("id").Find(&events).Error)
	// 只有用户新建与用户追加回复通知管理员；管理员回复和所有状态变更都不再通知。
	require.Len(t, events, 2)
	actions := make([]string, 0, len(events))
	keys := map[string]bool{}
	for _, event := range events {
		assert.Equal(t, NotificationEventTypeTicketUserMessage, event.EventType)
		assert.False(t, keys[event.EventKey])
		keys[event.EventKey] = true
		assert.NotContains(t, event.Payload, "secret-content-never-notify")
		var payload map[string]any
		require.NoError(t, common.UnmarshalJsonStr(event.Payload, &payload))
		assert.EqualValues(t, 10, payload["user_id"])
		assert.Equal(t, "user", payload["actor_role"])
		assert.NotContains(t, payload, "content")
		actions = append(actions, payload["action"].(string)+"/"+payload["action_label"].(string))
	}
	assert.Equal(t, []string{"created/新工单", "replied/用户追加回复"}, actions)
}

func TestMigrateTicketNotificationTasksMergesUserEventsAndDisablesAdminEvents(t *testing.T) {
	setupTicketTestDB(t)
	bot := &NotificationBot{Name: "工单通知", Token: "ticket-migration-token", Enabled: true}
	require.NoError(t, CreateNotificationBot(bot))
	tasks := []NotificationTask{
		{Name: "新工单", EventType: "ticket_created", Template: "{{mention}} 新工单 #{{ticket_id}}：{{title}}\n用户：{{user_id}}\n分类：{{category}} 优先级：{{priority}}"},
		{Name: "用户追问", EventType: "ticket_user_replied", Template: "自定义：{{title}}"},
		{Name: "管理员回复", EventType: "ticket_admin_replied", Template: "{{title}}"},
		{Name: "状态变更", EventType: "ticket_status_changed", Template: "{{status}}"},
		{Name: "渠道禁用", EventType: NotificationEventTypeChannelDisabled, Template: "{{channel_name}}"},
	}
	for i := range tasks {
		tasks[i].BotId, tasks[i].Enabled = bot.Id, true
		require.NoError(t, DB.Create(&tasks[i]).Error)
	}
	for round := 0; round < 2; round++ {
		require.NoError(t, migrateTicketNotificationTasks())
	}
	var stored []NotificationTask
	require.NoError(t, DB.Order("id").Find(&stored).Error)
	require.Len(t, stored, 5)
	assert.Equal(t, NotificationEventTypeTicketUserMessage, stored[0].EventType)
	assert.Equal(t, NotificationTicketUserMessageTemplate, stored[0].Template, "旧默认模板换成合并后的默认模板")
	assert.True(t, stored[0].Enabled)
	assert.Equal(t, NotificationEventTypeTicketUserMessage, stored[1].EventType)
	assert.Equal(t, "自定义：{{title}}", stored[1].Template, "自定义模板保留")
	assert.True(t, stored[1].Enabled)
	for _, task := range stored[2:4] {
		assert.False(t, task.Enabled, "已下线事件的任务停用但保留，便于管理员删除")
	}
	assert.Equal(t, NotificationEventTypeChannelDisabled, stored[4].EventType)
	assert.True(t, stored[4].Enabled)
}

func TestTicketValidationAndUnicodeLimits(t *testing.T) {
	setupTicketTestDB(t)
	valid := TicketCreateInput{Title: strings.Repeat("工", 120), Category: "general", Priority: "normal", Content: strings.Repeat("文", 10000)}
	ticket, err := CreateTicket(1, valid)
	require.NoError(t, err)
	for _, tc := range []struct {
		name  string
		input TicketCreateInput
	}{
		{"空白标题", TicketCreateInput{Title: " ", Category: "general", Priority: "normal", Content: "内容"}},
		{"标题超长", TicketCreateInput{Title: strings.Repeat("工", 121), Category: "general", Priority: "normal", Content: "内容"}},
		{"正文超长", TicketCreateInput{Title: "工单", Category: "general", Priority: "normal", Content: strings.Repeat("文", 10001)}},
		{"未知分类", TicketCreateInput{Title: "工单", Category: "other", Priority: "normal", Content: "内容"}},
		{"未知优先级", TicketCreateInput{Title: "工单", Category: "general", Priority: "urgent", Content: "内容"}},
		{"跨数据库零字符", TicketCreateInput{Title: "工单", Category: "general", Priority: "normal", Content: "a\x00b"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := CreateTicket(1, tc.input)
			assert.ErrorIs(t, err, ErrTicketInvalidInput)
		})
	}
	for _, content := range []string{"  ", strings.Repeat("a", 10001), "a\x00b"} {
		_, err := ReplyTicket(ticket.Id, 1, false, content)
		assert.ErrorIs(t, err, ErrTicketInvalidInput)
	}
	_, err = ChangeTicketStatus(ticket.Id, 1, true, "invalid")
	assert.ErrorIs(t, err, ErrTicketInvalidInput)
}

func TestTicketBoundedHistoryAndLiteralKeywordFilter(t *testing.T) {
	setupTicketTestDB(t)
	ticket := createTicketFixture(t, 10)
	for i := 0; i < 51; i++ {
		require.NoError(t, DB.Create(&TicketMessage{TicketId: ticket.Id, AuthorId: 10, AuthorRole: "user", Content: fmt.Sprint(i), CreatedAt: int64(i + 1)}).Error)
	}
	detail, err := GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 50)
	assert.True(t, detail.HasMore)
	for i := 1; i < len(detail.Messages); i++ {
		assert.Less(t, detail.Messages[i-1].Id, detail.Messages[i].Id)
	}
	older, err := GetTicketDetail(ticket.Id, 10, false, detail.Messages[0].Id)
	require.NoError(t, err)
	require.Len(t, older.Messages, 2)
	assert.False(t, older.HasMore)
	assert.Less(t, older.Messages[1].Id, detail.Messages[0].Id)
	_, err = CreateTicket(10, TicketCreateInput{Title: "100%_正常", Category: "general", Priority: "normal", Content: "内容"})
	require.NoError(t, err)
	items, total, err := ListTickets(10, false, TicketFilter{Keyword: "%_"}, &common.PageInfo{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.EqualValues(t, 1, total)
	assert.Equal(t, "100%_正常", items[0].Title)
}

func TestTicketNotificationFailureRollsBackBusinessWrite(t *testing.T) {
	setupTicketTestDB(t)
	subscribeTicketEvents(t)
	ticket := createTicketFixture(t, 10)
	fail := errors.New("notification write fixture failure")
	require.NoError(t, DB.Callback().Create().Before("gorm:create").Register("test:ticket_notification_fail", func(tx *gorm.DB) {
		if tx.Statement.Table == "notification_deliveries" {
			tx.AddError(fail)
		}
	}))
	t.Cleanup(func() { _ = DB.Callback().Create().Remove("test:ticket_notification_fail") })
	_, err := ReplyTicket(ticket.Id, 10, false, "应回滚")
	assert.ErrorIs(t, err, fail)
	_, err = CreateTicket(10, TicketCreateInput{Title: "应回滚", Category: "general", Priority: "normal", Content: "应回滚"})
	assert.ErrorIs(t, err, fail)
	var count int64
	require.NoError(t, DB.Model(&Ticket{}).Count(&count).Error)
	assert.EqualValues(t, 1, count)
	detail, err := GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 1)
	assert.Equal(t, TicketStatusOpen, detail.Ticket.Status)
	assert.EqualValues(t, 1, detail.Ticket.Version)

	// 管理员回复与状态变更不再入队通知，通知写入故障不能阻断它们。
	_, err = ReplyTicket(ticket.Id, 99, true, "管理员处理中")
	require.NoError(t, err)
	_, err = ChangeTicketStatus(ticket.Id, 10, false, TicketStatusClosed)
	require.NoError(t, err)
	detail, err = GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 2)
	assert.Equal(t, TicketStatusClosed, detail.Ticket.Status)
	assert.EqualValues(t, 3, detail.Ticket.Version)
}

func TestTicketConcurrentRepliesPreserveMessagesVersionsAndEvents(t *testing.T) {
	setupTicketTestDB(t)
	subscribeTicketEvents(t)
	ticket := createTicketFixture(t, 10)
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, content := range []string{"第一条追问", "第二条追问"} {
		go func(content string) {
			<-start
			_, err := ReplyTicket(ticket.Id, 10, false, content)
			results <- err
		}(content)
	}
	close(start)
	first, second := <-results, <-results
	require.NoError(t, first)
	require.NoError(t, second)
	detail, err := GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 3)
	assert.EqualValues(t, 3, detail.Ticket.Version)
	assert.ElementsMatch(t, []string{"第一条追问", "第二条追问"}, []string{detail.Messages[1].Content, detail.Messages[2].Content})
	var events, deliveries int64
	require.NoError(t, DB.Model(&NotificationEvent{}).Count(&events).Error)
	require.NoError(t, DB.Model(&NotificationDelivery{}).Count(&deliveries).Error)
	assert.EqualValues(t, 3, events)
	assert.Equal(t, events, deliveries)
}

func TestTicketConcurrentCloseAndReplyRemainConsistent(t *testing.T) {
	setupTicketTestDB(t)
	subscribeTicketEvents(t)
	ticket := createTicketFixture(t, 10)
	start := make(chan struct{})
	closed, replied := make(chan error, 1), make(chan error, 1)
	go func() {
		<-start
		_, err := ChangeTicketStatus(ticket.Id, 10, false, TicketStatusClosed)
		closed <- err
	}()
	go func() {
		<-start
		_, err := ReplyTicket(ticket.Id, 10, false, "与关闭并发的追问")
		replied <- err
	}()
	close(start)
	closeErr, replyErr := <-closed, <-replied
	require.NoError(t, closeErr)
	// 关闭不再通知：版本计入每次变更，通知只计新建与成功的用户追问。
	messageCount, version, eventCount := 2, 3, int64(2)
	if replyErr != nil {
		require.ErrorIs(t, replyErr, ErrTicketClosed)
		messageCount, version, eventCount = 1, 2, 1
	}
	detail, err := GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	assert.Equal(t, TicketStatusClosed, detail.Ticket.Status)
	assert.Len(t, detail.Messages, messageCount)
	assert.EqualValues(t, version, detail.Ticket.Version)
	var events int64
	require.NoError(t, DB.Model(&NotificationEvent{}).Count(&events).Error)
	assert.Equal(t, eventCount, events)
}

func TestTicketCreationWithoutNotificationMigrationPreservesOptionalSemantics(t *testing.T) {
	setupTicketTestDB(t)
	require.NoError(t, DB.Migrator().DropTable(&NotificationEvent{}))
	ticket := createTicketFixture(t, 10)
	detail, err := GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	require.Len(t, detail.Messages, 1)
}
