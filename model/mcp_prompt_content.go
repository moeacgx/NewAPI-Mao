package model

import "errors"

// ListMCPPromptContentPage 先做数据库分页，再批量装配当前页正文，避免加载整个技能目录。
func ListMCPPromptContentPage(page, pageSize int) ([]PromptTemplate, int64, error) {
	if page < 1 || page > 10000 || pageSize < 1 || pageSize > 100 {
		return nil, 0, errors.New("提示词分页参数无效")
	}
	var total int64
	if err := DB.Model(&PromptTemplate{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]PromptTemplate, 0)
	if err := DB.Order("sort_order, id").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	if err := hydratePromptTemplateGroups(DB, items, true); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
