package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service/authz"
)

var ErrMCPUnauthorized = errors.New("MCP credential is invalid or unavailable")

type MCPIdentity struct {
	Key  *model.MCPKey
	User *model.UserBase
}

// 私有 context key 不接受 HTTP header 或 Gin 字符串键注入。只有进程内桥接可创建。
type mcpInternalIdentityKey struct{}

func MCPInternalContext(ctx context.Context, keyID int) context.Context {
	return context.WithValue(ctx, mcpInternalIdentityKey{}, keyID)
}

func MCPInternalKeyID(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(mcpInternalIdentityKey{}).(int)
	return id, ok && id > 0
}

func MCPKeyDigest(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func AuthenticateMCP(secret string) (*MCPIdentity, error) {
	if len(secret) != 47 || !strings.HasPrefix(secret, "mcp_") {
		return nil, ErrMCPUnauthorized
	}
	var key model.MCPKey
	if err := model.DB.Where("key_hash = ?", MCPKeyDigest(secret)).First(&key).Error; err != nil {
		return nil, ErrMCPUnauthorized
	}
	return validateMCPKey(&key)
}

func ValidateMCPIdentity(keyID int) (*MCPIdentity, error) {
	var key model.MCPKey
	if err := model.DB.First(&key, keyID).Error; err != nil {
		return nil, ErrMCPUnauthorized
	}
	return validateMCPKey(&key)
}

func validateMCPKey(key *model.MCPKey) (*MCPIdentity, error) {
	if key.RevokedAt != 0 || key.ExpiresAt <= time.Now().Unix() {
		return nil, ErrMCPUnauthorized
	}
	// 每次直读数据库，不能用可能尚未同步撤权的缓存快照。
	user, err := model.GetUserById(key.UserID, false)
	if err != nil || user.Status != common.UserStatusEnabled || user.Role < common.RoleAdminUser || !common.IsValidateRole(user.Role) {
		return nil, ErrMCPUnauthorized
	}
	return &MCPIdentity{Key: key, User: user.ToBaseUser()}, nil
}

// MCPPermissionAllowed 使用与 authz 相同的用户覆盖优先、deny 优先和管理员基线规则，
// 但直接读取共享数据库，避免跨实例权限同步窗口继续允许已经撤销的操作。
func MCPPermissionAllowed(userID, role int, permission authz.Permission) bool {
	if role < common.RoleAdminUser {
		return false
	}
	if permission.Resource == "" && permission.Action == "" {
		return true
	}
	known := false
	for _, candidate := range authz.AllPermissions() {
		if candidate == permission {
			known = true
			break
		}
	}
	if !known {
		return false
	}
	if role == common.RoleRootUser {
		return true
	}
	subject := authz.UserSubject(userID)
	var rules []model.CasbinRule
	if err := model.DB.Where("ptype = ? AND v0 IN ? AND v1 = ? AND v2 = ?", "p", []string{subject, authz.RoleSubject(authz.BuiltInRoleAdmin)}, permission.Resource, permission.Action).Find(&rules).Error; err != nil {
		return false
	}
	for _, current := range []string{subject, authz.RoleSubject(authz.BuiltInRoleAdmin)} {
		allow, present := false, false
		for _, rule := range rules {
			if rule.V0 != current {
				continue
			}
			present = true
			if rule.V3 == authz.EffectDeny {
				return false
			}
			allow = allow || rule.V3 == authz.EffectAllow || rule.V3 == ""
		}
		if present {
			return allow
		}
	}
	return false
}

func MCPVisibleTools(user *model.UserBase, key *model.MCPKey) []MCPTool {
	tools := make([]MCPTool, 0)
	granted := make(map[string]bool)
	if key != nil {
		for _, name := range key.Tools {
			granted[name] = true
		}
	}
	// 仅本次发现复用相同权限的结果；下次调用重新读取数据库，保持即时撤权。
	permissions := make(map[authz.Permission]bool)
	for _, tool := range MCPTools() {
		if key != nil && !granted[tool.Name] {
			continue
		}
		if user.Role < tool.MinRole {
			continue
		}
		allowed, cached := permissions[tool.Permission]
		if !cached {
			allowed = MCPPermissionAllowed(user.Id, user.Role, tool.Permission)
			permissions[tool.Permission] = allowed
		}
		if !allowed {
			continue
		}
		tools = append(tools, tool)
	}
	return tools
}

func CreateMCPKey(user *model.UserBase, name string, tools []string, expiresAt int64) (*model.MCPKey, string, error) {
	name = strings.TrimSpace(name)
	// 以公元 9999 年末为上界，保持 SQL 时间与管理端日期显示的共同有效范围。
	if name == "" || len([]rune(name)) > 100 || expiresAt <= time.Now().Unix() || expiresAt > 253402300799 || len(tools) == 0 || len(tools) > len(MCPTools()) {
		return nil, "", errors.New("invalid name, tools or expires_at")
	}
	available := make(map[string]bool)
	for _, tool := range MCPVisibleTools(user, nil) {
		available[tool.Name] = true
	}
	seen := make(map[string]bool)
	for _, name := range tools {
		if !available[name] || seen[name] {
			return nil, "", errors.New("tool is unavailable or duplicated")
		}
		seen[name] = true
	}
	var entropy [32]byte
	if _, err := rand.Read(entropy[:]); err != nil {
		return nil, "", errors.New("cannot generate MCP key")
	}
	secret := "mcp_" + base64.RawURLEncoding.EncodeToString(entropy[:])
	key := &model.MCPKey{UserID: user.Id, Name: name, KeyHash: MCPKeyDigest(secret), KeyPrefix: secret[:12], Tools: tools, ExpiresAt: expiresAt}
	if err := model.DB.Create(key).Error; err != nil {
		return nil, "", errors.New("cannot save MCP key")
	}
	return key, secret, nil
}

type mcpKeyLimit struct {
	window int64
	count  int
	active int
}

var mcpLimits = struct {
	sync.Mutex
	keys map[int]*mcpKeyLimit
}{keys: make(map[int]*mcpKeyLimit)}

// 每个实例每 key 每分钟 60 次、最多 4 个在途请求；独立于管理员限流豁免。
// 表有容量上限，活跃请求不被清理；进程退出不会遗留永久占用的数据库锁。
func AcquireMCPLimit(keyID int) (func(), bool) {
	mcpLimits.Lock()
	defer mcpLimits.Unlock()
	now := time.Now().Unix() / 60
	for id, entry := range mcpLimits.keys {
		if entry.active == 0 && entry.window < now-1 {
			delete(mcpLimits.keys, id)
		}
	}
	entry := mcpLimits.keys[keyID]
	if entry == nil {
		if len(mcpLimits.keys) >= 4096 {
			return nil, false
		}
		entry = &mcpKeyLimit{window: now}
		mcpLimits.keys[keyID] = entry
	}
	if entry.window != now {
		entry.window, entry.count = now, 0
	}
	if entry.count >= 60 || entry.active >= 4 {
		return nil, false
	}
	entry.count++
	entry.active++
	var once sync.Once
	return func() {
		once.Do(func() {
			mcpLimits.Lock()
			entry.active--
			mcpLimits.Unlock()
		})
	}, true
}
