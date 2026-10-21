// Package activerequests 维护短期请求状态，不参与鉴权、路由或计费。
package activerequests

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Entry struct {
	ID              string `json:"id"`
	RequestID       string `json:"request_id"`
	TokenID         int    `json:"token_id"`
	TokenName       string `json:"token_name"`
	ModelName       string `json:"model_name"`
	RequestType     string `json:"request_type"`
	IsStream        bool   `json:"is_stream"`
	StartedAt       int64  `json:"started_at"`
	FirstResponseMS *int64 `json:"first_response_ms"`
	Status          string `json:"status"`
	RetryCount      int    `json:"retry_count"`
}

type Summary struct {
	Total        int            `json:"total"`
	Streaming    int            `json:"streaming"`
	NonStreaming int            `json:"non_streaming"`
	Waiting      int            `json:"waiting"`
	ActiveTokens int            `json:"active_tokens"`
	ByToken      map[string]int `json:"by_token"`
}

type Query struct {
	Page, PageSize int
	Search, Status string
	TokenID        int
}

type Page struct {
	Items      []Entry `json:"items"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PageSize   int     `json:"page_size"`
	Summary    Summary `json:"summary"`
	Scope      string  `json:"scope"`
	ObservedAt int64   `json:"observed_at"`
}

// Record 的用户身份只用于存储与隔离，不属于公开响应。
type Record struct {
	UserID int   `json:"user_id"`
	Entry  Entry `json:"entry"`
}

type Removal struct {
	UserID int
	ID     string
	At     time.Time
}

type Store interface {
	Write(context.Context, []Record, []Removal, time.Time) error
	Read(context.Context, int, time.Time) ([]Entry, error)
}

type Registry struct {
	mu         sync.Mutex
	entries    map[string]Record
	removals   map[string]Removal
	store      Store
	publishMu  sync.Mutex
	publishErr error
}

func NewRegistry(store Store) *Registry {
	return &Registry{entries: make(map[string]Record), removals: make(map[string]Removal), store: store}
}

type Handle struct {
	registry *Registry
	id       string
}

func (r *Registry) Start(userID int, entry Entry) *Handle {
	if userID <= 0 {
		return nil
	}
	entry.ID = uuid.NewString()
	if entry.StartedAt <= 0 {
		entry.StartedAt = time.Now().UnixMilli()
	}
	entry.RequestID = string([]rune(entry.RequestID)[:min(len([]rune(entry.RequestID)), 128)])
	entry.TokenName = string([]rune(entry.TokenName)[:min(len([]rune(entry.TokenName)), 128)])
	entry.ModelName = string([]rune(entry.ModelName)[:min(len([]rune(entry.ModelName)), 256)])
	entry.Status = "processing"
	if entry.IsStream {
		entry.Status = "waiting_response"
	}
	entry.FirstResponseMS = nil
	r.mu.Lock()
	r.entries[entry.ID] = Record{UserID: userID, Entry: entry}
	r.mu.Unlock()
	return &Handle{registry: r, id: entry.ID}
}

func (h *Handle) Attempt(stream bool, retry int) {
	if h == nil {
		return
	}
	r := h.registry
	r.mu.Lock()
	record, exists := r.entries[h.id]
	if exists {
		record.Entry.IsStream = stream
		record.Entry.RetryCount = retry
		record.Entry.FirstResponseMS = nil
		record.Entry.Status = "processing"
		if stream {
			record.Entry.Status = "waiting_response"
		}
		if retry > 0 {
			record.Entry.Status = "retrying"
		}
		r.entries[h.id] = record
	}
	r.mu.Unlock()
}

func (h *Handle) FirstResponse(at time.Time) {
	if h == nil {
		return
	}
	r := h.registry
	r.mu.Lock()
	record, exists := r.entries[h.id]
	changed := exists && record.Entry.FirstResponseMS == nil
	if changed {
		elapsed := max(int64(0), at.UnixMilli()-record.Entry.StartedAt)
		record.Entry.FirstResponseMS = &elapsed
		if record.Entry.IsStream {
			record.Entry.Status = "receiving"
		}
		r.entries[h.id] = record
	}
	r.mu.Unlock()
}

func (h *Handle) Finish() {
	if h == nil {
		return
	}
	r := h.registry
	r.mu.Lock()
	if record, exists := r.entries[h.id]; exists {
		delete(r.entries, h.id)
		if r.store != nil {
			r.removals[h.id] = Removal{UserID: record.UserID, ID: h.id, At: time.Now()}
		}
	}
	r.mu.Unlock()
}

// Publish 串行发布不可变快照，避免结束删除被旧的并行续租覆盖。
func (r *Registry) Publish(ctx context.Context, now time.Time) error {
	if r.store == nil {
		return nil
	}
	r.publishMu.Lock()
	defer r.publishMu.Unlock()
	r.mu.Lock()
	entries := make([]Record, 0, len(r.entries))
	for _, record := range r.entries {
		entries = append(entries, record)
	}
	removals := make([]Removal, 0, len(r.removals))
	for id, removal := range r.removals {
		if now.Sub(removal.At) > 30*time.Second {
			delete(r.removals, id)
			continue
		}
		removals = append(removals, removal)
	}
	r.mu.Unlock()
	if err := r.store.Write(ctx, entries, removals, now); err != nil {
		r.mu.Lock()
		r.publishErr = err
		r.mu.Unlock()
		return err
	}
	r.mu.Lock()
	r.publishErr = nil
	for _, removal := range removals {
		delete(r.removals, removal.ID)
	}
	r.mu.Unlock()
	return nil
}

// Run 由宿主启动一个发布循环；业务请求只更新内存，不等待 Redis。
func (r *Registry) Run(ctx context.Context) {
	if r.store == nil {
		return
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		// 固定节拍合并所有变化，避免高频请求触发全量续租放大。
		writeCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		_ = r.Publish(writeCtx, time.Now())
		cancel()
	}
}

func (r *Registry) List(ctx context.Context, userID int, query Query) (Page, error) {
	if userID <= 0 {
		return Page{}, errors.New("invalid active request owner")
	}
	now := time.Now()
	page := Page{Items: []Entry{}, Page: max(1, query.Page), PageSize: query.PageSize, Scope: "instance", ObservedAt: now.UnixMilli(), Summary: Summary{ByToken: make(map[string]int)}}
	if page.PageSize <= 0 || page.PageSize > 50 {
		page.PageSize = 10
	}
	var entries []Entry
	if r.store != nil {
		page.Scope = "cluster"
		r.mu.Lock()
		publishErr := r.publishErr
		r.mu.Unlock()
		if publishErr != nil {
			return Page{}, errors.New("active request publishing unavailable")
		}
		var err error
		entries, err = r.store.Read(ctx, userID, now)
		if err != nil {
			return Page{}, err
		}
	} else {
		r.mu.Lock()
		for _, record := range r.entries {
			if record.UserID == userID {
				entries = append(entries, record.Entry)
			}
		}
		r.mu.Unlock()
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].StartedAt == entries[j].StartedAt {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].StartedAt < entries[j].StartedAt
	})
	search := strings.ToLower(strings.TrimSpace(query.Search))
	filtered := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		page.Summary.Total++
		if entry.IsStream {
			page.Summary.Streaming++
		} else {
			page.Summary.NonStreaming++
		}
		if entry.Status == "waiting_response" || entry.Status == "retrying" {
			page.Summary.Waiting++
		}
		if entry.TokenID > 0 {
			page.Summary.ByToken[strconv.Itoa(entry.TokenID)]++
		}
		if query.TokenID > 0 && entry.TokenID != query.TokenID || query.Status != "" && entry.Status != query.Status {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(entry.ModelName+" "+entry.TokenName+" "+entry.RequestID), search) {
			continue
		}
		filtered = append(filtered, entry)
	}
	page.Summary.ActiveTokens = len(page.Summary.ByToken)
	page.Total = len(filtered)
	// 先限制页号，避免不可信页号与大小相乘溢出。
	if page.Page <= (page.Total+page.PageSize-1)/page.PageSize {
		start := (page.Page - 1) * page.PageSize
		page.Items = filtered[start:min(start+page.PageSize, page.Total)]
	}
	return page, nil
}
