# OpenAI 上游 Gemini 图片响应转换修复

## 目标与范围

修复客户端使用 Gemini `generateContent` 协议、渠道上游使用 OpenAI Chat
协议时的图片响应丢失：OpenAI 响应中的图片载体必须转换为 Gemini
`inlineData` 或 `fileData`，文本、工具调用和 usage metadata 保持不变。

本次只覆盖非流式 OpenAI Chat 响应转换。流式响应仍按原协议逐 chunk 将文本
转换为 Gemini `text`，不尝试跨 chunk 解析 Markdown 图片。

## 根因

`relay/channel/openai/relay-openai.go` 在目标格式为 Gemini 时调用
`relayconvert.ConvertResponse`。旧的
`relaykit/relayconvert/internal/oai_chat/to_gemini_chat_resp.go` 只读取
`choice.Message.StringContent()` 和工具调用：

- `content` 数组中的 `image_url` 被 `StringContent` 忽略；
- `message.images` 没有 DTO 字段，解码时被丢弃；
- Markdown 图片被当成普通文本；
- usage 在响应转换前已归一化，所以图片丢失仍可能正常计费。

## 转换契约

- `data:image/<type>;base64,<非空合法 Base64>` 转为 `inlineData`；不联网抓取。
- 语法有效的 `http://` 或 `https://` 图片 URL 转为 `fileData.fileUri`，MIME
  类型使用显式图片 MIME 或按 URL 扩展名推断；不联网抓取。
- `message.images` 支持字符串、`image_url.url`、`url`、`data` 和
  `b64_json` 载体；`b64_json` 也必须是非空合法 Base64。
- 普通文本、普通 URL、无效 data URL 和无效 HTTP URL 保持文本或被跳过，
  不制造图片 part。
- 文本、工具调用、finish reason 和 usage metadata 不改变。

请求侧的 Gemini `responseModalities` / `imageConfig` 不在本 PR 强行映射到
OpenAI Chat：OpenAI 标准 Chat 请求没有对应的图片字段。后续若某个上游定义
了明确兼容契约，应在该渠道适配器中单独实现并增加协议测试，不能把 Gemini
字段盲目透传给所有 OpenAI 上游。

## 验证

在隔离 Paseo worktree、`fix/openai-gemini-image-parts` 分支执行：

```powershell
cd relaykit
$env:GOWORK = "off"
$env:GOARCH = "amd64"
$env:GOPROXY = "off"
$env:GOTOOLCHAIN = "local"
go test ./relayconvert/...
go build ./...
```

回归覆盖内容数组图片、Markdown data/URL、`message.images`、非法载体、普通
文本与流式 Markdown 保持文本；离线画布验收 fixture 的五种图片载体各返回
一张图片，`usageMetadata` 与修复前 deep equal。

## 已知边界

- 没有取得生产故障请求的原始上游响应正文，因此不能把某一种 OpenAI 图片
  载体认定为该次生产请求的实际载体；生产证据只确认了 Gemini
  `generateContent` 到 OpenAI Compatible 的转换链。
- 未发送上游请求、未发付费生图请求、未修改生产配置或部署。
