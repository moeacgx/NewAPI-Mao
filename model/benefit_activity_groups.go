package model

import (
	"errors"
	"slices"
	"sort"

	"gorm.io/gorm"
)

// BenefitActivityGroup stores secondary bindings. The primary binding remains
// on the activity so historical activities need no backfill.
type BenefitActivityGroup struct {
	ActivityID   int    `gorm:"primaryKey;autoIncrement:false"`
	GroupID      int    `gorm:"primaryKey;autoIncrement:false;index"`
	NameSnapshot string `gorm:"size:128;not null"`
}

func (BenefitActivityGroup) TableName() string { return "benefit_activity_groups" }

func benefitActivityPointers(activities []BenefitActivity) []*BenefitActivity {
	pointers := make([]*BenefitActivity, len(activities))
	for i := range activities {
		pointers[i] = &activities[i]
	}
	return pointers
}

// alias is a fixed, internal SQL table alias, never request input.
func benefitActivitiesInGroups(db *gorm.DB, alias string, ids []int) *gorm.DB {
	return db.Where("("+alias+".group_id IN ? OR EXISTS (SELECT 1 FROM benefit_activity_groups AS bg WHERE bg.activity_id = "+alias+".id AND bg.group_id IN ?))", ids, ids)
}

func benefitGroupBindings(tx *gorm.DB, activity *BenefitActivity) ([]BenefitActivityGroup, error) {
	ids := activity.GroupIDs
	if ids == nil {
		ids = []int{activity.GroupId}
	}
	if len(ids) == 0 || len(ids) > 500 || (activity.GroupId != 0 && ids[0] != activity.GroupId) {
		return nil, errors.New("福利活动分组列表无效")
	}
	var groups []Group
	if err := tx.Where("id IN ? AND status = ?", ids, GroupStatusActive).Find(&groups).Error; err != nil {
		return nil, err
	}
	if len(groups) != len(ids) {
		return nil, errors.New("福利活动分组不存在、未启用或重复")
	}
	byID := make(map[int]Group, len(groups))
	for _, group := range groups {
		byID[group.Id] = group
	}
	activity.GroupId = ids[0]
	activity.GroupCodeSnapshot = byID[ids[0]].Code
	activity.GroupNameSnapshot = byID[ids[0]].Name
	activity.GroupIDs = ids
	activity.GroupNames = []string{activity.GroupNameSnapshot}
	bindings := make([]BenefitActivityGroup, 0, len(ids)-1)
	for _, id := range ids[1:] {
		name := byID[id].Name
		bindings = append(bindings, BenefitActivityGroup{GroupID: id, NameSnapshot: name})
		activity.GroupNames = append(activity.GroupNames, name)
	}
	return bindings, nil
}

func saveBenefitGroupBindings(tx *gorm.DB, activityID int, bindings []BenefitActivityGroup) error {
	if err := tx.Where("activity_id = ?", activityID).Delete(&BenefitActivityGroup{}).Error; err != nil {
		return err
	}
	if len(bindings) == 0 {
		return nil
	}
	for i := range bindings {
		bindings[i].ActivityID = activityID
	}
	return tx.CreateInBatches(bindings, 100).Error
}

func loadBenefitActivityGroups(db *gorm.DB, activities ...*BenefitActivity) error {
	if len(activities) == 0 {
		return nil
	}
	byID := make(map[int]*BenefitActivity, len(activities))
	ids := make([]int, 0, len(activities))
	for _, a := range activities {
		ids = append(ids, a.Id)
		byID[a.Id] = a
		a.GroupIDs = []int{a.GroupId}
		a.GroupNames = []string{a.GroupNameSnapshot}
	}
	// Both activity and group lists can grow beyond the SQLite parameter limit.
	const lookupBatchSize = 500
	for batch := range slices.Chunk(ids, lookupBatchSize) {
		var bindings []BenefitActivityGroup
		if err := db.Where("activity_id IN ?", batch).Order("group_id ASC").Find(&bindings).Error; err != nil {
			return err
		}
		for _, b := range bindings {
			a := byID[b.ActivityID]
			a.GroupIDs = append(a.GroupIDs, b.GroupID)
			a.GroupNames = append(a.GroupNames, b.NameSnapshot)
		}
	}
	groupIDs := make([]int, 0, len(activities))
	names := make(map[int]string)
	for _, a := range activities {
		for _, id := range a.GroupIDs {
			if _, seen := names[id]; id > 0 && !seen {
				groupIDs = append(groupIDs, id)
				names[id] = ""
			}
		}
	}
	for batch := range slices.Chunk(groupIDs, lookupBatchSize) {
		var groups []Group
		if err := db.Select("id", "name").Where("id IN ?", batch).Find(&groups).Error; err != nil {
			return err
		}
		for _, group := range groups {
			names[group.Id] = group.Name
		}
	}
	for _, a := range activities {
		for i, id := range a.GroupIDs {
			if name := names[id]; name != "" {
				a.GroupNames[i] = name
			}
		}
	}
	return nil
}

func lockBenefitActivityGroups(tx *gorm.DB, activity *BenefitActivity) ([]int, error) {
	var bindings []BenefitActivityGroup
	if err := lockForUpdate(tx).Where("activity_id = ?", activity.Id).Find(&bindings).Error; err != nil {
		return nil, err
	}
	ids := []int{activity.GroupId}
	for _, binding := range bindings {
		ids = append(ids, binding.GroupID)
	}
	sort.Ints(ids)
	for _, id := range ids {
		if err := lockBenefitActivityGroupTx(tx, id); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

func benefitVoucherGroupNames(db *gorm.DB, vouchers []*BenefitUserVoucher) error {
	byID := make(map[int]*BenefitActivity)
	var activities []*BenefitActivity
	for _, voucher := range vouchers {
		if byID[voucher.ActivityId] == nil {
			a := &BenefitActivity{Id: voucher.ActivityId, GroupId: voucher.GroupID, GroupNameSnapshot: voucher.GroupNameSnapshot}
			byID[a.Id] = a
			activities = append(activities, a)
		}
	}
	if err := loadBenefitActivityGroups(db, activities...); err != nil {
		return err
	}
	for _, voucher := range vouchers {
		voucher.GroupNames = byID[voucher.ActivityId].GroupNames
	}
	return nil
}
