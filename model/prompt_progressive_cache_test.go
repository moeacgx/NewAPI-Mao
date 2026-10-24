package model

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestProgressiveSkillCacheBoundsBytesAndEvictsLeastRecentlyUsed(t *testing.T) {
	database := &gorm.DB{}
	page := SkillFilePage{Content: "正文", TotalBytes: 6, NextOffset: 6, EOF: true}
	entryBytes := progressiveSkillCacheEntryBytes + len("skill") + len("a.md") + len(page.Content)
	cache := newProgressiveSkillContentCache(2 * entryBytes)
	first := progressiveSkillCacheKey{TemplateID: 1, Version: 1, SkillID: "skill", Path: "a.md"}
	second, third := first, first
	second.Path, third.Path = "b.md", "c.md"
	cache.put(database, first, page)
	cache.put(database, second, page)
	_, found := cache.get(database, first)
	require.True(t, found)
	cache.put(database, third, page)
	_, found = cache.get(database, second)
	assert.False(t, found, "容量不足时淘汰最久未使用的正文页")
	for _, key := range []progressiveSkillCacheKey{first, third} {
		got, found := cache.get(database, key)
		require.True(t, found)
		assert.Equal(t, page, got)
	}
	assert.LessOrEqual(t, cache.used, cache.capacity)

	oversized := SkillFilePage{Resources: []string{strings.Repeat("x", cache.capacity)}}
	cache.put(database, progressiveSkillCacheKey{TemplateID: 2, Version: 1, SkillID: "skill"}, oversized)
	_, found = cache.get(database, first)
	assert.True(t, found, "单个超大条目不入缓存，也不驱逐已有正文")
	assert.LessOrEqual(t, cache.used, cache.capacity)
	cache.put(database, first, SkillFilePage{Content: strings.Repeat("x", entryBytes)})
	assert.LessOrEqual(t, cache.used, cache.capacity, "替换条目也必须重新计算容量")
	_, found = cache.get(database, third)
	assert.False(t, found)
}

func TestProgressiveSkillCacheIsolatesVersionDatabaseAndConcurrentAccess(t *testing.T) {
	database := &gorm.DB{}
	cache := newProgressiveSkillContentCache(16 * 1024)
	key := progressiveSkillCacheKey{TemplateID: 1, Version: 1, SkillID: "skill"}
	original := SkillFilePage{Resources: []string{"references/规范.md"}}
	cache.put(database, key, original)
	original.Resources[0] = "调用方改写"
	otherVersion := key
	otherVersion.Version = 2
	cache.put(database, otherVersion, SkillFilePage{Resources: []string{"references/新版.md"}})

	results := make(chan []string, 2)
	start := make(chan struct{})
	for _, readKey := range []progressiveSkillCacheKey{key, otherVersion} {
		go func(readKey progressiveSkillCacheKey) {
			<-start
			page, found := cache.get(database, readKey)
			if !found {
				results <- nil
				return
			}
			paths := append([]string{}, page.Resources...)
			cache.put(database, readKey, page)
			page.Resources[0] = "读者改写"
			results <- paths
		}(readKey)
	}
	close(start)
	assert.ElementsMatch(t, [][]string{{"references/规范.md"}, {"references/新版.md"}}, [][]string{<-results, <-results})
	page, found := cache.get(database, key)
	require.True(t, found)
	assert.Equal(t, []string{"references/规范.md"}, page.Resources)
	_, found = cache.get(&gorm.DB{}, key)
	assert.False(t, found, "不同数据库相同 ID 与版本不能复用缓存")
	cache.put(database, key, page)
	cache.clear()
	_, found = cache.get(database, key)
	assert.False(t, found)
	assert.Zero(t, cache.used)
}

