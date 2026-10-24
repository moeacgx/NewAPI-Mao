package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/gin-gonic/gin"
)

const GatewaySkillMaxRounds = 8
const GatewaySkillMaxCalls = 8
const gatewaySkillReadBudget = 192 * 1024
const gatewaySkillCatalogueBudget = 64 * 1024
const gatewaySkillAuditKey = "gateway_skills_audit"

// GatewaySkillResponseUsageKey 保存本次正式生成实际读取成功的技能。
// 该键只在 FinalBody 完成所有授权校验并成功装载后设置，不能用目录发现数量替代。
const gatewaySkillResponseUsageKey = "gateway_skill_response_usage"

// GatewaySkillResponseSkill 是响应头和回复标识使用的最小技能元数据。
type GatewaySkillResponseSkill struct {
	TemplateID int    `json:"template_id"`
	SkillID    string `json:"skill_id"`
	Version    int    `json:"version"`
	Title      string `json:"title"`
}

// GatewaySkillResponseUsage 描述本次实际装载的技能，不包含正文或资源路径。
type GatewaySkillResponseUsage struct {
	Version   int                         `json:"version"`
	Total     int                         `json:"total"`
	Skills    []GatewaySkillResponseSkill `json:"skills"`
	Truncated bool                        `json:"truncated"`
}

type gatewaySkillPage struct {
	Skill model.TokenProgressiveSkill
	Path  string
	Page  model.SkillFilePage
}

type gatewaySkillCatalogueQuery struct {
	Query  string
	Offset int
}

// GatewaySkillSession 只在一个已选定分组、渠道的读取阶段内有效，不跨重试复用。
type GatewaySkillSession struct {
	tokenID   int
	group     string
	format    types.RelayFormat
	modelName string
	skills    []model.TokenProgressiveSkill
	pages     []gatewaySkillPage
	readBytes int
	resources map[string]map[string]bool
	// catalogueQueries 只保存模型成功执行过的查询参数；正式生成前会按当前授权重新查询，
	// 避免把已经撤销或改版的目录结果带入正式请求。
	catalogueQueries []gatewaySkillCatalogueQuery
}

// HasReadFiles 表示本次私有阶段实际读到了授权正文，不把目录或空 EOF 页当作正文读取。
func (s *GatewaySkillSession) HasReadFiles() bool {
	for _, page := range s.pages {
		if len(page.Page.Content) > 0 {
			return true
		}
	}
	return false
}

func NewGatewaySkillSession(c *gin.Context, format types.RelayFormat) (*GatewaySkillSession, error) {
	c.Set(gatewaySkillAuditKey, nil)
	ResetGatewaySkillResponse(c)
	if !supportsTokenPromptEndpoint(c, format) {
		return nil, nil
	}
	tokenID := common.GetContextKeyInt(c, constant.ContextKeyTokenId)
	if tokenID <= 0 {
		return nil, nil
	}
	group := common.GetContextKeyString(c, constant.ContextKeyUsingGroup)
	skills, err := model.GetActiveTokenSkills(tokenID, group)
	if err != nil || len(skills) == 0 {
		return nil, err
	}
	c.Set(tokenPromptManagedKey, true)
	return &GatewaySkillSession{tokenID: tokenID, group: group, format: format,
		modelName: common.GetContextKeyString(c, constant.ContextKeyOriginalModel), skills: skills,
		resources: make(map[string]map[string]bool)}, nil
}

