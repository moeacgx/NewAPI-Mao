package model

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTicketArchivedRequiresReopen(t *testing.T) {
	setupTicketTestDB(t)
	ticket := createTicketFixture(t, 10)
	archived, err := ChangeTicketStatus(ticket.Id, 99, true, "archived")
	require.NoError(t, err)
	assert.Equal(t, "archived", archived.Status)
	for _, admin := range []bool{false, true} {
		_, err = ReplyTicket(ticket.Id, 10, admin, "归档期间不能回复")
		assert.ErrorIs(t, err, ErrTicketClosed)
	}
	_, err = ChangeTicketStatus(ticket.Id, 10, false, "archived")
	assert.ErrorIs(t, err, ErrTicketInvalidInput)
	_, err = ChangeTicketStatus(ticket.Id, 10, false, TicketStatusOpen)
	require.NoError(t, err)
	_, err = ReplyTicket(ticket.Id, 10, false, "重新打开后回复")
	require.NoError(t, err)
}

func TestTicketBatchStatusAtomicVersionsAndNotification(t *testing.T) {
	setupTicketTestDB(t)
	subscribeTicketEvents(t)
	first, second := createTicketFixture(t, 10), createTicketFixture(t, 20)
	refs := []TicketVersionRef{{Id: second.Id, Version: 1}, {Id: first.Id, Version: 1}}
	changed, err := ChangeTicketsStatus(99, refs, TicketStatusArchived)
	require.NoError(t, err)
	assert.Equal(t, 2, changed)
	var logs []TicketActionLog
	require.NoError(t, DB.Order("id").Find(&logs).Error)
	require.Len(t, logs, 2)
	assert.Equal(t, 99, logs[0].ActorId)
	assert.Equal(t, "batch_status", logs[0].Action)
	assert.Equal(t, TicketStatusOpen, logs[0].FromStatus)
	assert.Equal(t, TicketStatusArchived, logs[0].ToStatus)
	var events int64
	require.NoError(t, DB.Model(&NotificationEvent{}).Where("event_type = ?", NotificationEventTypeTicketStatusChanged).Count(&events).Error)
	assert.EqualValues(t, 2, events)
	changed, err = ChangeTicketsStatus(99, refs, TicketStatusArchived)
	assert.ErrorIs(t, err, ErrTicketConflict)
	assert.Zero(t, changed)
	refs[0].Version, refs[1].Version = 2, 2
	changed, err = ChangeTicketsStatus(99, refs, TicketStatusArchived)
	require.NoError(t, err)
	assert.Zero(t, changed)
	_, err = ReplyTicket(first.Id, 10, false, "归档不能追加")
	assert.ErrorIs(t, err, ErrTicketClosed)
	_, err = ChangeTicketStatus(first.Id, 10, false, TicketStatusOpen)
	require.NoError(t, err)
	changed, err = ChangeTicketsStatus(99, refs, TicketStatusClosed)
	assert.ErrorIs(t, err, ErrTicketConflict)
	assert.Zero(t, changed)
	detail, err := GetTicketDetail(second.Id, 20, false, 0)
	require.NoError(t, err)
	assert.Equal(t, TicketStatusArchived, detail.Ticket.Status)
}

