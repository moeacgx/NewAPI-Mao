// Package skillstore 提供随仓库版本发布的技能商店文本快照。
package skillstore

import "embed"

// Files 仅收录公开的原创技能示例；发布后站点可在线刷新同目录。
//go:embed index.json packages/*/*/SKILL.md
var Files embed.FS