// Catalogue 只发送名称与用途；目录较大时通过搜索工具继续发现，绝不截断文件正文。
func (s *GatewaySkillSession) Catalogue() (string, error) {
	entries := make([]map[string]any, 0)
	for _, skill := range s.skills {
		entries = append(entries, map[string]any{"template_id": skill.TemplateID, "skill_id": skill.SkillID,
			"title": skill.Title, "description": skill.Description, "package": skill.PackageTitle, "version": skill.Version})
	}
	encoded, err := common.Marshal(map[string]any{"skills": entries, "total": len(entries)})
	if err != nil {
		return "", err
	}
	if len(encoded) <= 64*1024 {
		return string(encoded), nil
	}
	page, end, err := gatewaySkillMetadataPage(entries, 0)
	if err != nil {
		return "", err
	}
	encoded, err = common.Marshal(map[string]any{"skills": page, "total": len(entries),
		"next_offset": end, "note": "Use newapi_skills_list with query or offset to discover more relevant skills."})
	return string(encoded), err
}

func (s *GatewaySkillSession) Execute(c *gin.Context, calls []GatewaySkillCall) ([]GatewaySkillResult, error) {
	if common.GetContextKeyString(c, constant.ContextKeyUsingGroup) != s.group {
		return nil, errors.New("技能读取期间实际分组已改变")
	}
	if len(calls) > GatewaySkillMaxCalls {
		return nil, errors.New("技能读取阶段单轮工具调用超过 8 项")
	}
	results := make([]GatewaySkillResult, 0, len(calls))
	for _, call := range calls {
		if err := c.Request.Context().Err(); err != nil {
			return nil, err
		}
		var payload any
		var toolErr error
		switch call.Name {
		case GatewaySkillsListTool:
			var args struct {
				Query  string `json:"query"`
				Offset int    `json:"offset"`
			}
			if common.GetJsonType(call.Arguments) != "object" || common.Unmarshal(call.Arguments, &args) != nil || args.Offset < 0 || len(args.Query) > 1024 {
				toolErr = errors.New("技能目录查询参数无效")
				break
			}
			current, err := model.GetActiveTokenSkills(s.tokenID, s.group)
			if err != nil {
				return nil, err
			}
			entries := make([]map[string]any, 0)
			query := strings.ToLower(strings.TrimSpace(args.Query))
			for _, skill := range current {
				if !s.contains(skill.TemplateID, skill.SkillID, skill.Version) ||
					(query != "" && !strings.Contains(strings.ToLower(skill.Title+"\n"+skill.Description+"\n"+skill.PackageTitle), query)) {
					continue
				}
				entries = append(entries, map[string]any{"template_id": skill.TemplateID, "skill_id": skill.SkillID,
					"title": skill.Title, "description": skill.Description, "package": skill.PackageTitle, "version": skill.Version})
			}
			page, end, err := gatewaySkillMetadataPage(entries, args.Offset)
			if err != nil {
				return nil, err
			}
			payload = map[string]any{"skills": page, "total": len(entries), "next_offset": end, "eof": end == len(entries)}
			s.rememberCatalogueQuery(gatewaySkillCatalogueQuery{Query: strings.TrimSpace(args.Query), Offset: args.Offset})
		case GatewaySkillsFilesTool:
			var args struct {
				TemplateID int    `json:"template_id"`
				SkillID    string `json:"skill_id"`
				Version    int    `json:"version"`
				Query      string `json:"query"`
				Offset     int    `json:"offset"`
			}
			if common.GetJsonType(call.Arguments) != "object" || common.Unmarshal(call.Arguments, &args) != nil || args.Offset < 0 || len(args.Query) > 1024 ||
				!s.contains(args.TemplateID, args.SkillID, args.Version) {
				toolErr = errors.New("技能文件目录参数无效或不属于本次授权目录")
				break
			}
			key := strconv.Itoa(args.TemplateID) + "/" + args.SkillID
			if s.resources[key] == nil {
				toolErr = errors.New("请先读取 SKILL.md，再查询引用文件目录")
				break
			}
			main, err := model.ReadActiveTokenSkillFile(s.tokenID, s.group, args.TemplateID, args.SkillID, args.Version, "SKILL.md", 0, 4)
			if err != nil {
				toolErr = err
				break
			}
			paths := make([]string, 0)
			query := strings.ToLower(strings.TrimSpace(args.Query))
			for _, path := range main.Resources {
				if query == "" || strings.Contains(strings.ToLower(path), query) {
					paths = append(paths, path)
				}
			}
			page, end, err := gatewaySkillResourcePaths(paths, args.Offset)
			if err != nil {
				return nil, err
			}
			for _, path := range page {
				s.resources[key][path] = true
			}
			payload = map[string]any{"resources": page, "total": len(paths), "next_offset": end, "eof": end == len(paths)}
		case GatewaySkillsReadTool:
			var args struct {
				TemplateID int    `json:"template_id"`
				SkillID    string `json:"skill_id"`
				Version    int    `json:"version"`
				Path       string `json:"path"`
				Offset     int    `json:"offset"`
			}
			if common.GetJsonType(call.Arguments) != "object" || common.Unmarshal(call.Arguments, &args) != nil || args.Offset < 0 ||
				!s.contains(args.TemplateID, args.SkillID, args.Version) {
				toolErr = errors.New("技能读取参数无效或不属于本次授权目录")
				break
			}
			if args.Path == "" {
				args.Path = "SKILL.md"
			}
			key := strconv.Itoa(args.TemplateID) + "/" + args.SkillID
			if args.Path != "SKILL.md" && !s.resources[key][args.Path] {
				toolErr = errors.New("请先读取 SKILL.md，再按返回的路径读取引用文本")
				break
			}
			limit := min(model.MaxSkillReadBytes, gatewaySkillReadBudget-s.readBytes)
			duplicate := false
			for _, existing := range s.pages {
				if existing.Skill.TemplateID == args.TemplateID && existing.Skill.SkillID == args.SkillID && existing.Path == args.Path && existing.Page.Offset == args.Offset {
					limit = max(1, len(existing.Page.Content))
					duplicate = true
					break
				}
			}
			if limit <= 0 {
				toolErr = errors.New("本次技能读取预算已用完，请根据已读资料继续")
				break
			}
			page, err := model.ReadActiveTokenSkillFile(s.tokenID, s.group, args.TemplateID, args.SkillID, args.Version, args.Path, args.Offset, limit)
			if err != nil {
				toolErr = err
				break
			}
			paths, pathEnd, err := gatewaySkillResourcePaths(page.Resources, 0)
			if err != nil {
				return nil, err
			}
			if args.Path == "SKILL.md" {
				if s.resources[key] == nil {
					s.resources[key] = make(map[string]bool)
				}
				for _, path := range paths {
					s.resources[key][path] = true
				}
			}
			if !duplicate {
				for _, skill := range s.skills {
					if skill.TemplateID == args.TemplateID && skill.SkillID == args.SkillID {
						stored := *page
						stored.Resources = nil
						s.pages = append(s.pages, gatewaySkillPage{Skill: skill, Path: args.Path, Page: stored})
						break
					}
				}
				s.readBytes += len(page.Content)
			}
			payload = map[string]any{"template_id": args.TemplateID, "skill_id": args.SkillID, "version": args.Version,
				"path": args.Path, "content": page.Content, "offset": page.Offset, "next_offset": page.NextOffset,
				"total_bytes": page.TotalBytes, "eof": page.EOF, "resources": paths, "resources_total": len(page.Resources),
				"resources_next_offset": pathEnd, "remaining_bytes": gatewaySkillReadBudget - s.readBytes}
		default:
			return nil, errors.New("技能读取阶段不能执行客户端工具")
		}
		if toolErr != nil {
			payload = map[string]any{"error": toolErr.Error()}
		}
		encoded, err := common.Marshal(payload)
		if err != nil {
			return nil, err
		}
		results = append(results, GatewaySkillResult{Call: call, Content: json.RawMessage(encoded)})
	}
	return results, nil
}

