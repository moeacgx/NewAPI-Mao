package constant

type TaskPlatform string

const (
	TaskPlatformSuno        TaskPlatform = "suno"
	TaskPlatformMidjourney               = "mj"
	TaskPlatformCanvasImage              = "canvas_image"
	TaskPlatformImage                    = "image"
)

// ImageTaskPlatforms returns a fresh slice of local async image wrapper platforms.
func ImageTaskPlatforms() []TaskPlatform {
	return []TaskPlatform{TaskPlatformCanvasImage, TaskPlatformImage}
}

func IsImageTaskPlatform(platform TaskPlatform) bool {
	return platform == TaskPlatformCanvasImage || platform == TaskPlatformImage
}

const (
	SunoActionMusic  = "MUSIC"
	SunoActionLyrics = "LYRICS"

	TaskActionGenerate          = "generate"
	TaskActionTextGenerate      = "textGenerate"
	TaskActionFirstTailGenerate = "firstTailGenerate"
	TaskActionReferenceGenerate = "referenceGenerate"
	TaskActionRemix             = "remixGenerate"
)

var SunoModel2Action = map[string]string{
	"suno_music":  SunoActionMusic,
	"suno_lyrics": SunoActionLyrics,
}

// NormalizeTaskAction 仅供官方插件边界使用，不改写原生历史 action。
func NormalizeTaskAction(action string) string {
	switch action {
	case "generate":
		return "image_to_video"
	case "textGenerate":
		return "text_to_video"
	case "firstTailGenerate":
		return "first_tail_to_video"
	case "referenceGenerate":
		return "reference_to_video"
	case "remixGenerate":
		return "remix"
	}
	return action
}
