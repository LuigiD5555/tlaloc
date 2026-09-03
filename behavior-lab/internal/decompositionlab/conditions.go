package decompositionlab

// Condition names the four T0-A oracle conditions (section 17) and three
// T0-B real conditions (section 24). This is a fixed, bounded vocabulary
// (E0.13, E0.14): T0 compiles exactly these seven recipes and nothing else,
// never a free-form plan.
type Condition string

const (
	ConditionC0ParrotDirect Condition = "C0_PARROT_DIRECT"
	ConditionC1OracleCrop   Condition = "C1_ORACLE_CROP_PARROT"
	ConditionC2Normalize    Condition = "C2_ORACLE_CROP_PARROT_NORMALIZE"
	ConditionC3Verify       Condition = "C3_ORACLE_CROP_PARROT_NORMALIZE_VERIFY"

	ConditionB1RealCrop  Condition = "B1_REAL_REGION_PARROT"
	ConditionB2Normalize Condition = "B2_REAL_REGION_PARROT_NORMALIZE"
	ConditionB3Verify    Condition = "B3_REAL_REGION_PARROT_NORMALIZE_VERIFY"
)

// ExternalizedCapabilities is the section 17/24 capability count each
// condition externalizes from Parrot — the x-axis of the substitution
// curve (section 8, section 18's primary graph).
func (c Condition) ExternalizedCapabilities() int {
	switch c {
	case ConditionC0ParrotDirect:
		return 0
	case ConditionC1OracleCrop, ConditionB1RealCrop:
		return 1
	case ConditionC2Normalize, ConditionB2Normalize:
		return 2
	case ConditionC3Verify, ConditionB3Verify:
		return 3
	default:
		return -1
	}
}

// IsOracle reports whether a condition belongs to the T0-A oracle track
// (uses a frozen ground-truth address only to select geometry) as opposed
// to the T0-B real track (uses only legitimate runtime information).
func (c Condition) IsOracle() bool {
	switch c {
	case ConditionC0ParrotDirect, ConditionC1OracleCrop, ConditionC2Normalize, ConditionC3Verify:
		return true
	default:
		return false
	}
}

// UsesLocateRegion reports whether a condition externalizes localization
// at all. C0 does not: Parrot receives the original full page.
func (c Condition) UsesLocateRegion() bool {
	return c != ConditionC0ParrotDirect
}

// UsesNormalize reports whether a condition routes Parrot's raw text
// through the deterministic Normalize Tlaloque before scoring.
func (c Condition) UsesNormalize() bool {
	switch c {
	case ConditionC2Normalize, ConditionC3Verify, ConditionB2Normalize, ConditionB3Verify:
		return true
	default:
		return false
	}
}

// UsesVerify reports whether a condition promotes the (possibly
// normalized) result through the Verify Tlaloque before scoring.
func (c Condition) UsesVerify() bool {
	switch c {
	case ConditionC3Verify, ConditionB3Verify:
		return true
	default:
		return false
	}
}

// AllConditions lists every T0 condition in the fixed report order
// (section 3 "COMPARE FOUR CONDITIONS", section 24 "MIRROR").
func AllConditions() []Condition {
	return []Condition{
		ConditionC0ParrotDirect, ConditionC1OracleCrop, ConditionC2Normalize, ConditionC3Verify,
		ConditionB1RealCrop, ConditionB2Normalize, ConditionB3Verify,
	}
}
