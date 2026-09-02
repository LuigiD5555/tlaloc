# Fold/Unfold Validation Test Suite

## Overview

The validation test suite provides **automated, rigorous content verification** for PDF fold/unfold harness. Unlike simple cover-only testing, validation:

1. **Randomly selects spaced pages** across the full document
2. **Extracts real page content** from the store
3. **Generates questions** based on actual content
4. **Compares model answers** against ground truth
5. **Scores systematically** with keyword/length/semantic checks
6. **Produces JSON metrics** for CI/tracking

## Architecture

### Components

```
ValidationConfig
├── SelectSpacedPages(total, numPages, seed) → []int
├── ExtractPageContent(store, manifest, page) → string
├── GeneratePageQuestions(content, page) → []string
├── ValidateAnswer(q, answer, content, flex) → ValidationScore
└── RunValidationTest(ctx, config) → ValidationResult
```

### Data Flow

```
PDF Store
    ↓
SelectSpacedPages [page 44, 144, 225, 317, 362]
    ↓
For each page:
  ├─ ExtractPageContent → "actual text from page..."
  ├─ GeneratePageQuestions → ["Q1", "Q2", "Q3"]
  ├─ For each question:
  │   ├─ RunSession (call model with cover)
  │   └─ ValidateAnswer (compare answer vs real content)
  └─ Aggregate scores
    ↓
ValidationResult JSON
```

## Scoring Algorithm

Each answer receives a score from 0.0-1.0 based on:

### 1. Keyword Matching (40%)
```
keywords_score = matched_keywords / total_keywords
```
- Extract important terms (>4 chars) from real page content
- Count how many appear in model's answer
- Higher match = higher score

### 2. Answer Length (30%)
```
length_score = {
  0.0 if answer is empty
  0.6 if answer < 10% of page length (too short)
  1.0 if answer is similar length to page (good)
  0.7 if answer > 2× page length (hallucinating)
}
```

### 3. Semantic Similarity (30%)
```
semantic_score = {
  1.0 if keywords_matched > 0
  0.3 if keywords_matched == 0 (completely wrong topic)
}
```

### 4. Apply Flexibility
```
flexibility_score = base_score * flexibility_factor + 
                    (1 - flexibility_factor) * 0.5

Default flexibility = 0.8:
  • strict validation: 0.5-0.6
  • balanced: 0.75-0.85
  • lenient: 0.9-1.0
```

## CLI Usage

### Basic Test
```bash
/tmp/tlaloc-fold-bench validate \
  -store /tmp/foldstore-swarms \
  -model lfm2-vl-1.6b
```
→ Tests 5 random pages, outputs to stdout

### Production Grade
```bash
/tmp/tlaloc-fold-bench validate \
  -store /tmp/foldstore-swarms \
  -model lfm2-vl-1.6b \
  -pages 10 \
  -seed 12345 \
  -flexibility 0.6 \
  -turns 4 \
  -url http://127.0.0.1:1234/v1 \
  -out validation-$(date +%Y%m%d).json
```

### CI Integration
```bash
#!/bin/bash
result=$(/tmp/tlaloc-fold-bench validate \
  -store /tmp/store \
  -model lfm2-vl-1.6b \
  -pages 5 \
  -seed 42 \
  -flexibility 0.75 \
  -out /tmp/validation.json)

score=$(jq '.aggregate_metrics.average_score' /tmp/validation.json)
if (( $(echo "$score < 0.7" | bc -l) )); then
  echo "FAIL: Validation score $score < 0.7"
  exit 1
fi
echo "PASS: Validation score $score"
```

## Parameters

| Flag | Default | Range | Purpose |
|------|---------|-------|---------|
| `-store` | — | path | Store directory (required) |
| `-model` | lfm2-vl-1.6b | string | Model name at LM Studio |
| `-pages` | 5 | 1-404 | Number of pages to sample |
| `-seed` | 0 (now) | int64 | Random seed (0 = use time) |
| `-flexibility` | 0.8 | 0.0-1.0 | Scoring strictness |
| `-turns` | 6 | 1+ | Max conversation turns |
| `-url` | http://127.0.0.1:1234/v1 | URL | LM Studio endpoint |
| `-out` | stdout | path | JSON output file |

## Output Format

### Top-Level
```json
{
  "timestamp": "2026-09-01T22:47:55Z",
  "model": "lfm2-vl-1.6b",
  "total_pages": 404,
  "sample_size": 5,
  "selected_pages": [44, 144, 225, 317, 362],
  "tests": [...],
  "aggregate_metrics": {...},
  "status": "VALIDATION_PASSED|PARTIAL|FAILED"
}
```

### Per-Page Test
```json
{
  "page_number": 44,
  "page_content": "Lorem ipsum...",
  "questions": ["Q1?", "Q2?", "Q3?"],
  "model_answers": ["A1", "A2", "A3"],
  "validations": [
    {
      "question": "Q1?",
      "answer": "A1",
      "score": 0.85,
      "confidence": 0.9,
      "keywords_matched": 7,
      "keywords_total": 10,
      "notes": "Answer matches content well"
    }
  ],
  "average_score": 0.82,
  "timing_ms": 18750
}
```

