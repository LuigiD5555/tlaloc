package foldtest

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"

	"tlaloc.local/behaviorlab/internal/pdfmemory"
	"tlaloc.local/behaviorlab/internal/tlaloque/answerscore"
	"tlaloc.local/behaviorlab/internal/tlaloque/questiongen"
)

// ValidationConfig holds configuration for automated validation tests
type ValidationConfig struct {
	WorkDir             string
	StoreDir            string
	Manifest            pdfmemory.Manifest
	Cover               string
	Model               string
	BaseURL             string
	MaxTurns            int
	Budget              int
	RandomSeed          int64   // For reproducible page selection
	PageCount           int     // Total pages in document
	SampleSize          int     // Number of pages to sample (default 5)
	TimeoutSecs         int     // Timeout per question (default 30)
	FlexibilityScore    float64 // How lenient validation is (0.0-1.0, default 0.8)
	EmbeddingServiceURL string  // Optional; enables the resident embedding-similarity scorer when set
}

// ValidationTest represents a single test case
type ValidationTest struct {
	PageNumber           int               `json:"page_number"`
	PageContent          string            `json:"page_content"`
	Questions            []string          `json:"questions"`
	QuestionsGeneratedBy string            `json:"questions_generated_by,omitempty"`
	Answers              []string          `json:"model_answers"`
	Validations          []ValidationScore `json:"validations"`
	AverageScore         float64           `json:"average_score"`
	TimingMs             int64             `json:"timing_ms"`
}

// ValidationScore represents the quality of an answer
type ValidationScore struct {
	Question        string  `json:"question"`
	Answer          string  `json:"answer"`
	Score           float64 `json:"score"`      // 0.0-1.0
	Confidence      float64 `json:"confidence"` // Model confidence (0.0-1.0)
	KeywordsMatched int     `json:"keywords_matched"`
	KeywordsTotal   int     `json:"keywords_total"`
	Notes           string  `json:"notes,omitempty"`
	WorkerID        string  `json:"scored_by,omitempty"`
}

// ValidationResult aggregates all validation tests
type ValidationResult struct {
	Timestamp        string                     `json:"timestamp"`
	Model            string                     `json:"model"`
	TotalPages       int                        `json:"total_pages"`
	SampleSize       int                        `json:"sample_size"`
	SelectedPages    []int                      `json:"selected_pages"`
	Tests            []ValidationTest           `json:"tests"`
	AggregateMetrics AggregateValidationMetrics `json:"aggregate_metrics"`
	Status           string                     `json:"status"`
}

// AggregateValidationMetrics summarizes validation performance
type AggregateValidationMetrics struct {
	TotalTests      int     `json:"total_tests"`
	TotalQuestions  int     `json:"total_questions"`
	AverageScore    float64 `json:"average_score"`
	MedianScore     float64 `json:"median_score"`
	MinScore        float64 `json:"min_score"`
	MaxScore        float64 `json:"max_score"`
	AverageTiming   int64   `json:"average_timing_ms"`
	TotalTokens     int     `json:"total_tokens"`
	FailedTests     int     `json:"failed_tests"`
	PartialTests    int     `json:"partial_tests"`
	SuccessfulTests int     `json:"successful_tests"`
}

// SelectSpacedPages selects N pages randomly but spaced apart
func SelectSpacedPages(totalPages, numPages int, seed int64) []int {
	if numPages >= totalPages {
		// Return all pages if requesting more than available
		result := make([]int, totalPages)
		for i := 0; i < totalPages; i++ {
			result[i] = i + 1
		}
		return result
	}

	rng := rand.New(rand.NewSource(seed))
	spacing := totalPages / numPages
	selected := make([]int, 0, numPages)

	for i := 0; i < numPages; i++ {
		// Add random offset within this spacing segment
		offset := rng.Intn(spacing)
		pageNum := (i * spacing) + offset + 1
		if pageNum > totalPages {
			pageNum = totalPages - (numPages - i - 1)
		}
		selected = append(selected, pageNum)
	}

	// Ensure no duplicates and sort
	seen := make(map[int]bool)
	final := make([]int, 0)
	for _, p := range selected {
		if !seen[p] && p > 0 && p <= totalPages {
			seen[p] = true
			final = append(final, p)
		}
	}

	return final
}

