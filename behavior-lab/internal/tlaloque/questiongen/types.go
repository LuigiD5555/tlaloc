// Package questiongen provides the GENERATE_PAGE_QUESTIONS Tlaloque: a
// bounded capability that produces test questions about a page's content.
package questiongen

const (
	Capability = "GENERATE_PAGE_QUESTIONS"

	TemplateWorkerID      = "questiongen-template"
	SemanticModelWorkerID = "questiongen-semantic-model"

	inputSchema  = "tlaloc.foldtest.r0.page-questions-input"
	outputSchema = "tlaloc.foldtest.r0.page-questions-output"

	// minModelQuestions is the fewest questions a model response must
	// contain to be accepted; fewer is treated as a worker failure so the
	// caller falls back to TemplateWorker instead of accepting a poor list.
	minModelQuestions = 2
)

// GenerateInput is the CapabilityRequest.Input payload for GENERATE_PAGE_QUESTIONS.
type GenerateInput struct {
	PageContent string `json:"page_content"`
	PageNumber  int    `json:"page_number"`
}

// GenerateOutput is the CapabilityResponse.Output payload for GENERATE_PAGE_QUESTIONS.
type GenerateOutput struct {
	Questions []string `json:"questions"`
}