func TestTicketBatchInputRejectsDuplicatesMissingAndOversize(t *testing.T) {
	setupTicketTestDB(t)
	ticket := createTicketFixture(t, 10)
	ref := TicketVersionRef{Id: ticket.Id, Version: 1}
	cases := []struct {
		name string
		refs []TicketVersionRef
		want error
	}{
		{"empty", nil, ErrTicketInvalidInput},
		{"duplicate", []TicketVersionRef{ref, ref}, ErrTicketInvalidInput},
		{"id", []TicketVersionRef{{Id: 0, Version: 1}}, ErrTicketInvalidInput},
		{"version", []TicketVersionRef{{Id: ticket.Id, Version: 0}}, ErrTicketInvalidInput},
		{"missing", []TicketVersionRef{ref, {Id: 99999, Version: 1}}, gorm.ErrRecordNotFound},
		{"limit", make([]TicketVersionRef, 101), ErrTicketInvalidInput},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			count, err := ChangeTicketsStatus(99, tc.refs, TicketStatusClosed)
			assert.Zero(t, count)
			assert.ErrorIs(t, err, tc.want)
			count, err = DeleteTickets(99, tc.refs)
			assert.Zero(t, count)
			assert.ErrorIs(t, err, tc.want)
		})
	}
	_, err := ChangeTicketsStatus(99, []TicketVersionRef{ref}, "unknown")
	assert.ErrorIs(t, err, ErrTicketInvalidInput)
	detail, err := GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 1, detail.Ticket.Version)
}

func TestTicketSoftDeletePreservesMessagesAndBlocksAllAccess(t *testing.T) {
	setupTicketTestDB(t)
	ticket := createTicketFixture(t, 10)
	count, err := DeleteTickets(99, []TicketVersionRef{{Id: ticket.Id, Version: 1}})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	for _, admin := range []bool{false, true} {
		items, total, err := ListTickets(10, admin, TicketFilter{}, &common.PageInfo{Page: 1, PageSize: 20})
		require.NoError(t, err)
		assert.Empty(t, items)
		assert.Zero(t, total)
		_, err = GetTicketDetail(ticket.Id, 10, admin, 0)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = ReplyTicket(ticket.Id, 10, admin, "删除后不能追问")
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		_, err = ChangeTicketStatus(ticket.Id, 10, admin, TicketStatusOpen)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
	}
	var stored Ticket
	require.NoError(t, DB.Unscoped().First(&stored, ticket.Id).Error)
	assert.True(t, stored.DeletedAt.Valid)
	var messages int64
	require.NoError(t, DB.Model(&TicketMessage{}).Where("ticket_id = ?", ticket.Id).Count(&messages).Error)
	assert.EqualValues(t, 1, messages)
	var audit TicketActionLog
	require.NoError(t, DB.Where("ticket_id = ?", ticket.Id).First(&audit).Error)
	assert.Equal(t, "delete", audit.Action)
	assert.Equal(t, 99, audit.ActorId)
}

func TestTicketBatchRollsBackOnAuditOrNotificationFailure(t *testing.T) {
	for _, failureTable := range []string{"ticket_action_logs", "notification_deliveries"} {
		t.Run(failureTable, func(t *testing.T) {
			setupTicketTestDB(t)
			subscribeTicketEvents(t)
			first, second := createTicketFixture(t, 10), createTicketFixture(t, 20)
			fixtureErr := errors.New("simulated storage failure")
			n := 0
			require.NoError(t, DB.Callback().Create().Before("gorm:create").Register("test:bulk_failure", func(tx *gorm.DB) {
				if tx.Statement.Table == failureTable {
					n++
					if n == 2 {
						tx.AddError(fixtureErr)
					}
				}
			}))
			t.Cleanup(func() { _ = DB.Callback().Create().Remove("test:bulk_failure") })
			refs := []TicketVersionRef{{Id: first.Id, Version: 1}, {Id: second.Id, Version: 1}}
			count, err := ChangeTicketsStatus(99, refs, TicketStatusClosed)
			assert.Zero(t, count)
			assert.ErrorIs(t, err, fixtureErr)
			for _, id := range []int{first.Id, second.Id} {
				var got Ticket
				require.NoError(t, DB.First(&got, id).Error)
				assert.Equal(t, TicketStatusOpen, got.Status)
				assert.EqualValues(t, 1, got.Version)
			}
			var logs int64
			require.NoError(t, DB.Model(&TicketActionLog{}).Count(&logs).Error)
			assert.Zero(t, logs)
			if failureTable == "ticket_action_logs" {
				n = 0
				count, err = DeleteTickets(99, refs)
				assert.Zero(t, count)
				assert.ErrorIs(t, err, fixtureErr)
				var remaining int64
				require.NoError(t, DB.Model(&Ticket{}).Count(&remaining).Error)
				assert.EqualValues(t, 2, remaining)
			}
		})
	}
}

