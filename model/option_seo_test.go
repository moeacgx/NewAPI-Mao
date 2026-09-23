package model

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSystemDescriptionUsesUnicodeCodePointLimit(t *testing.T) {
	assert.NoError(t, validateOptionValue("SystemDescription", ""))
	assert.NoError(t, validateOptionValue("SystemDescription", strings.Repeat("界", 200)))
	require.Error(t, validateOptionValue("SystemDescription", strings.Repeat("界", 201)))
}
