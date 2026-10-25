package service

import (
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/require"
)

func TestGatewaySkillCataloguePagesAdvanceWithLongDescriptions(t *testing.T) {
	// 控制字符经 JSON 转义后，单条说明超过 32 KiB 分页预算；每页仍需返回一条，目录发现不能卡在同一偏移。
	entries := make([]map[string]any, 0, 3)
	for i := 0; i < 3; i++ {
		entries = append(entries, map[string]any{"skill_id": "skill-" + strconv.Itoa(i), "description": strings.Repeat("\x01", model.MaxPromptDescriptionBytes)})
	}
	for offset := 0; offset < len(entries); {
		page, next, err := gatewaySkillMetadataPage(entries, offset)
		require.NoError(t, err)
		require.Len(t, page, 1)
		require.Equal(t, entries[offset]["skill_id"], page[0]["skill_id"])
		require.Equal(t, offset+1, next)
		offset = next
	}
}
