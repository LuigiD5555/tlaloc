package calibration

import "strings"

// Query is a single runtime classification/decoding the caller wants a
// verdict for: the specialist's stated confidence, and (optionally) the
// domain the input belongs to.
type Query struct {
	Confidence float64
	Domain     string
}

// Verdict decides whether a specialist's prediction may be trusted for this
// query. Order matters: an explicit unsupported domain wins over a missing
// allow-list entry, which wins over a low confidence, which wins over
// missing OOD evidence.
func (profile CalibrationProfile) Verdict(query Query) AbstentionVerdict {
	domain := strings.TrimSpace(strings.ToLower(query.Domain))
	if domain != "" {
		for _, unsupported := range profile.UnsupportedDomains {
			if strings.ToLower(strings.TrimSpace(unsupported)) == domain {
				return Unsupported
			}
		}
		if len(profile.SupportedDomains) > 0 {
			supported := false
			for _, candidate := range profile.SupportedDomains {
				if strings.ToLower(strings.TrimSpace(candidate)) == domain {
					supported = true
					break
				}
			}
			if !supported {
				return Unsupported
			}
		}
	}

	if query.Confidence < profile.ConfidenceFloor {
		return LowEvidence
	}

	// A model with no out-of-distribution measurement has no basis to
	// claim competence on anything but its exact training distribution.
	if profile.OutOfDistribution.N == 0 {
		return Unknown
	}

	return Answered
}

// Admission thresholds. Deliberately strict: the point of the swarm is
// many specialists that are trustworthy inside a narrow lane, not many
// that answer confidently everywhere.
const (
	// MinOODSamples is the smallest out-of-distribution eval set that
	// counts as "measured".
	MinOODSamples = 100
	// MaxInDistributionECE is the calibration-error ceiling on the
	// in-distribution slice.
	MaxInDistributionECE = 0.10
	// MinOODAccuracy is the accuracy floor on the out-of-distribution
	// slice — below this the specialist does not generalize enough to be
	// ACTIVE, even if it is perfect in-distribution.
	MinOODAccuracy = 0.80
	// MinAbstentionCurvePoints ensures an abstention policy actually
	// exists for this specialist.
	MinAbstentionCurvePoints = 3
)

// AdmitAsActive is the gate a learned Tlaloque must clear to be promoted
// from CANDIDATE/SHADOW to ACTIVE. It returns every failed requirement, so
// a caller can report exactly why a model is being held back.
func (profile CalibrationProfile) AdmitAsActive() (admitted bool, reasons []string) {
	if profile.Schema != Schema {
		reasons = append(reasons, "profile schema is not "+Schema)
	}
	if profile.OutOfDistribution.N < MinOODSamples {
		reasons = append(reasons, "out-of-distribution slice is not measured on enough samples")
	}
	if len(profile.AbstentionCurve) < MinAbstentionCurvePoints {
		reasons = append(reasons, "abstention curve has too few points to form a policy")
	}
	if profile.InDistribution.ECE > MaxInDistributionECE {
		reasons = append(reasons, "in-distribution calibration error is above the ceiling")
	}
	if profile.OutOfDistribution.N >= MinOODSamples && profile.OutOfDistribution.Accuracy < MinOODAccuracy {
		reasons = append(reasons, "out-of-distribution accuracy is below the floor")
	}
	if profile.ConfidenceFloor <= 0 || profile.ConfidenceFloor >= 1 {
		reasons = append(reasons, "confidence floor must be set within (0, 1)")
	}
	return len(reasons) == 0, reasons
}