func TestTicketAutoArchiveConfigValidationAndSaveFailure(t *testing.T) {
	setupTicketTestDB(t)
	defaults, err := GetTicketAutoArchiveConfig()
	require.NoError(t, err)
	assert.False(t, defaults.Enabled)
	assert.Equal(t, 30, defaults.InactiveDays)
	assert.ElementsMatch(t, []string{TicketStatusResolved, TicketStatusClosed}, defaults.Statuses)
	for _, input := range []TicketAutoArchiveConfig{
		{InactiveDays: 0, Statuses: []string{"open"}}, {InactiveDays: 3651, Statuses: []string{"open"}},
		{InactiveDays: 30}, {InactiveDays: 30, Statuses: []string{"archived"}},
		{InactiveDays: 30, Statuses: []string{"open", "open"}},
	} {
		_, err = UpdateTicketAutoArchiveConfig(input)
		assert.ErrorIs(t, err, ErrTicketInvalidInput)
	}
	config := TicketAutoArchiveConfig{Enabled: true, InactiveDays: 7, Statuses: []string{TicketStatusOpen}}
	_, err = UpdateTicketAutoArchiveConfig(config)
	require.NoError(t, err)
	require.NoError(t, DB.Callback().Update().Before("gorm:update").Register("test:archive_config_save", func(tx *gorm.DB) {
		if tx.Statement.Table == "options" {
			tx.AddError(errors.New("write failed"))
		}
	}))
	t.Cleanup(func() { _ = DB.Callback().Update().Remove("test:archive_config_save") })
	_, err = UpdateTicketAutoArchiveConfig(defaults)
	require.Error(t, err)
	actual, err := GetTicketAutoArchiveConfig()
	require.NoError(t, err)
	assert.Equal(t, config, actual)
}

func TestTicketAutoArchiveCutoffSelectionAndExactlyOnce(t *testing.T) {
	setupTicketTestDB(t)
	subscribeTicketEvents(t)
	now := nowUnix()
	cutoff := now - 7*86400
	config := TicketAutoArchiveConfig{Enabled: true, InactiveDays: 7, Statuses: []string{TicketStatusOpen}}
	_, err := UpdateTicketAutoArchiveConfig(config)
	require.NoError(t, err)
	old, boundary, recent, closed, deleted := createTicketFixture(t, 10), createTicketFixture(t, 10), createTicketFixture(t, 10), createTicketFixture(t, 10), createTicketFixture(t, 10)
	for _, ticket := range []*Ticket{old, boundary, recent, closed, deleted} {
		require.NoError(t, DB.Model(&Ticket{}).Where("id = ?", ticket.Id).Update("updated_at", cutoff-1).Error)
	}
	require.NoError(t, DB.Model(&Ticket{}).Where("id = ?", boundary.Id).Update("updated_at", cutoff).Error)
	require.NoError(t, DB.Model(&Ticket{}).Where("id = ?", recent.Id).Update("updated_at", cutoff+1).Error)
	_, err = ChangeTicketStatus(closed.Id, 99, true, TicketStatusClosed)
	require.NoError(t, err)
	_, err = DeleteTickets(99, []TicketVersionRef{{Id: deleted.Id, Version: 1}})
	require.NoError(t, err)
	count, err := ArchiveInactiveTickets(context.Background(), now, 1)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	count, err = ArchiveInactiveTickets(context.Background(), now, 100)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	count, err = ArchiveInactiveTickets(context.Background(), now, 100)
	require.NoError(t, err)
	assert.Zero(t, count)
	var logs []TicketActionLog
	require.NoError(t, DB.Where("action = ?", "auto_archive").Find(&logs).Error)
	require.Len(t, logs, 2)
	assert.Zero(t, logs[0].ActorId)
	var events []NotificationEvent
	require.NoError(t, DB.Where("event_type = ?", NotificationEventTypeTicketStatusChanged).Find(&events).Error)
	found := 0
	for _, event := range events {
		if strings.Contains(event.Payload, `"actor_role":"system"`) {
			found++
		}
	}
	assert.Equal(t, 2, found)
	for _, ticket := range []*Ticket{old, boundary} {
		detail, err := GetTicketDetail(ticket.Id, 10, false, 0)
		require.NoError(t, err)
		assert.Equal(t, TicketStatusArchived, detail.Ticket.Status)
	}
}

