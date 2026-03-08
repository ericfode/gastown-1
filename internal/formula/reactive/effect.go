package reactive

// Quality levels for LLM output. Maps to the Lean formalization (GasCity.lean Section 1).
type Quality int

const (
	QualityDraft    Quality = iota // Quick, cheap, possibly wrong.
	QualityAdequate                // Good enough for downstream consumption.
	QualityGood                    // Careful, considered output.
	QualityExcellent               // Best achievable, reviewed.
)

var qualityNames = map[Quality]string{
	QualityDraft:     "draft",
	QualityAdequate:  "adequate",
	QualityGood:      "good",
	QualityExcellent: "excellent",
}

var qualityFromString = map[string]Quality{
	"draft":     QualityDraft,
	"adequate":  QualityAdequate,
	"good":      QualityGood,
	"excellent": QualityExcellent,
}

func (q Quality) String() string {
	if s, ok := qualityNames[q]; ok {
		return s
	}
	return "unknown"
}

// ParseQuality converts a string to a Quality level.
func ParseQuality(s string) (Quality, bool) {
	q, ok := qualityFromString[s]
	return q, ok
}

// Min returns the lower of two quality levels.
func (q Quality) Min(other Quality) Quality {
	if q < other {
		return q
	}
	return other
}

// Effect describes the cost of an LLM computation.
// Maps to GasCity.lean Section 1: Effect = { tokens : Nat, quality : Quality }.
type Effect struct {
	Tokens  int     `json:"tokens" yaml:"tokens"`
	Quality Quality `json:"quality" yaml:"quality"`
}

// Zero returns the identity effect: no cost, maximum quality.
func EffectZero() Effect {
	return Effect{Tokens: 0, Quality: QualityExcellent}
}

// Seq composes effects sequentially: costs ADD, quality takes MINIMUM.
func (e Effect) Seq(other Effect) Effect {
	return Effect{
		Tokens:  e.Tokens + other.Tokens,
		Quality: e.Quality.Min(other.Quality),
	}
}

// Par composes effects in parallel: costs take MAX, quality takes MINIMUM.
func (e Effect) Par(other Effect) Effect {
	tokens := e.Tokens
	if other.Tokens > tokens {
		tokens = other.Tokens
	}
	return Effect{
		Tokens:  tokens,
		Quality: e.Quality.Min(other.Quality),
	}
}

// Fidelity represents information fidelity as an abstract preorder.
// Maps to GasCity.lean Section 14 (refactored).
//
// Laws:
//   - Preorder: Le is reflexive and transitive.
//   - Data Processing Inequality: Seq(a, b).Le(a) — sequential composition cannot increase fidelity.
//   - Best-path: a.Le(Par(a, b)) — parallel composition cannot decrease fidelity.
//   - Lossless is the top element.
type Fidelity int

// Lossless returns the top element: no information loss.
func Lossless() Fidelity { return Fidelity(0) }

// Le returns true if f ≤ other in the fidelity preorder.
// Lower underlying value = higher fidelity (Lossless is 0).
func (f Fidelity) Le(other Fidelity) bool { return f >= other }

// Seq composes fidelity sequentially. Monotone decreasing: Seq(a, b) ≤ a.
// Returns the worse (higher-valued) of the two.
func (f Fidelity) Seq(other Fidelity) Fidelity {
	if other > f {
		return other
	}
	return f
}

// Par composes fidelity in parallel. Monotone increasing: a ≤ Par(a, b).
// Returns the better (lower-valued) of the two.
func (f Fidelity) Par(other Fidelity) Fidelity {
	if other < f {
		return other
	}
	return f
}

// PipelineCost computes total cost for a sequence of effects.
func PipelineCost(effects []Effect) int {
	total := 0
	for _, e := range effects {
		total += e.Tokens
	}
	return total
}
