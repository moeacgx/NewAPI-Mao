package dto

// ReasoningConversionState carries provider-neutral reasoning controls between
// request parsing and a channel adaptor. It is internal metadata and must not
// be serialized into an upstream request.
type ReasoningConversionState struct {
	Mode         string
	Effort       string
	BudgetTokens *int
}
