package model

import (
	"strconv"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOptionValueRejectsInvalidMaxTokenAutoGroups(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "1.5", "invalid"} {
		t.Run(value, func(t *testing.T) {
			assert.Error(t, validateOptionValue("MaxTokenAutoGroups", value))
		})
	}
	require.NoError(t, validateOptionValue("MaxTokenAutoGroups", "999999"))
}

func TestMaxTokenAutoGroupsOptionReadWrite(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&Option{}))
	original := setting.GetMaxTokenAutoGroups()
	common.OptionMapRWMutex.Lock()
	previousMap := common.OptionMap
	if common.OptionMap == nil {
		common.OptionMap = map[string]string{}
	}
	previousValue, hadValue := common.OptionMap["MaxTokenAutoGroups"]
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(strconv.Itoa(original)))
		_ = DB.Where(commonKeyCol+" = ?", "MaxTokenAutoGroups").Delete(&Option{}).Error
		common.OptionMapRWMutex.Lock()
		if previousMap == nil {
			common.OptionMap = nil
		} else if hadValue {
			common.OptionMap["MaxTokenAutoGroups"] = previousValue
		} else if common.OptionMap != nil {
			delete(common.OptionMap, "MaxTokenAutoGroups")
		}
		common.OptionMapRWMutex.Unlock()
	})

	require.NoError(t, UpdateOption("MaxTokenAutoGroups", "8"))
	assert.Equal(t, 8, setting.GetMaxTokenAutoGroups())
	common.OptionMapRWMutex.RLock()
	assert.Equal(t, "8", common.OptionMap["MaxTokenAutoGroups"])
	common.OptionMapRWMutex.RUnlock()
	var stored Option
	require.NoError(t, DB.Where(commonKeyCol+" = ?", "MaxTokenAutoGroups").First(&stored).Error)
	assert.Equal(t, "8", stored.Value)

	require.Error(t, UpdateOption("MaxTokenAutoGroups", "0"))
	assert.Equal(t, 8, setting.GetMaxTokenAutoGroups())
	require.NoError(t, DB.Where(commonKeyCol+" = ?", "MaxTokenAutoGroups").First(&stored).Error)
	assert.Equal(t, "8", stored.Value)

	require.NoError(t, validateGroupConfigOptionUpdates(map[string]string{"MaxTokenAutoGroups": "6"}))
	require.Error(t, validateGroupConfigOptionUpdates(map[string]string{"MaxTokenAutoGroups": "-1"}))
}
