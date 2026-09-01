package swarmbench

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const dateLayout = "2006-01-02"

var spanishWeekdays = map[string]time.Weekday{
	"lunes":     time.Monday,
	"martes":    time.Tuesday,
	"miercoles": time.Wednesday,
	"miércoles": time.Wednesday,
	"jueves":    time.Thursday,
	"viernes":   time.Friday,
	"sabado":    time.Saturday,
	"sábado":    time.Saturday,
	"domingo":   time.Sunday,
}

// datePattern binds one bounded surface form to its exact resolution. Ordering
// matters: more specific forms are tried first.
type datePattern struct {
	expression *regexp.Regexp
	resolve    func(match []string, reference time.Time) (time.Time, error)
}

var datePatterns = []datePattern{
	{
		expression: regexp.MustCompile(`(?i)el pr[oó]ximo (\p{L}+)`),
		resolve: func(match []string, reference time.Time) (time.Time, error) {
			return shiftToWeekday(match[1], reference, +1)
		},
	},
	{
		expression: regexp.MustCompile(`(?i)el (\p{L}+) pasado`),
		resolve: func(match []string, reference time.Time) (time.Time, error) {
			return shiftToWeekday(match[1], reference, -1)
		},
	},
	{
		expression: regexp.MustCompile(`(?i)en (\d+) d[ií]as?`),
		resolve: func(match []string, reference time.Time) (time.Time, error) {
			days, err := strconv.Atoi(match[1])
			if err != nil {
				return time.Time{}, err
			}
			return reference.AddDate(0, 0, days), nil
		},
	},
	{
		expression: regexp.MustCompile(`(?i)hace (\d+) d[ií]as?`),
		resolve: func(match []string, reference time.Time) (time.Time, error) {
			days, err := strconv.Atoi(match[1])
			if err != nil {
				return time.Time{}, err
			}
			return reference.AddDate(0, 0, -days), nil
		},
	},
	{
		expression: regexp.MustCompile(`(?i)a fin de mes`),
		resolve: func(_ []string, reference time.Time) (time.Time, error) {
			firstOfNextMonth := time.Date(reference.Year(), reference.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 1, 0)
			return firstOfNextMonth.AddDate(0, 0, -1), nil
		},
	},
	{
		expression: regexp.MustCompile(`(?i)la semana pasada`),
		resolve: func(_ []string, reference time.Time) (time.Time, error) {
			return reference.AddDate(0, 0, -7), nil
		},
	},
	{
		expression: regexp.MustCompile(`(?i)\bayer\b`),
		resolve: func(_ []string, reference time.Time) (time.Time, error) {
			return reference.AddDate(0, 0, -1), nil
		},
	},
	{
		expression: regexp.MustCompile(`(?i)ma[ñn]ana`),
		resolve: func(_ []string, reference time.Time) (time.Time, error) {
			return reference.AddDate(0, 0, 1), nil
		},
	},
	{
		expression: regexp.MustCompile(`(?i)\bhoy\b`),
		resolve: func(_ []string, reference time.Time) (time.Time, error) {
			return reference, nil
		},
	},
}

// shiftToWeekday walks day by day in `direction` until the named weekday is
// reached, strictly excluding the reference day itself.
func shiftToWeekday(name string, reference time.Time, direction int) (time.Time, error) {
	weekday, ok := spanishWeekdays[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return time.Time{}, fmt.Errorf("unknown weekday %q", name)
	}
	candidate := reference
	for step := 0; step < 7; step++ {
		candidate = candidate.AddDate(0, 0, direction)
		if candidate.Weekday() == weekday {
			return candidate, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not reach weekday %q", name)
}

// ResolveDate turns one bounded Spanish date expression into an ISO date. It
// carries no parameters and is exact, which is the categorical advantage a
// deterministic Tlaloque holds over a small language model on this subtask.
func ResolveDate(expression string, referenceISO string) (string, error) {
	reference, err := time.Parse(dateLayout, strings.TrimSpace(referenceISO))
	if err != nil {
		return "", fmt.Errorf("resolve date: unparsable reference %q: %w", referenceISO, err)
	}
	for _, pattern := range datePatterns {
		match := pattern.expression.FindStringSubmatch(expression)
		if match == nil {
			continue
		}
		resolved, err := pattern.resolve(match, reference)
		if err != nil {
			return "", fmt.Errorf("resolve date %q: %w", expression, err)
		}
		return resolved.Format(dateLayout), nil
	}
	return "", fmt.Errorf("resolve date: unsupported expression %q", expression)
}

// ExtractDate locates a known date expression anywhere in a document and
// resolves it. This is what the deterministic date Tlaloque runs.
func ExtractDate(text string, referenceISO string) (string, error) {
	reference, err := time.Parse(dateLayout, strings.TrimSpace(referenceISO))
	if err != nil {
		return "", fmt.Errorf("extract date: unparsable reference %q: %w", referenceISO, err)
	}
	for _, pattern := range datePatterns {
		match := pattern.expression.FindStringSubmatch(text)
		if match == nil {
			continue
		}
		resolved, err := pattern.resolve(match, reference)
		if err != nil {
			return "", fmt.Errorf("extract date: %w", err)
		}
		return resolved.Format(dateLayout), nil
	}
	return "", fmt.Errorf("extract date: no known expression in %q", text)
}

// amountPattern requires an explicit currency marker so that quantities such
// as "en 30 dias" are never mistaken for money.
var amountPattern = regexp.MustCompile(`(?i)(?:\$|MXN)\s*([\d,]+(?:\.\d{1,2})?)`)

// ExtractAmount recovers the monetary amount in cents. Like ExtractDate it is
// exact and parameter-free.
func ExtractAmount(text string) (int64, error) {
	match := amountPattern.FindStringSubmatch(text)
	if match == nil {
		return 0, fmt.Errorf("extract amount: no currency amount in %q", text)
	}
	return ParseAmountCents(match[1])
}

// ParseAmountCents converts a Spanish-formatted amount body such as
// "12,450.00" into cents.
func ParseAmountCents(body string) (int64, error) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(body), ",", "")
	whole, fraction, hasFraction := strings.Cut(cleaned, ".")
	units, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse amount %q: %w", body, err)
	}
	cents := units * 100
	if hasFraction {
		if len(fraction) == 1 {
			fraction += "0"
		}
		if len(fraction) > 2 {
			return 0, fmt.Errorf("parse amount %q: too many decimals", body)
		}
		fractional, err := strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("parse amount %q: %w", body, err)
		}
		cents += fractional
	}
	return cents, nil
}

// FormatAmount renders cents back into the surface form the documents use.
func FormatAmount(cents int64) string {
	units := cents / 100
	fraction := cents % 100
	return fmt.Sprintf("$%s.%02d", groupThousands(units), fraction)
}

func groupThousands(value int64) string {
	digits := strconv.FormatInt(value, 10)
	if len(digits) <= 3 {
		return digits
	}
	var builder strings.Builder
	lead := len(digits) % 3
	if lead > 0 {
		builder.WriteString(digits[:lead])
	}
	for index := lead; index < len(digits); index += 3 {
		if builder.Len() > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(digits[index : index+3])
	}
	return builder.String()
}
