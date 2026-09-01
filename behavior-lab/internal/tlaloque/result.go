package tlaloque

// ResultCode describes an expected domain outcome. Infrastructure failures and
// broken contracts continue to use Go errors; these codes are for outcomes the
// orchestration layer is expected to reason about.
type ResultCode string

const (
	ResultSuccess         ResultCode = "SUCCESS"
	ResultPartial         ResultCode = "PARTIAL"
	ResultConflict        ResultCode = "CONFLICT"
	ResultNoCandidate     ResultCode = "NO_CANDIDATE"
	ResultNoQuorum        ResultCode = "NO_QUORUM"
	ResultResidual        ResultCode = "RESIDUAL"
	ResultBudgetExhausted ResultCode = "BUDGET_EXHAUSTED"
	ResultInvalidRequest  ResultCode = "INVALID_REQUEST"
)

// Diagnostic is structured context for a normal domain outcome. It deliberately
// avoids overloading error strings with machine-readable control flow.
type Diagnostic struct {
	Code    string         `json:"code"`
	Message string         `json:"message,omitempty"`
	Fields  map[string]any `json:"fields,omitempty"`
}

// Result carries an expected orchestration outcome. Err remains reserved for
// exceptional failures at the compatibility boundary; new orchestration code
// should branch on Code rather than parse error text.
type Result[T any] struct {
	Code        ResultCode   `json:"code"`
	Value       T            `json:"value,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics,omitempty"`
	Err         error        `json:"-"`
}

func Success[T any](value T) Result[T] {
	return Result[T]{Code: ResultSuccess, Value: value}
}

func DomainResult[T any](code ResultCode, value T, diagnostics ...Diagnostic) Result[T] {
	return Result[T]{Code: code, Value: value, Diagnostics: diagnostics}
}

func Failure[T any](err error) Result[T] {
	return Result[T]{Err: err}
}

func (r Result[T]) OK() bool {
	return r.Err == nil && r.Code == ResultSuccess
}