### Aggregate Metrics
```json
{
  "total_tests": 5,
  "total_questions": 15,
  "average_score": 0.81,
  "median_score": 0.82,
  "min_score": 0.65,
  "max_score": 0.91,
  "average_timing_ms": 15600,
  "total_tokens": 6850,
  "failed_tests": 0,
  "partial_tests": 1,
  "successful_tests": 4
}
```

## Status Meanings

| Status | Condition | Interpretation |
|--------|-----------|-----------------|
| `VALIDATION_PASSED` | All tests avg >= 0.7 | ✓ Model correctly understands content |
| `VALIDATION_PARTIAL` | Mixed pass/fail | ⚠ Model struggles with some topics |
| `VALIDATION_FAILED` | Most tests < 0.7 | ✗ Cover/model insufficient |

## Use Cases

### 1. Development/Debugging
```bash
# Test quickly with lenient scoring
/tmp/tlaloc-fold-bench validate -store /tmp/store -model lfm2-vl-1.6b \
  -pages 3 -flexibility 0.9 -turns 2
```
**Purpose:** Rapid iteration, understand failures

### 2. Regression Testing (CI)
```bash
# Reproducible, moderate strictness
/tmp/tlaloc-fold-bench validate -store /tmp/store -model lfm2-vl-1.6b \
  -pages 5 -seed 42 -flexibility 0.75 -out test-results.json
```
**Purpose:** Catch regressions in cover generation or model behavior

### 3. Production Validation
```bash
# Comprehensive, strict
/tmp/tlaloc-fold-bench validate -store /tmp/store -model lfm2-vl-1.6b \
  -pages 10 -seed 999 -flexibility 0.6 -out prod-validation.json
```
**Purpose:** Before releasing, validate model + cover work correctly

### 4. Model Comparison
```bash
for model in lfm2-vl-1.6b llama-7b gemma-7b; do
  /tmp/tlaloc-fold-bench validate -store /tmp/store -model $model \
    -seed 42 -out results-$model.json
done
# Compare: jq '.aggregate_metrics.average_score' results-*.json
```
**Purpose:** Benchmark different models

## Implementation Details

### Page Selection
- Uses `math/rand` with seed for reproducibility
- Divides document into N segments, picks one page per segment
- Result: Evenly distributed across document (not random clustering)

### Content Extraction
- Calls `pdfmemory.Expand()` with address format
- Tries multiple address formats for compatibility
- Returns page content + metadata from Packet

### Question Generation
- Creates 3 questions per page:
  1. "What is the main content of page N?"
  2. "What technical terms/concepts appear on page N?"
  3. Topic-specific: "How does [keyword] relate to page N?"

### Answer Scoring
- Extracts keywords (len > 4) from page content
- Counts keyword matches in answer
- Checks answer length against expected range
- Applies flexibility factor for configurable strictness

## Testing

### Unit Tests
```bash
cd /mnt/.../tlaloc/behavior-lab
go test ./internal/foldtest -v -run Validation
```

**Coverage:**
- `TestSelectSpacedPages` — Spacing distribution
- `TestExtractKeywords` — Keyword extraction logic
- `TestEstimateConfidence` — Confidence parsing
- `TestValidateAnswer` — Scoring formula
- `TestValidateAddressFormat` — Address formats
- `TestGeneratePageQuestions` — Question generation

### Integration Test
```bash
# Real PDF with real model
/tmp/tlaloc-fold-bench validate \
  -store /tmp/foldstore-swarms \
  -model lfm2-vl-1.6b \
  -pages 3 \
  -seed 42
```

## Known Limitations

1. **Address format** — Currently tries multiple formats; best to standardize
2. **Question generation** — Basic templates; could use NLP for better questions
3. **Scoring heuristic** — Keyword-based; misses semantic understanding
4. **Confidence extraction** — Pattern-based; may miss subtle language
5. **Page extraction** — If ExtractPageContent fails, page is marked failed

## Future Improvements

- [ ] LLM-based question generation (more semantic)
- [ ] Similarity scoring (embedding-based keyword matching)
- [ ] Hierarchical validation (validate cover → unfold → validate unfold)
- [ ] Partial credit for partially correct answers
- [ ] Per-model calibration of flexibility factor
- [ ] Visualization of results (score heatmap by page)
- [ ] Batch processing (multiple PDFs, multiple models)

## Troubleshooting

### "Could not extract page X"
**Cause:** Page number out of range or address format issue
**Fix:** Check page count in manifest, verify address format

### "Model call failed: connection refused"
**Cause:** LM Studio not running
**Fix:** Start LM Studio, verify endpoint: `curl http://127.0.0.1:1234/v1/models`

### Scores consistently < 0.5
**Cause:** Model not following instructions, or cover text too minimal
**Fix:** 
- Increase `-flexibility` to 0.9 (testing)
- Check if model is loaded and responsive
- Review cover text for sufficient content

### "No keywords found in page"
**Cause:** Page is image-only or extraction failed
**Fix:** Verify page has text content in store

## Files

```
internal/foldtest/
├── validation.go          Core validation engine
├── validation_test.go     8 unit tests
├── session.go             (existing)
├── cover.go               (existing)
└── unfold.go              (existing)

cmd/tlaloc-fold-bench/
└── main.go                Updated with cmdValidate()
```

## References

- [Session Management](internal/foldtest/session.go)
- [Cover Text Generation](internal/foldtest/cover.go)
- [PDF Memory Store](internal/pdfmemory/)
