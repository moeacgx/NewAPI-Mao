package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptTemplateBindingsRespectOwnershipGroupsAndLiveChanges(t *testing.T) {
	first, second := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&PromptCategory{}, &PromptTemplate{}, &PromptTemplateGroup{}, &TokenPromptBinding{}))
	token := Token{UserId: 77, Key: "prompt-test-key", Name: "提示词令牌"}
	require.NoError(t, DB.Create(&token).Error)
	installationToken, err := GetTokenForPromptInstallation(77, token.Id)
	require.NoError(t, err)
	assert.Empty(t, installationToken.Key)
	prompt, err := SavePromptTemplate(PromptTemplate{Title: "编程助手", Content: "保留客户端上下文", Enabled: true, GroupIDs: []int{first.Id}})
	require.NoError(t, err)
	bindings := []TokenPromptBindingInput{{TemplateID: prompt.ID, Enabled: true, SortOrder: 2}}
	require.Error(t, ReplaceTokenPromptBindings(78, token.Id, bindings, []int{first.Id}))
	require.Error(t, ReplaceTokenPromptBindings(77, token.Id, bindings, []int{second.Id}))
	require.NoError(t, ReplaceTokenPromptBindings(77, token.Id, bindings, []int{first.Id}))
	loaded, err := GetActiveTokenPrompts(token.Id, first.Code)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	assert.Equal(t, prompt.Content, loaded[0].Content)
	loaded, err = GetActiveTokenPrompts(token.Id, second.Code)
	require.NoError(t, err)
	assert.Empty(t, loaded)
	require.NoError(t, DB.Create(&GroupAlias{Alias: "prompt-old-alias", GroupId: first.Id}).Error)
	require.NoError(t, DB.Model(first).Update("name", "已改名分组").Error)
	loaded, err = GetActiveTokenPrompts(token.Id, "prompt-old-alias")
	require.NoError(t, err)
	assert.Len(t, loaded, 1)
	installed, err := GetTokenPromptBindings(77, token.Id)
	require.NoError(t, err)
	assert.Equal(t, "已改名分组", installed[0].Template.GroupDetails[0].Name)
	old := *prompt
	prompt.Content = "新内容"
	prompt, err = SavePromptTemplate(*prompt)
	require.NoError(t, err)
	_, err = SavePromptTemplate(old)
	assert.ErrorIs(t, err, ErrPromptVersionConflict)
	loaded, err = GetActiveTokenPrompts(token.Id, first.Code)
	require.NoError(t, err)
	assert.Equal(t, prompt.Version, loaded[0].Version)
	prompt.Enabled = false
	_, err = SavePromptTemplate(*prompt)
	require.NoError(t, err)
	assert.Error(t, ReplaceTokenPromptBindings(77, token.Id, bindings, []int{first.Id}))
	bindings[0].Enabled = false
	require.NoError(t, ReplaceTokenPromptBindings(77, token.Id, bindings, nil))
	require.NoError(t, HydrateTokenPromptCounts([]*Token{&token}))
	assert.Equal(t, 1, token.PromptCount)
	assert.Zero(t, token.EnabledPromptCount)
	require.NoError(t, ReplaceTokenPromptBindings(77, token.Id, nil, nil))
}

