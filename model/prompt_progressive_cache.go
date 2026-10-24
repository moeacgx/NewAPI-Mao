package model

import (
	"container/list"
	"strings"
	"sync"

	"gorm.io/gorm"
)

const progressiveSkillCacheBytes = 32 * 1024 * 1024

// 正文页和目录共用预算，固定开销覆盖索引、链表及字段，空正文也不能无限占用条目。
const progressiveSkillCacheEntryBytes = 512

type progressiveSkillCacheKey struct {
	TemplateID int
	Version    int
	SkillID    string
	Path       string
	Offset     int
}

type progressiveSkillCacheEntry struct {
	key   progressiveSkillCacheKey
	page  SkillFilePage
	bytes int
}

// 空路径键只存版本化资源目录；文件页不重复缓存整份目录。
type progressiveSkillContentCache struct {
	mu       sync.Mutex
	database *gorm.DB
	capacity int
	used     int
	entries  map[progressiveSkillCacheKey]*list.Element
	order    list.List
}

var progressiveSkillContents = newProgressiveSkillContentCache(progressiveSkillCacheBytes)

func newProgressiveSkillContentCache(capacity int) *progressiveSkillContentCache {
	return &progressiveSkillContentCache{
		capacity: capacity,
		entries:  make(map[progressiveSkillCacheKey]*list.Element),
	}
}

// ClearProgressiveSkillContentCache 清理内容缓存，不影响实时权限检查和数据库内容。
func ClearProgressiveSkillContentCache() {
	progressiveSkillContents.clear()
}

func (cache *progressiveSkillContentCache) clear() {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.reset(nil)
}

// 调用方持有缓存锁；更换数据库连接时不能复用同 ID、同版本的其他数据库内容。
func (cache *progressiveSkillContentCache) reset(database *gorm.DB) {
	cache.database = database
	cache.used = 0
	cache.entries = make(map[progressiveSkillCacheKey]*list.Element)
	cache.order.Init()
}

func (cache *progressiveSkillContentCache) get(database *gorm.DB, key progressiveSkillCacheKey) (SkillFilePage, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.database != database {
		cache.reset(database)
	}
	element, ok := cache.entries[key]
	if !ok {
		return SkillFilePage{}, false
	}
	cache.order.MoveToFront(element)
	page := element.Value.(*progressiveSkillCacheEntry).page
	if page.Resources != nil {
		page.Resources = append([]string{}, page.Resources...)
	}
	return page, true
}

func (cache *progressiveSkillContentCache) put(database *gorm.DB, key progressiveSkillCacheKey, page SkillFilePage) {
	bytes := progressiveSkillCacheEntryBytes + len(key.SkillID) + len(key.Path) + len(page.Content) + 16*len(page.Resources)
	for _, path := range page.Resources {
		bytes += len(path)
	}
	if bytes > cache.capacity || cache.capacity <= 0 {
		return
	}
	// 页正文可能引用整个分块，工具参数也可能引用请求正文；必须切断这些大对象引用。
	key.SkillID, key.Path = strings.Clone(key.SkillID), strings.Clone(key.Path)
	page.Content = strings.Clone(page.Content)
	if page.Resources != nil {
		resources := make([]string, len(page.Resources))
		for i, path := range page.Resources {
			resources[i] = strings.Clone(path)
		}
		page.Resources = resources
	}

	cache.mu.Lock()
	defer cache.mu.Unlock()
	if cache.database != database {
		cache.reset(database)
	}
	if element, ok := cache.entries[key]; ok {
		cache.used -= element.Value.(*progressiveSkillCacheEntry).bytes
		cache.order.Remove(element)
		delete(cache.entries, key)
	}
	for cache.used+bytes > cache.capacity {
		oldest := cache.order.Back()
		entry := oldest.Value.(*progressiveSkillCacheEntry)
		delete(cache.entries, entry.key)
		cache.used -= entry.bytes
		cache.order.Remove(oldest)
	}
	entry := &progressiveSkillCacheEntry{key: key, page: page, bytes: bytes}
	cache.entries[key] = cache.order.PushFront(entry)
	cache.used += bytes
}
