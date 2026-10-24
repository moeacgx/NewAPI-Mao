package service

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTicketArchiveServiceDB(t *testing.T) {
	t.Helper()
	originalDB, originalType := model.DB, common.MainDatabaseType()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "ticket-archive.db")), &gorm.Config{})
	require.NoError(t, err)
	connection, err := db.DB()
	require.NoError(t, err)
	connection.SetMaxOpenConns(1)
	model.DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	model.InitDBColumns()
	t.Cleanup(func() {
		require.NoError(t, connection.Close())
		model.DB = originalDB
		common.SetMainDatabaseType(originalType)
		model.InitDBColumns()
	})
	require.NoError(t, db.AutoMigrate(&model.Option{}, &model.Ticket{}, &model.TicketActionLog{}, &model.SystemTask{}, &model.SystemTaskLock{}))
	config, err := common.Marshal(model.TicketAutoArchiveConfig{Enabled: true, InactiveDays: 30, Statuses: []string{model.TicketStatusResolved}})
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.Option{Key: model.TicketAutoArchiveOptionKey, Value: string(config)}).Error)
}

func claimTicketArchiveTask(t *testing.T) *model.SystemTask {
	t.Helper()
	task, err := model.CreateSystemTask(SystemTaskTypeTicketAutoArchive, nil, nil)
	require.NoError(t, err)
	task, claimed, err := model.ClaimSystemTask(task.ID, task.Type, "ticket-archive-test", time.Now().Add(time.Minute).Unix())
	require.NoError(t, err)
	require.True(t, claimed)
	return task
}

func TestTicketArchiveTaskRegistersAndLimitsEachPass(t *testing.T) {
	setupTicketArchiveServiceDB(t)
	withSystemTaskRegistry(t)
	RegisterTicketArchiveSystemTask()
	handlers := registeredSystemTaskHandlers()
	require.Len(t, handlers, 1)
	handler, ok := handlers[0].(ScheduledSystemTaskHandler)
	require.True(t, ok)
	require.True(t, handler.Enabled())
	old := time.Now().Add(-31 * 24 * time.Hour).Unix()
	tickets := make([]model.Ticket, 101)
	for i := range tickets {
		tickets[i] = model.Ticket{UserId: 1, Title: "等待归档", Category: "general", Priority: "normal", Status: model.TicketStatusResolved, Version: 1, CreatedAt: old, UpdatedAt: old}
	}
	require.NoError(t, model.DB.Create(&tickets).Error)
	task := claimTicketArchiveTask(t)
	handler.Run(context.Background(), task, "ticket-archive-test")
	saved, err := model.GetSystemTaskByTaskID(task.TaskID)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, model.SystemTaskStatusSucceeded, saved.Status)
	assert.Empty(t, saved.Error)
	assert.JSONEq(t, `{"archived_count":100}`, saved.Result)
	assert.Nil(t, saved.ActiveKey)
	var archived, remaining int64
	require.NoError(t, model.DB.Model(&model.Ticket{}).Where("status = ?", model.TicketStatusArchived).Count(&archived).Error)
	require.NoError(t, model.DB.Model(&model.Ticket{}).Where("status = ?", model.TicketStatusResolved).Count(&remaining).Error)
	assert.EqualValues(t, 100, archived)
	assert.EqualValues(t, 1, remaining)
}

func TestTicketArchiveTaskHonorsDisabledConfigAfterQueueing(t *testing.T) {
	setupTicketArchiveServiceDB(t)
	handler := ticketArchiveHandler{}
	require.True(t, handler.Enabled())
	task := claimTicketArchiveTask(t)
	old := time.Now().Add(-31 * 24 * time.Hour).Unix()
	ticket := &model.Ticket{UserId: 1, Title: "保留工单", Category: "general", Priority: "normal", Status: model.TicketStatusResolved, Version: 1, CreatedAt: old, UpdatedAt: old}
	require.NoError(t, model.DB.Create(ticket).Error)
	require.NoError(t, model.DB.Model(&model.Option{}).Where(&model.Option{Key: model.TicketAutoArchiveOptionKey}).Update("value", `{"enabled":false,"inactive_days":30,"statuses":["resolved"]}`).Error)
	assert.False(t, handler.Enabled())
	handler.Run(context.Background(), task, "ticket-archive-test")
	saved, err := model.GetSystemTaskByTaskID(task.TaskID)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, model.SystemTaskStatusSucceeded, saved.Status)
	assert.JSONEq(t, `{"archived_count":0}`, saved.Result)
	require.NoError(t, model.DB.First(ticket, ticket.Id).Error)
	assert.Equal(t, model.TicketStatusResolved, ticket.Status)
}

func TestTicketArchiveTaskRedactsDatabaseFailureAndFinishes(t *testing.T) {
	setupTicketArchiveServiceDB(t)
	task := claimTicketArchiveTask(t)
	const privateError = "private ticket content must not appear in task history"
	require.NoError(t, model.DB.Callback().Query().Before("gorm:query").Register("test:ticket-archive-query", func(tx *gorm.DB) {
		if tx.Statement.Table == "tickets" {
			tx.AddError(errors.New(privateError))
		}
	}))
	t.Cleanup(func() { _ = model.DB.Callback().Query().Remove("test:ticket-archive-query") })
	ticketArchiveHandler{}.Run(context.Background(), task, "ticket-archive-test")
	saved, err := model.GetSystemTaskByTaskID(task.TaskID)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, model.SystemTaskStatusFailed, saved.Status)
	assert.Equal(t, "ticket auto-archive failed", saved.Error)
	assert.JSONEq(t, `{"archived_count":0}`, saved.Result)
	assert.Nil(t, saved.ActiveKey)
	assert.NotContains(t, saved.Error+saved.Result, privateError)
}

func TestTicketArchiveTaskFinishesCancelledPass(t *testing.T) {
	setupTicketArchiveServiceDB(t)
	task := claimTicketArchiveTask(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	ticketArchiveHandler{}.Run(ctx, task, "ticket-archive-test")
	saved, err := model.GetSystemTaskByTaskID(task.TaskID)
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, model.SystemTaskStatusFailed, saved.Status)
	assert.Equal(t, "ticket auto-archive failed", saved.Error)
	assert.JSONEq(t, `{"archived_count":0}`, saved.Result)
	assert.Nil(t, saved.ActiveKey)
}