func TestProgressiveSkillCacheReusesFilePagesAndDirectoriesAfterLiveChecks(t *testing.T) {
	group, _, token := setupProgressiveSkillTest(t)
	body := strings.Repeat("中文。", 9000)
	pack, err := SavePromptTemplate(PromptTemplate{Title: "缓存复核", Description: "按需读取", Content: body, LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{group.Id}, Skills: []PromptSkill{{Resources: []PromptSkillResource{{Path: "references/规范.md", Content: "引用"}}}}})
	require.NoError(t, err)
	bindings := []TokenPromptBindingInput{{TemplateID: pack.ID, Enabled: true}}
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, bindings, []int{group.Id}))

	var queries []string
	directoryReads := 0
	require.NoError(t, DB.Callback().Query().After("gorm:query").Register("test_progressive_cache_queries", func(tx *gorm.DB) {
		queries = append(queries, tx.Statement.SQL.String())
		if _, directory := tx.Statement.Dest.(*[]string); directory && tx.Statement.Table == "prompt_skill_files" {
			directoryReads++
		}
	}))
	t.Cleanup(func() { _ = DB.Callback().Query().Remove("test_progressive_cache_queries") })
	page, err := ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "", 0, MaxSkillReadBytes)
	require.NoError(t, err)
	assert.Contains(t, strings.Join(queries, "\n"), "prompt_skill_chunks")
	queries = nil
	verified, err := ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "SKILL.md", 0, len(page.Content))
	require.NoError(t, err)
	assert.Equal(t, page, verified)
	warmSQL := strings.Join(queries, "\n")
	assert.NotContains(t, warmSQL, "prompt_skill_chunks", "正式生成以实际页长复核时应复用正文")
	assert.NotContains(t, warmSQL, "prompt_skill_files", "命中后不再扫描文件或目录")
	for _, table := range []string{"tokens", "prompt_templates", "token_prompt_bindings", "groups"} {
		assert.Contains(t, warmSQL, table, "缓存不能跳过实时状态与授权检查")
	}
	page.Resources[0] = "调用方改写"
	queries = nil
	short, err := ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "", 0, 5)
	require.NoError(t, err)
	assert.Equal(t, "中", short.Content)
	assert.Equal(t, 3, short.NextOffset)
	assert.False(t, short.EOF)
	assert.Equal(t, []string{"references/规范.md"}, short.Resources)
	assert.NotContains(t, strings.Join(queries, "\n"), "prompt_skill_chunks")
	_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "", 0, 1)
	require.ErrorContains(t, err, "UTF-8")

	otherToken := &Token{UserId: token.UserId + 1, Key: "progressive-cache-other", Group: group.Code}
	require.NoError(t, DB.Create(otherToken).Error)
	_, err = ReadActiveTokenSkillFile(otherToken.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "", 0, MaxSkillReadBytes)
	require.ErrorContains(t, err, "无权")
	require.NoError(t, ReplaceTokenPromptBindings(otherToken.UserId, otherToken.Id, bindings, []int{group.Id}))
	queries = nil
	shared, err := ReadActiveTokenSkillFile(otherToken.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "", 0, MaxSkillReadBytes)
	require.NoError(t, err)
	assert.Equal(t, verified, shared)
	assert.NotContains(t, strings.Join(queries, "\n"), "prompt_skill_chunks", "另一授权令牌可复用相同版本正文")

	queries = nil
	directoryReads = 0
	_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "", shared.NextOffset, MaxSkillReadBytes)
	require.NoError(t, err)
	assert.Zero(t, directoryReads, "翻页不重复读取完整资源目录")

	previousVersion := pack.Version
	pack.Content = "更新后的正文"
	pack.Skills[0].Resources = []PromptSkillResource{{Path: "references/新版.md", Content: "新引用"}}
	pack, err = SavePromptTemplate(*pack)
	require.NoError(t, err)
	_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, pack.Skills[0].ID, previousVersion, "", 0, MaxSkillReadBytes)
	require.ErrorContains(t, err, "版本")
	updated, err := ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "", 0, MaxSkillReadBytes)
	require.NoError(t, err)
	assert.Equal(t, pack.Content, updated.Content)
	assert.Equal(t, []string{"references/新版.md"}, updated.Resources)
}

func TestProgressiveSkillCacheDoesNotPublishFailedReadTransaction(t *testing.T) {
	group, _, token := setupProgressiveSkillTest(t)
	pack, err := SavePromptTemplate(PromptTemplate{Title: "失败回滚", Description: "按需读取", Content: "正文", LoadMode: PromptLoadModeProgressive, Enabled: true, GroupIDs: []int{group.Id}})
	require.NoError(t, err)
	require.NoError(t, ReplaceTokenPromptBindings(token.UserId, token.Id, []TokenPromptBindingInput{{TemplateID: pack.ID, Enabled: true}}, []int{group.Id}))
	failDirectory := true
	chunkReads := 0
	require.NoError(t, DB.Callback().Query().After("gorm:query").Register("test_progressive_cache_failure", func(tx *gorm.DB) {
		if strings.Contains(tx.Statement.SQL.String(), "prompt_skill_chunks") {
			chunkReads++
		}
		if _, directory := tx.Statement.Dest.(*[]string); directory && tx.Statement.Table == "prompt_skill_files" && failDirectory {
			tx.AddError(errors.New("模拟目录读取失败"))
		}
	}))
	t.Cleanup(func() { _ = DB.Callback().Query().Remove("test_progressive_cache_failure") })
	_, err = ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "", 0, MaxSkillReadBytes)
	require.ErrorContains(t, err, "模拟目录读取失败")
	assert.Equal(t, 1, chunkReads)
	failDirectory = false
	page, err := ReadActiveTokenSkillFile(token.Id, group.Code, pack.ID, pack.Skills[0].ID, pack.Version, "", 0, MaxSkillReadBytes)
	require.NoError(t, err)
	assert.Equal(t, "正文", page.Content)
	assert.Equal(t, 2, chunkReads, "事务回滚后不能命中失败读取的正文")
}
