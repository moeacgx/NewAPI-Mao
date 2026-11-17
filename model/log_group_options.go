package model

import "strings"

// GetLogGroupNamesForIDs 只返回已授权实体的真实标识名称，不以历史显示标签判断归属。
func GetLogGroupNamesForIDs(groupIDs []int) (map[string]string, error) {
	names := make(map[string]string)
	if len(groupIDs) == 0 {
		return names, nil
	}
	allowed := make(map[int]struct{}, len(groupIDs))
	for _, id := range groupIDs {
		if id > 0 {
			allowed[id] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return names, nil
	}
	// 全局当前 code 优先于别名，包含用户无权访问的实体；不向响应暴露这些实体。
	var groups []Group
	if err := DB.Select("id", "code", "name").Order("id ASC").Find(&groups).Error; err != nil {
		return nil, err
	}
	currentCodes := make(map[string]struct{}, len(groups))
	namesByID := make(map[int]string, len(allowed))
	for _, group := range groups {
		code := strings.TrimSpace(group.Code)
		currentCodes[code] = struct{}{}
		if _, ok := allowed[group.Id]; !ok || code == "" || isVirtualAutoCode(code) {
			continue
		}
		name := strings.TrimSpace(group.Name)
		if name == "" {
			name = code
		}
		names[code] = name
		namesByID[group.Id] = name
	}
	var aliases []GroupAlias
	if err := DB.Where("group_id IN ?", groupIDs).Order("id ASC").Find(&aliases).Error; err != nil {
		return nil, err
	}
	for _, alias := range aliases {
		value := strings.TrimSpace(alias.Alias)
		name, allowed := namesByID[alias.GroupId]
		if !allowed || value == "" || isVirtualAutoCode(value) {
			continue
		}
		if _, isCurrentCode := currentCodes[value]; isCurrentCode {
			continue
		}
		if _, exists := names[value]; !exists {
			names[value] = name
		}
	}
	return names, nil
}
