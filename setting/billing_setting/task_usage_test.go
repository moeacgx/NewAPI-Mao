package billing_setting

import (
	"github.com/QuantumNous/new-api/pkg/jsplugin"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTaskExprCompatibleRequiresDeclaredUsage(t *testing.T) {
	schema := map[string]jsplugin.UsageFieldSchema{"input_tokens": {Type: "number", Unit: "token"}}
	assert.True(t, TaskExprCompatible(`u("input_tokens") * 0.042 / 1000000`, schema))
	assert.False(t, TaskExprCompatible(`u("unknown") * 0.042`, schema))
	assert.False(t, TaskExprCompatible(`true ? u("input_tokens") : u("unknown")`, schema))
	assert.False(t, TaskExprCompatible(`u(`, schema))
	assert.False(t, TaskExprCompatible("", schema))
}
