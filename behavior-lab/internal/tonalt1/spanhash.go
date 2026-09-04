package tonalt1

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

// The normalized source-span hash is a page-scoped physical-line identity.
// It deliberately does NOT hash text alone: identical text on different
// pages is NOT the same physical instance (protocol section 6/9). The hash
// binds the normalized containing-line text to its page and the token's
// rune span within that line.
//
// Normalization (SpanNormVersion), applied to the containing-line text
// (stdlib only — the behaviorlab module has no external dependencies):
//  1. Map a small fixed table of common typographic code points to ASCII
//     (curly quotes, dashes, ligatures, ellipsis).
//  2. Replace every Unicode space / control run with a single U+0020.
//  3. Trim leading/trailing spaces.
//  4. Lower-case.
//
// Letters and punctuation are otherwise preserved: over-normalization
// would collide unrelated physical spans. Case and whitespace are folded
// because the store's born-digital text layer is stable there and R1
// artifacts recorded the same line text with the same casing.

var typographicToASCII = map[rune]string{
	'‘': "'", '’': "'", '‚': "'", '‛': "'",
	'“': `"`, '”': `"`, '„': `"`, '‟': `"`,
	'–': "-", '—': "-", '‒': "-", '−': "-", '‐': "-", '‑': "-",
	'…': "...",
	'ﬀ': "ff", 'ﬁ': "fi", 'ﬂ': "fl", 'ﬃ': "ffi", 'ﬄ': "ffl",
}

// normalizeLineText canonicalizes a containing-line text for hashing and
// for line-equality comparisons in the prior-use matcher.
func normalizeLineText(text string) string {
	var builder strings.Builder
	builder.Grow(len(text))
	prevSpace := false
	for _, runeValue := range text {
		if replacement, ok := typographicToASCII[runeValue]; ok {
			for _, r := range replacement {
				writeNormRune(&builder, r, &prevSpace)
			}
			continue
		}
		writeNormRune(&builder, runeValue, &prevSpace)
	}
	return strings.TrimSpace(builder.String())
}

func writeNormRune(builder *strings.Builder, runeValue rune, prevSpace *bool) {
	if unicode.IsSpace(runeValue) || unicode.IsControl(runeValue) {
		if !*prevSpace {
			builder.WriteByte(' ')
			*prevSpace = true
		}
		return
	}
	builder.WriteRune(unicode.ToLower(runeValue))
	*prevSpace = false
}

// normalizedSpanHash returns the page-scoped physical-line identity hash.
// tokenStart/tokenEnd are rune offsets of the operand token in the
// ORIGINAL (pre-normalization) line text; they are folded in so that two
// different single-token lines that normalize identically stay distinct.
func normalizedSpanHash(page int, lineText string, tokenStart, tokenEnd int) string {
	payload := strings.Join([]string{
		SpanNormVersion,
		"p" + itoa(page),
		normalizeLineText(lineText),
		"tok" + itoa(tokenStart) + "-" + itoa(tokenEnd),
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

// lineIdentityHash is the page + normalized-line-text identity without the
// token span, used by the prior-use matcher to detect that a whole
// containing line was previously consumed even when token offsets differ
// between algorithm versions.
func lineIdentityHash(page int, lineText string) string {
	payload := strings.Join([]string{
		SpanNormVersion,
		"p" + itoa(page),
		normalizeLineText(lineText),
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var digits [20]byte
	index := len(digits)
	for n > 0 {
		index--
		digits[index] = byte('0' + n%10)
		n /= 10
	}
	if negative {
		index--
		digits[index] = '-'
	}
	return string(digits[index:])
}