// gatewaySkillMetadataPage 同时限制条数和序列化字节数，避免长描述挤满读取上下文。
func gatewaySkillMetadataPage(entries []map[string]any, offset int) ([]map[string]any, int, error) {
	start := min(offset, len(entries))
	end, size := start, 2
	for end < len(entries) && end-start < 32 {
		encoded, err := common.Marshal(entries[end])
		if err != nil {
			return nil, 0, err
		}
		// 单条说明可接近 16 KiB；每页至少返回一条，保证目录分页总能前进。
		if size+len(encoded)+1 > 32*1024 && end > start {
			break
		}
		size += len(encoded) + 1
		end++
	}
	return entries[start:end], end, nil
}

// gatewaySkillResourcePaths 为引用目录单独分页，正文读取预算不包含未披露的路径。
func gatewaySkillResourcePaths(paths []string, offset int) ([]string, int, error) {
	start := min(offset, len(paths))
	end, size := start, 2
	for end < len(paths) && end-start < 32 {
		encoded, err := common.Marshal(paths[end])
		if err != nil {
			return nil, 0, err
		}
		if size+len(encoded)+1 > 8*1024 {
			break
		}
		size += len(encoded) + 1
		end++
	}
	return paths[start:end], end, nil
}

func (s *GatewaySkillSession) contains(templateID int, skillID string, version int) bool {
	for _, skill := range s.skills {
		if skill.TemplateID == templateID && skill.SkillID == skillID && skill.Version == version {
			return true
		}
	}
	return false
}