func TestPromptTemplateRejectsInvalidInputAndAggregateSize(t *testing.T) {
	first, _ := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&PromptCategory{}, &PromptTemplate{}, &PromptTemplateGroup{}, &TokenPromptBinding{}))
	valid := PromptTemplate{Title: "提示词", Content: "正文", Enabled: true, GroupIDs: []int{first.Id}}
	for _, invalid := range []PromptTemplate{
		{Title: " ", Content: "正文", GroupIDs: []int{first.Id}},
		{Title: "提示词", Content: strings.Repeat("a", 32769), GroupIDs: []int{first.Id}},
		{Title: strings.Repeat("字", 81), Content: "正文", GroupIDs: []int{first.Id}},
		{Title: "提示词", Content: "正文", GroupIDs: []int{99999}},
		{Title: "提示词", Content: "正文", Enabled: true},
	} {
		_, err := SavePromptTemplate(invalid)
		require.Error(t, err)
	}
	token := Token{UserId: 88, Key: "aggregate-key"}
	require.NoError(t, DB.Create(&token).Error)
	inputs := make([]TokenPromptBindingInput, 0, 3)
	for i := 0; i < 3; i++ {
		valid.Content = strings.Repeat("a", 32768)
		prompt, err := SavePromptTemplate(valid)
		require.NoError(t, err)
		inputs = append(inputs, TokenPromptBindingInput{TemplateID: prompt.ID, Enabled: true})
	}
	require.Error(t, ReplaceTokenPromptBindings(88, token.Id, inputs, []int{first.Id}))
	require.NoError(t, ReplaceTokenPromptBindings(88, token.Id, inputs[:2], []int{first.Id}))
	require.Error(t, ReplaceTokenPromptBindings(88, token.Id, []TokenPromptBindingInput{inputs[0], inputs[0]}, []int{first.Id}))
	require.Error(t, ReplaceTokenPromptBindings(88, token.Id, make([]TokenPromptBindingInput, 9), []int{first.Id}))
}

func TestPromptTemplateUpdateCannotExceedInstalledTokenBudget(t *testing.T) {
	first, _ := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&PromptCategory{}, &PromptTemplate{}, &PromptTemplateGroup{}, &TokenPromptBinding{}))
	token := Token{UserId: 99, Key: "update-budget-key"}
	require.NoError(t, DB.Create(&token).Error)
	inputs := make([]TokenPromptBindingInput, 0, 3)
	var updated *PromptTemplate
	for i := 0; i < 3; i++ {
		prompt, err := SavePromptTemplate(PromptTemplate{Title: "提示词", Content: strings.Repeat("a", 20000), Enabled: true, GroupIDs: []int{first.Id}})
		require.NoError(t, err)
		inputs = append(inputs, TokenPromptBindingInput{TemplateID: prompt.ID, Enabled: true, SortOrder: 3 - i})
		updated = prompt
	}
	require.NoError(t, ReplaceTokenPromptBindings(99, token.Id, inputs, []int{first.Id}))
	updated.Content = strings.Repeat("b", 30000)
	_, err := SavePromptTemplate(*updated)
	require.Error(t, err)
	loaded, err := GetActiveTokenPrompts(token.Id, first.Code)
	require.NoError(t, err)
	require.Len(t, loaded, 3)
	assert.Equal(t, updated.ID, loaded[0].ID)
	assert.Len(t, loaded[0].Content, 20000)
	assert.Equal(t, 1, loaded[0].Version)
	require.NoError(t, HydrateTokenPromptCounts([]*Token{&token}))
	assert.Equal(t, 3, token.PromptCount)
	assert.Equal(t, 3, token.EnabledPromptCount)
	require.NoError(t, DB.Delete(first).Error)
	loaded, err = GetActiveTokenPrompts(token.Id, first.Code)
	require.NoError(t, err)
	assert.Empty(t, loaded)
}

func TestPromptTemplateCanUnpublishAfterAllGroupsDeleted(t *testing.T) {
	group, _ := setupGroupBindingsTest(t)
	require.NoError(t, DB.AutoMigrate(&PromptCategory{}, &PromptTemplate{}, &PromptTemplateGroup{}, &TokenPromptBinding{}))
	prompt, err := SavePromptTemplate(PromptTemplate{Title: "旧模板", Content: "正文", Enabled: true, GroupIDs: []int{group.Id}})
	require.NoError(t, err)
	require.NoError(t, DB.Delete(group).Error)
	prompt.Enabled = false
	prompt.GroupIDs = []int{}
	_, err = SavePromptTemplate(*prompt)
	require.NoError(t, err)
}
