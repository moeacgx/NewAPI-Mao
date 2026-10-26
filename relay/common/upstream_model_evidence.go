package common

// UpstreamModelEvidence 保留上游模型声明的来源，不承诺识别底层模型身份。
type UpstreamModelEvidence struct {
	Model  string
	Source string
}