func (s *GatewaySkillSession) rememberCatalogueQuery(query gatewaySkillCatalogueQuery) {
	for _, existing := range s.catalogueQueries {
		if existing == query {
			return
		}
	}
	s.catalogueQueries = append(s.catalogueQueries, query)
}

// catalogueText 只重建模型实际查询过的目录页，不把完整技能包自动注入正式请求。
// 重新读取当前授权数据是为了避免目录发现与正式生成之间发生卸载、下架或版本更新时泄漏旧元数据。
func (s *GatewaySkillSession) catalogueText() (string, error) {
	if len(s.catalogueQueries) == 0 {
		return "", nil
	}
	current, err := model.GetActiveTokenSkills(s.tokenID, s.group)
	if err != nil {
		return "", err
	}
	lists := make([]map[string]any, 0, len(s.catalogueQueries))
	for _, query := range s.catalogueQueries {
		entries := make([]map[string]any, 0)
		needle := strings.ToLower(query.Query)
		for _, skill := range current {
			if !s.contains(skill.TemplateID, skill.SkillID, skill.Version) {
				continue
			}
			if needle != "" && !strings.Contains(strings.ToLower(skill.Title+"\n"+skill.Description+"\n"+skill.PackageTitle), needle) {
				continue
			}
			entries = append(entries, map[string]any{"template_id": skill.TemplateID, "skill_id": skill.SkillID,
				"title": skill.Title, "description": skill.Description, "package": skill.PackageTitle, "version": skill.Version})
		}
		page, end, err := gatewaySkillMetadataPage(entries, query.Offset)
		if err != nil {
			return "", err
		}
		lists = append(lists, map[string]any{"query": query.Query, "offset": query.Offset, "skills": page,
			"total": len(entries), "next_offset": end, "eof": end == len(entries)})
	}
	metadata := map[string]any{"lists": lists, "truncated": false}
	encoded, err := common.Marshal(metadata)
	if err != nil {
		return "", err
	}
	// 单页已受 32 KiB 限制；按完整查询页收拢，保留有效 JSON 和 UTF-8 边界。
	for len(encoded) > gatewaySkillCatalogueBudget && len(lists) > 0 {
		lists = lists[:len(lists)-1]
		metadata["lists"] = lists
		metadata["truncated"] = true
		encoded, err = common.Marshal(metadata)
		if err != nil {
			return "", err
		}
	}
	return "网关已完成当前令牌可用 Skills 的目录查询，可依据以下真实结果回答技能列表或用途问题。" +
		"以下仅是本轮实际查询的元数据，不包含技能正文，也不代表已经使用技能：\n<gateway_skill_catalogue>\n" +
		string(encoded) + "\n</gateway_skill_catalogue>", nil
}

