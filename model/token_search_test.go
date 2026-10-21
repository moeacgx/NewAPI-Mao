package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchUserTokensUsesLiteralContainsMatching(t *testing.T) {
	setupGroupBindingsTest(t)
	tokens := []*Token{
		{UserId: 7101, Key: "prefix-secret-suffix", Name: "prefix Gemini Pro suffix"},
		{UserId: 7101, Key: "other-key", Name: "Gemini_Pro"},
		{UserId: 7102, Key: "different-secret", Name: "other user's token"},
	}
	for _, token := range tokens {
		require.NoError(t, DB.Create(token).Error)
	}

	tests := []struct {
		name    string
		keyword string
		secret  string
		want    []string
	}{
		{name: "名称中间片段", keyword: "Gemini Pro", want: []string{"prefix Gemini Pro suffix"}},
		{name: "密钥中间片段", secret: "secret", want: []string{"prefix Gemini Pro suffix"}},
		{name: "下划线按字面量匹配", keyword: "Gemini_Pro", want: []string{"Gemini_Pro"}},
		{name: "无结果", keyword: "not-found", want: []string{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results, total, err := SearchUserTokens(7101, test.keyword, test.secret, 0, 20)
			require.NoError(t, err)
			assert.Equal(t, int64(len(test.want)), total)
			got := make([]string, 0, len(results))
			for _, token := range results {
				got = append(got, token.Name)
			}
			assert.Equal(t, test.want, got)
		})
	}

	pattern, err := sanitizeLikePattern("Gemini_Pro")
	require.NoError(t, err)
	assert.Equal(t, "%Gemini!_Pro%", pattern)
	pattern, err = sanitizeLikePattern("A!B")
	require.NoError(t, err)
	assert.Equal(t, "%A!!B%", pattern)
	pattern, err = sanitizeLikePattern("%secret%")
	require.NoError(t, err)
	assert.Equal(t, "%secret%", pattern)
}

func TestSearchUserTokensContainsMatchingSupportsPagination(t *testing.T) {
	setupGroupBindingsTest(t)
	for _, name := range []string{"matching first", "matching second", "matching third"} {
		require.NoError(t, DB.Create(&Token{UserId: 7201, Key: name, Name: name}).Error)
	}

	results, total, err := SearchUserTokens(7201, "matching", "", 1, 1)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, results, 1)
	assert.Equal(t, "matching second", results[0].Name)
}

func TestSearchUserTokensStatusFiltersBeforePagination(t *testing.T) {
	setupGroupBindingsTest(t)
	for _, token := range []*Token{
		{UserId: 7301, Key: "first-status-key", Name: "demo first", Status: 2},
		{UserId: 7301, Key: "second-status-key", Name: "demo second", Status: 1},
		{UserId: 7301, Key: "third-status-key", Name: "demo third", Status: 2},
		{UserId: 7302, Key: "other-status-key", Name: "demo other", Status: 2},
	} {
		require.NoError(t, DB.Create(token).Error)
	}
	results, total, err := SearchUserTokens(7301, "demo", "", 1, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, results, 1)
	assert.Equal(t, "demo first", results[0].Name)
	results, total, err = SearchUserTokens(7301, "", "", 0, 20, 0)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	assert.Len(t, results, 3)
}

func TestSearchUserTokensByIDsFiltersBeforeCountAndPagination(t *testing.T) {
	setupGroupBindingsTest(t)
	tokens := []*Token{
		{UserId: 7401, Key: "first-needle-key", Name: "demo_foo first", Status: 2},
		{UserId: 7401, Key: "inactive-needle-key", Name: "demo_foo inactive", Status: 2},
		{UserId: 7401, Key: "second-needle-key", Name: "demo_foo second", Status: 2},
		{UserId: 7401, Key: "wrong-name-needle-key", Name: "demoXfoo", Status: 2},
		{UserId: 7401, Key: "wrong-secret-key", Name: "demo_foo other secret", Status: 2},
		{UserId: 7401, Key: "enabled-needle-key", Name: "demo_foo enabled", Status: 1},
		{UserId: 7402, Key: "foreign-needle-key", Name: "demo_foo foreign", Status: 2},
		{UserId: 7401, Key: "deleted-needle-key", Name: "demo_foo deleted", Status: 2},
	}
	for _, token := range tokens {
		require.NoError(t, DB.Create(token).Error)
	}
	require.NoError(t, DB.Delete(tokens[7]).Error)

	ids := []int{tokens[0].Id, tokens[2].Id, tokens[3].Id, tokens[4].Id, tokens[5].Id, tokens[6].Id, tokens[7].Id, tokens[2].Id, 0, -1}
	results, total, err := SearchUserTokensByIDs(7401, "demo_foo", "sk-needle", 1, 1, ids, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(2), total)
	require.Len(t, results, 1)
	assert.Equal(t, tokens[0].Id, results[0].Id)

	results, total, err = SearchUserTokensByIDs(7401, "", "", 0, 20, ids)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	require.Len(t, results, 5)
	assert.Equal(t, []int{tokens[5].Id, tokens[4].Id, tokens[3].Id, tokens[2].Id, tokens[0].Id}, []int{results[0].Id, results[1].Id, results[2].Id, results[3].Id, results[4].Id})
}

func TestSearchUserTokensByIDsDistinguishesUnfilteredAndEmptySets(t *testing.T) {
	setupGroupBindingsTest(t)
	for _, token := range []*Token{
		{UserId: 7501, Key: "own-active-key", Name: "own active", Status: 1},
		{UserId: 7501, Key: "own-disabled-key", Name: "own disabled", Status: 2},
		{UserId: 7502, Key: "foreign-active-key", Name: "foreign active", Status: 1},
	} {
		require.NoError(t, DB.Create(token).Error)
	}

	legacy, legacyTotal, err := SearchUserTokens(7501, "own", "", 0, 20, 1)
	require.NoError(t, err)
	results, total, err := SearchUserTokensByIDs(7501, "own", "", 0, 20, nil, 1)
	require.NoError(t, err)
	assert.Equal(t, legacyTotal, total)
	assert.Equal(t, legacy, results)

	for _, ids := range [][]int{{}, {0, -1}} {
		results, total, err = SearchUserTokensByIDs(7501, "", "", 0, 20, ids)
		require.NoError(t, err)
		assert.Zero(t, total)
		assert.Empty(t, results)
	}

	_, _, err = SearchUserTokensByIDs(7501, "%", "", 0, 20, []int{})
	assert.Error(t, err)
}