func TestTicketAutoArchiveRechecksReplyAndDisabledConfiguration(t *testing.T) {
	for _, change := range []string{"reply", "disable", "delete"} {
		t.Run(change, func(t *testing.T) {
			setupTicketTestDB(t)
			config := TicketAutoArchiveConfig{Enabled: true, InactiveDays: 1, Statuses: []string{TicketStatusOpen}}
			_, err := UpdateTicketAutoArchiveConfig(config)
			require.NoError(t, err)
			ticket := createTicketFixture(t, 10)
			now := nowUnix()
			require.NoError(t, DB.Model(&Ticket{}).Where("id = ?", ticket.Id).Update("updated_at", now-86401).Error)
			// 在候选扫描完成后执行确定性竞争操作，不依赖 sleep 或线程时序。
			fired := false
			require.NoError(t, DB.Callback().Query().After("gorm:after_query").Register("test:archive_recheck", func(tx *gorm.DB) {
				_, isCandidateScan := tx.Statement.Dest.(*[]int)
				if fired || tx.Statement.Table != "tickets" || !isCandidateScan {
					return
				}
				fired = true
				switch change {
				case "reply":
					_, err = ReplyTicket(ticket.Id, 10, false, "扫描期间的新回复")
				case "disable":
					config.Enabled = false
					_, err = UpdateTicketAutoArchiveConfig(config)
				case "delete":
					_, err = DeleteTickets(99, []TicketVersionRef{{Id: ticket.Id, Version: 1}})
				}
				require.NoError(t, err)
			}))
			t.Cleanup(func() { _ = DB.Callback().Query().Remove("test:archive_recheck") })
			count, err := ArchiveInactiveTickets(context.Background(), now, 100)
			require.NoError(t, err)
			assert.True(t, fired)
			assert.Zero(t, count)
		})
	}
}

func TestTicketAutoArchiveNotificationFailureRollsBack(t *testing.T) {
	setupTicketTestDB(t)
	subscribeTicketEvents(t)
	_, err := UpdateTicketAutoArchiveConfig(TicketAutoArchiveConfig{Enabled: true, InactiveDays: 1, Statuses: []string{TicketStatusOpen}})
	require.NoError(t, err)
	ticket := createTicketFixture(t, 10)
	now := nowUnix()
	require.NoError(t, DB.Model(&Ticket{}).Where("id = ?", ticket.Id).Update("updated_at", now-86401).Error)
	require.NoError(t, DB.Callback().Create().Before("gorm:create").Register("test:archive_notify_fail", func(tx *gorm.DB) {
		if tx.Statement.Table == "notification_deliveries" {
			tx.AddError(errors.New("notification failed"))
		}
	}))
	t.Cleanup(func() { _ = DB.Callback().Create().Remove("test:archive_notify_fail") })
	count, err := ArchiveInactiveTickets(context.Background(), now, 100)
	require.Error(t, err)
	assert.Zero(t, count)
	detail, err := GetTicketDetail(ticket.Id, 10, false, 0)
	require.NoError(t, err)
	assert.Equal(t, TicketStatusOpen, detail.Ticket.Status)
	var logs int64
	require.NoError(t, DB.Model(&TicketActionLog{}).Count(&logs).Error)
	assert.Zero(t, logs)
}
