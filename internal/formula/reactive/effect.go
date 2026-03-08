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

// Distortion tracks information fidelity through the pipeline.
// Maps to GasCity.lean Section 14.
type Distortion struct {
	Retention   int `json:"retention" yaml:"retention"`     // 0-100: percentage of information preserved.
	Sensitivity int `json:"sensitivity" yaml:"sensitivity"` // Amplification factor (1 = no amplification).
}

// PerfectDistortion returns 100% retention, no amplification.
func PerfectDistortion() Distortion {
	return Distortion{Retention: 100, Sensitivity: 1}
}

// Seq composes distortions sequentially: retentions MULTIPLY, sensitivities MULTIPLY.
func (d Distortion) Seq(other Distortion) Distortion {
	return Distortion{
		Retention:   d.Retention * other.Retention / 100,
		Sensitivity: d.Sensitivity * other.Sensitivity,
	}
}

// Par composes distortions in parallel: take BEST retention, WORST sensitivity.
func (d Distortion) Par(other Distortion) Distortion {
	ret := d.Retention
	if other.Retention > ret {
		ret = other.Retention
	}
	sens := d.Sensitivity
	if other.Sensitivity > sens {
		sens = other.Sensitivity
	}
	return Distortion{Retention: ret, Sensitivity: sens}
}

// FullEffect combines cost accounting with distortion tracking.
// Maps to GasCity.lean Section 14: FullCellEffect.
type FullEffect struct {
	Cost       Effect     `json:"cost" yaml:"cost"`
	Distortion Distortion `json:"distortion" yaml:"distortion"`
}

// Seq composes full effects sequentially.
func (fe FullEffect) Seq(other FullEffect) FullEffect {
	return FullEffect{
		Cost:       fe.Cost.Seq(other.Cost),
		Distortion: fe.Distortion.Seq(other.Distortion),
	}
}

// Par composes full effects in parallel.
func (fe FullEffect) Par(other FullEffect) FullEffect {
	return FullEffect{
		Cost:       fe.Cost.Par(other.Cost),
		Distortion: fe.Distortion.Par(other.Distortion),
	}
}

// PipelineCost computes total cost for a sequence of effects.
func PipelineCost(effects []Effect) int {
	total := 0
	for _, e := range effects {
		total += e.Tokens
	}
	return total
}

// PipelineRetention computes end-to-end retention for a sequence of distortions.
func PipelineRetention(distortions []Distortion) int {
	if len(distortions) == 0 {
		return 100
	}
	ret := distortions[0]
	for _, d := range distortions[1:] {
		ret = ret.Seq(d)
	}
	return ret.Retention
}