// ExtractPageContent retrieves the actual content of a page, looking up its
// address from the manifest instead of guessing a carrier-specific format.
func ExtractPageContent(storeDir string, manifest pdfmemory.Manifest, pageNumber int) (string, error) {
	for _, page := range manifest.Pages {
		if page.Number != pageNumber {
			continue
		}
		packet, err := pdfmemory.Expand(storeDir, manifest, page.Address, "high", 4000)
		if err != nil {
			return "", fmt.Errorf("expanding page %d (%s): %w", pageNumber, page.Address, err)
		}
		if len(packet.Evidence) == 0 {
			return "", fmt.Errorf("no evidence returned for page %d (%s)", pageNumber, page.Address)
		}
		return packet.Evidence[0].Content, nil
	}
	return "", fmt.Errorf("page %d not found in manifest (%d pages)", pageNumber, len(manifest.Pages))
}

// RunValidationTest executes the full automated validation
func RunValidationTest(ctx context.Context, config ValidationConfig) (ValidationResult, error) {
	result := ValidationResult{
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Model:      config.Model,
		TotalPages: config.PageCount,
		SampleSize: config.SampleSize,
		Status:     "RUNNING",
		Tests:      []ValidationTest{},
	}

	if config.SampleSize == 0 {
		config.SampleSize = 5
	}
	if config.TimeoutSecs == 0 {
		config.TimeoutSecs = 30
	}
	if config.FlexibilityScore == 0.0 {
		config.FlexibilityScore = 0.8
	}

	// Select spaced pages
	seed := config.RandomSeed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	selectedPages := SelectSpacedPages(config.PageCount, config.SampleSize, seed)
	result.SelectedPages = selectedPages

	scorerRegistry := answerscore.NewRegistry(config.Model, config.BaseURL, config.EmbeddingServiceURL)
	questionRegistry := questiongen.NewRegistry(config.Model, config.BaseURL)

	// Run tests for each selected page
	totalScore := 0.0
	allTimings := int64(0)
	totalTokens := 0

	for _, pageNum := range selectedPages {
		pageStart := time.Now()

		// Extract actual content
		pageContent, err := ExtractPageContent(config.StoreDir, config.Manifest, pageNum)
		if err != nil {
			result.Tests = append(result.Tests, ValidationTest{
				PageNumber:   pageNum,
				AverageScore: 0.0,
				TimingMs:     int64(time.Since(pageStart).Milliseconds()),
			})
			result.AggregateMetrics.FailedTests++
			continue
		}

		// Generate questions via the GENERATE_PAGE_QUESTIONS Tlaloque
		// (semantic model generator, falling back to fixed templates).
		questionCtx, questionCancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSecs)*time.Second)
		questionOut, generatedBy, questionErr := questiongen.GenerateQuestions(questionCtx, questionRegistry, questiongen.GenerateInput{
			PageContent: pageContent,
			PageNumber:  pageNum,
		})
		questionCancel()
		if questionErr != nil {
			result.Tests = append(result.Tests, ValidationTest{
				PageNumber:   pageNum,
				PageContent:  pageContent,
				AverageScore: 0.0,
				TimingMs:     int64(time.Since(pageStart).Milliseconds()),
			})
			result.AggregateMetrics.FailedTests++
			continue
		}
		questions := questionOut.Questions

		// Ask model each question
		test := ValidationTest{
			PageNumber:           pageNum,
			PageContent:          pageContent,
			Questions:            questions,
			QuestionsGeneratedBy: generatedBy,
			Answers:              []string{},
			Validations:          []ValidationScore{},
		}

		pageScores := 0.0
		for _, question := range questions {
			// Set timeout context
			ctx2, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSecs)*time.Second)

			// Run session
			sessionConfig := SessionConfig{
				WorkDir:  config.WorkDir,
				StoreDir: config.StoreDir,
				Manifest: config.Manifest,
				Cover:    config.Cover,
				Model:    config.Model,
				BaseURL:  config.BaseURL,
				MaxTurns: config.MaxTurns,
				Budget:   config.Budget,
			}

			sessionResult, err := RunSession(ctx2, sessionConfig, question)
			cancel()

			if err != nil {
				continue
			}

			test.Answers = append(test.Answers, sessionResult.Answer)
			totalTokens += sessionResult.TotalTokensPrompt + sessionResult.TotalTokensCompletion

			// Score the answer via the SCORE_ANSWER_RELEVANCE Tlaloque
			// (semantic model judge, falling back to keyword overlap).
			// ctx2 was already cancelled above, so this gets its own timeout.
			scoreCtx, scoreCancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSecs)*time.Second)
			scoreOut, scoredBy, scoreErr := answerscore.ScoreAnswer(scoreCtx, scorerRegistry, answerscore.ScoreInput{
				Question:         question,
				ModelAnswer:      sessionResult.Answer,
				PageContent:      pageContent,
				FlexibilityScore: config.FlexibilityScore,
			})
			scoreCancel()
			if scoreErr != nil {
				continue
			}
			validation := ValidationScore{
				Question:        question,
				Answer:          sessionResult.Answer,
				Score:           scoreOut.Score,
				Confidence:      scoreOut.Confidence,
				KeywordsMatched: scoreOut.KeywordsMatched,
				KeywordsTotal:   scoreOut.KeywordsTotal,
				Notes:           scoreOut.Notes,
				WorkerID:        scoredBy,
			}
			test.Validations = append(test.Validations, validation)
			pageScores += validation.Score
		}

		// Calculate average for this page
		if len(test.Validations) > 0 {
			test.AverageScore = pageScores / float64(len(test.Validations))
			totalScore += test.AverageScore

			if test.AverageScore >= 0.7 {
				result.AggregateMetrics.SuccessfulTests++
			} else if test.AverageScore >= 0.5 {
				result.AggregateMetrics.PartialTests++
			} else {
				result.AggregateMetrics.FailedTests++
			}
		}

		test.TimingMs = int64(time.Since(pageStart).Milliseconds())
		allTimings += test.TimingMs

		result.Tests = append(result.Tests, test)
	}

	// Aggregate metrics
	result.AggregateMetrics.TotalTests = len(result.Tests)
	result.AggregateMetrics.TotalQuestions = len(result.SelectedPages) * 3 // ~3 questions per page

	if result.AggregateMetrics.TotalTests > 0 {
		result.AggregateMetrics.AverageScore = totalScore / float64(result.AggregateMetrics.TotalTests)
		result.AggregateMetrics.AverageTiming = allTimings / int64(result.AggregateMetrics.TotalTests)

		// Calculate min/max/median
		scores := make([]float64, 0)
		for _, test := range result.Tests {
			scores = append(scores, test.AverageScore)
		}

		if len(scores) > 0 {
			result.AggregateMetrics.MinScore = scores[0]
			result.AggregateMetrics.MaxScore = scores[0]
			for _, s := range scores {
				if s < result.AggregateMetrics.MinScore {
					result.AggregateMetrics.MinScore = s
				}
				if s > result.AggregateMetrics.MaxScore {
					result.AggregateMetrics.MaxScore = s
				}
			}

			// Rough median (not perfect but works)
			if len(scores)%2 == 0 {
				result.AggregateMetrics.MedianScore = (scores[len(scores)/2-1] + scores[len(scores)/2]) / 2
			} else {
				result.AggregateMetrics.MedianScore = scores[len(scores)/2]
			}
		}
	}

	result.AggregateMetrics.TotalTokens = totalTokens

	if result.AggregateMetrics.FailedTests > result.AggregateMetrics.SuccessfulTests {
		result.Status = "VALIDATION_FAILED"
	} else if result.AggregateMetrics.PartialTests > 0 {
		result.Status = "VALIDATION_PARTIAL"
	} else {
		result.Status = "VALIDATION_PASSED"
	}

	return result, nil
}

// ValidateAddressFormat checks if an address is valid format
func ValidateAddressFormat(address string) bool {
	// Basic validation: page:XXXXXX or block:doc/page-N/blocks/M
	if strings.HasPrefix(address, "page:") {
		re := regexp.MustCompile(`^page:\d+$`)
		return re.MatchString(address)
	}
	if strings.HasPrefix(address, "block:") {
		re := regexp.MustCompile(`^block:doc/page-\d+/blocks/\d+$`)
		return re.MatchString(address)
	}
	return false
}