// FinalBody 重新检查撤销、版本和实际分组后，只装载本轮查询的目录元数据及实际读过的文件页。
func (s *GatewaySkillSession) FinalBody(c *gin.Context, original []byte) ([]byte, error) {
	ResetGatewaySkillResponse(c)
	if common.GetContextKeyString(c, constant.ContextKeyUsingGroup) != s.group {
		return nil, errors.New("技能读取期间实际分组已改变")
	}
	texts := make([]string, 0, len(s.pages))
	for _, read := range s.pages {
		if len(read.Page.Content) == 0 {
			continue
		}
		if err := c.Request.Context().Err(); err != nil {
			return nil, err
		}
		page, err := model.ReadActiveTokenSkillFile(s.tokenID, s.group, read.Skill.TemplateID, read.Skill.SkillID, read.Skill.Version,
			read.Path, read.Page.Offset, max(1, len(read.Page.Content)))
		if err != nil {
			return nil, fmt.Errorf("技能读取授权已失效: %w", err)
		}
		if page.Content != read.Page.Content {
			return nil, errors.New("技能文件在读取后发生变化")
		}
		texts = append(texts, fmt.Sprintf("Installed skill %q (package %q, version %d), file %q, bytes %d..%d of %d, eof=%t:\n%s",
			read.Skill.Title, read.Skill.PackageTitle, read.Skill.Version, read.Path, page.Offset, page.NextOffset, page.TotalBytes, page.EOF, page.Content))
	}
	catalogue, err := s.catalogueText()
	if err != nil {
		return nil, err
	}
	parts := make([]string, 0, 2)
	if catalogue != "" {
		parts = append(parts, catalogue)
	}
	if len(texts) > 0 {
		parts = append(parts, strings.Join(texts, "\n\n"))
	}
	if len(parts) == 0 {
		return original, nil
	}
	modified, err := appendSystemInstructionText(original, s.format, s.modelName, strings.Join(parts, "\n\n"))
	if err != nil {
		return nil, err
	}
	usage := make([]GatewaySkillResponseSkill, 0, len(s.pages))
	seen := make(map[string]struct{}, len(s.pages))
	for _, read := range s.pages {
		if len(read.Page.Content) == 0 {
			continue
		}
		key := fmt.Sprintf("%d\x00%s\x00%d", read.Skill.TemplateID, read.Skill.SkillID, read.Skill.Version)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		usage = append(usage, GatewaySkillResponseSkill{
			TemplateID: read.Skill.TemplateID,
			SkillID:    read.Skill.SkillID,
			Version:    read.Skill.Version,
			Title:      read.Skill.Title,
		})
	}
	if len(usage) > 0 {
		c.Set(gatewaySkillResponseUsageKey, &GatewaySkillResponseUsage{
			Version: 1,
			Total:   len(usage),
			Skills:  usage,
		})
	}
	return modified, nil
}

func (s *GatewaySkillSession) Audit(c *gin.Context, stage string, round int) {
	pages := make([]map[string]any, 0, len(s.pages))
	for _, read := range s.pages {
		pages = append(pages, map[string]any{"template_id": read.Skill.TemplateID, "skill_id": read.Skill.SkillID,
			"title": read.Skill.Title, "version": read.Skill.Version, "path": read.Path, "offset": read.Page.Offset, "bytes": len(read.Page.Content)})
	}
	queries := make([]map[string]any, 0, len(s.catalogueQueries))
	for _, query := range s.catalogueQueries {
		queries = append(queries, map[string]any{"query_bytes": len(query.Query), "offset": query.Offset})
	}
	c.Set(gatewaySkillAuditKey, map[string]any{"stage": stage, "round": round, "catalogue_count": len(s.skills), "catalogue_queries": queries, "files": pages})
}
