package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTicketAutoReplyConfigDefaultsAndValidation(t *testing.T) {
	setupTicketTestDB(t)
	require.NoError(t, DB.AutoMigrate(&Option{}))

	config, err := GetTicketAutoReplyConfig()
	require.NoError(t, err)
	assert.False(t, config.Enabled)
	assert.Empty(t, config.Content)

	_, err = UpdateTicketAutoReplyConfig(TicketAutoReplyConfig{Enabled: true})
	assert.ErrorIs(t, err, ErrTicketInvalidInput)
	_, err = UpdateTicketAutoReplyConfig(TicketAutoReplyConfig{Content: strings.Repeat("工", TicketContentMaxRunes+1)})
	assert.ErrorIs(t, err, ErrTicketInvalidInput)

	updated, err := UpdateTicketAutoReplyConfig(TicketAutoReplyConfig{Enabled: true, Content: "  已收到，我们会尽快处理。  "})
	require.NoError(t, err)
	assert.Equal(t, "已收到，我们会尽快处理。", updated.Content)
	loaded, err := GetTicketAutoReplyConfig()
	require.NoError(t, err)
	assert.Equal(t, updated, loaded)
}

func TestCreateTicketAddsOnlyOneConfiguredAutoReply(t *testing.T) {
	setupTicketTestDB(t)
	require.NoError(t, DB.AutoMigrate(&Option{}))
	_, err := UpdateTicketAutoReplyConfig(TicketAutoReplyConfig{Enabled: true, Content: "自动回复：已收到你的工单。"})
	require.NoError(t, err)

	ticket, err := CreateTicket(1001, TicketCreateInput{Title: "需要帮助", Category: "general", Priority: "normal", Content: "请协助处理"})
	require.NoError(t, err)
	var messages []TicketMessage
	require.NoError(t, DB.Where("ticket_id = ?", ticket.Id).Order("id ASC").Find(&messages).Error)
	require.Len(t, messages, 2)
	assert.Equal(t, "user", messages[0].AuthorRole)
	assert.Equal(t, "admin", messages[1].AuthorRole)
	assert.Zero(t, messages[1].AuthorId)
	assert.Equal(t, "自动回复：已收到你的工单。", messages[1].Content)

	_, err = UpdateTicketAutoReplyConfig(TicketAutoReplyConfig{Enabled: false, Content: "保留但不发送"})
	require.NoError(t, err)
	second, err := CreateTicket(1001, TicketCreateInput{Title: "第二个工单", Category: "general", Priority: "normal", Content: "另一条内容"})
	require.NoError(t, err)
	var count int64
	require.NoError(t, DB.Model(&TicketMessage{}).Where("ticket_id = ? AND author_role = ?", second.Id, "admin").Count(&count).Error)
	assert.Zero(t, count)
}
