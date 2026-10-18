package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetGroupDisplayNameForErrorUsesCurrentName(t *testing.T) {
	previousDB := DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	DB = db
	t.Cleanup(func() {
		DB = previousDB
	})
	require.NoError(t, db.AutoMigrate(&Group{}, &GroupAlias{}, &Option{}))

	group := &Group{Code: "Codex-Pro", Name: "Codex 专业组", Status: GroupStatusActive}
	require.NoError(t, db.Create(group).Error)
	require.NoError(t, db.Create(&GroupAlias{Alias: "legacy-codex-pro", GroupId: group.Id}).Error)

	require.Equal(t, "Codex 专业组", GetGroupDisplayNameForError("Codex-Pro"))
	require.Equal(t, "Codex 专业组", GetGroupDisplayNameForError("legacy-codex-pro"))
	require.Equal(t, "Codex 专业组, Codex 专业组", GetGroupDisplayNameForError("Codex-Pro,legacy-codex-pro"))
}

func TestGetGroupDisplayNameForErrorFallsBackWithoutDatabase(t *testing.T) {
	previousDB := DB
	DB = nil
	t.Cleanup(func() {
		DB = previousDB
	})

	require.Equal(t, "unknown-code", GetGroupDisplayNameForError(" unknown-code "))
	require.Equal(t, "", GetGroupDisplayNameForError("  "))
}

func TestFormatGroupDisplayNamesMapsEachIdentifier(t *testing.T) {
	names := map[string]string{
		"Codex-Plus":   "Codex-Basic | 高性价比",
		"codex-pro特价":  "Codex-Value | 价格与性能综合",
		"legacy-basic": "Codex-Basic | 高性价比",
		"blank":        "  ",
	}

	require.Equal(t, "Codex-Basic | 高性价比", FormatGroupDisplayNames("Codex-Plus", names))
	require.Equal(t, "Codex-Basic | 高性价比, Codex-Value | 价格与性能综合", FormatGroupDisplayNames("Codex-Plus,codex-pro特价", names))
	require.Equal(t, "Codex-Basic | 高性价比, Codex-Value | 价格与性能综合", FormatGroupDisplayNames(" legacy-basic , codex-pro特价 ", names))
	require.Equal(t, "blank", FormatGroupDisplayNames("blank", names))
	require.Equal(t, "missing", FormatGroupDisplayNames("missing", names))
	require.Equal(t, "", FormatGroupDisplayNames("  ", names))
}

func TestApplyLogGroupNamesMapsCommaSeparatedCodes(t *testing.T) {
	names := map[string]string{
		"Codex-Plus":  "Codex-Basic | 高性价比",
		"codex-pro特价": "Codex-Value | 价格与性能综合",
	}
	logs := []*Log{
		{Group: "Codex-Plus,codex-pro特价"},
		{Group: "Codex-Plus"},
		{Group: "", Other: `{"group":"codex-pro特价,Codex-Plus"}`},
		{Group: "missing-code"},
	}

	applyLogGroupNames(logs, names)

	require.Equal(t, "Codex-Basic | 高性价比, Codex-Value | 价格与性能综合", logs[0].GroupName)
	require.Equal(t, "Codex-Basic | 高性价比", logs[1].GroupName)
	require.Equal(t, "Codex-Value | 价格与性能综合, Codex-Basic | 高性价比", logs[2].GroupName)
	require.Equal(t, "missing-code", logs[3].GroupName)
}
