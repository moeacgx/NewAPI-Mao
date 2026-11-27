package model_setting

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalSettingsCanvasDefaultGroupExportsAndUpdates(t *testing.T) {
	assert.Empty(t, defaultOpenaiSettings.CanvasDefaultGroup)
	assert.Empty(t, defaultOpenaiSettings.CanvasTextDefaultGroup)
	assert.Empty(t, defaultOpenaiSettings.CanvasImageDefaultGroup)
	assert.Empty(t, defaultOpenaiSettings.CanvasAudioDefaultGroup)
	assert.Empty(t, defaultOpenaiSettings.CanvasVideoDefaultGroup)
	assert.Empty(t, config.GlobalConfig.ExportAllConfigs()["global.canvas_default_group"])
	assert.Empty(t, config.GlobalConfig.ExportAllConfigs()["global.canvas_text_default_group"])
	assert.Empty(t, config.GlobalConfig.ExportAllConfigs()["global.canvas_image_default_group"])
	assert.Empty(t, config.GlobalConfig.ExportAllConfigs()["global.canvas_audio_default_group"])
	assert.Empty(t, config.GlobalConfig.ExportAllConfigs()["global.canvas_video_default_group"])

	settings := GlobalSettings{}
	require.NoError(t, config.UpdateConfigFromMap(&settings, map[string]string{
		"canvas_default_group":       "vip",
		"canvas_text_default_group":  "text-code",
		"canvas_image_default_group": "image-code",
		"canvas_audio_default_group": "audio-code",
		"canvas_video_default_group": "video-code",
	}))
	assert.Equal(t, "vip", settings.CanvasDefaultGroup)
	assert.Equal(t, "text-code", settings.CanvasTextDefaultGroup)
	assert.Equal(t, "image-code", settings.CanvasImageDefaultGroup)
	assert.Equal(t, "audio-code", settings.CanvasAudioDefaultGroup)
	assert.Equal(t, "video-code", settings.CanvasVideoDefaultGroup)
}
