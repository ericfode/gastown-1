package reactive

import "testing"

func TestEffectSeq(t *testing.T) {
	a := Effect{Tokens: 5000, Quality: QualityGood}
	b := Effect{Tokens: 10000, Quality: QualityAdequate}
	got := a.Seq(b)

	if got.Tokens != 15000 {
		t.Errorf("tokens = %d, want 15000", got.Tokens)
	}
	if got.Quality != QualityAdequate {
		t.Errorf("quality = %s, want adequate", got.Quality)
	}
}

func TestEffectPar(t *testing.T) {
	a := Effect{Tokens: 5000, Quality: QualityGood}
	b := Effect{Tokens: 10000, Quality: QualityAdequate}
	got := a.Par(b)

	if got.Tokens != 10000 {
		t.Errorf("tokens = %d, want 10000", got.Tokens)
	}
	if got.Quality != QualityAdequate {
		t.Errorf("quality = %s, want adequate", got.Quality)
	}
}

func TestEffectZero(t *testing.T) {
	zero := EffectZero()
	a := Effect{Tokens: 5000, Quality: QualityGood}

	left := zero.Seq(a)
	if left != a {
		t.Errorf("zero.Seq(a) = %+v, want %+v", left, a)
	}

	right := a.Seq(zero)
	if right != a {
		t.Errorf("a.Seq(zero) = %+v, want %+v", right, a)
	}
}

func TestEffectParLeSeq(t *testing.T) {
	a := Effect{Tokens: 5000, Quality: QualityGood}
	b := Effect{Tokens: 10000, Quality: QualityAdequate}

	par := a.Par(b)
	seq := a.Seq(b)

	if par.Tokens > seq.Tokens {
		t.Errorf("par.Tokens (%d) > seq.Tokens (%d), violates par_le_seq", par.Tokens, seq.Tokens)
	}
}

func TestDistortionSeq(t *testing.T) {
	a := Distortion{Retention: 60, Sensitivity: 2}
	b := Distortion{Retention: 20, Sensitivity: 1}
	got := a.Seq(b)

	if got.Retention != 12 {
		t.Errorf("retention = %d, want 12 (60*20/100)", got.Retention)
	}
	if got.Sensitivity != 2 {
		t.Errorf("sensitivity = %d, want 2", got.Sensitivity)
	}
}

func TestDistortionPar(t *testing.T) {
	a := Distortion{Retention: 60, Sensitivity: 2}
	b := Distortion{Retention: 80, Sensitivity: 1}
	got := a.Par(b)

	if got.Retention != 80 {
		t.Errorf("retention = %d, want 80 (max)", got.Retention)
	}
	if got.Sensitivity != 2 {
		t.Errorf("sensitivity = %d, want 2 (max)", got.Sensitivity)
	}
}

func TestPipelineRetention(t *testing.T) {
	// 80% * 60% * 70% * 20% = 6.72% ≈ 6% (integer truncation)
	distortions := []Distortion{
		{Retention: 80, Sensitivity: 1},
		{Retention: 60, Sensitivity: 2},
		{Retention: 70, Sensitivity: 1},
		{Retention: 20, Sensitivity: 1},
	}
	got := PipelineRetention(distortions)
	if got != 6 {
		t.Errorf("retention = %d, want 6", got)
	}
}

func TestPipelineCost(t *testing.T) {
	effects := []Effect{
		{Tokens: 5000},
		{Tokens: 8000},
		{Tokens: 12000},
		{Tokens: 1000},
	}
	got := PipelineCost(effects)
	if got != 26000 {
		t.Errorf("cost = %d, want 26000", got)
	}
}

func TestFullEffectSeq(t *testing.T) {
	a := FullEffect{
		Cost:       Effect{Tokens: 5000, Quality: QualityGood},
		Distortion: Distortion{Retention: 60, Sensitivity: 1},
	}
	b := FullEffect{
		Cost:       Effect{Tokens: 12000, Quality: QualityExcellent},
		Distortion: Distortion{Retention: 80, Sensitivity: 2},
	}
	got := a.Seq(b)

	if got.Cost.Tokens != 17000 {
		t.Errorf("cost.tokens = %d, want 17000", got.Cost.Tokens)
	}
	if got.Cost.Quality != QualityGood {
		t.Errorf("cost.quality = %s, want good", got.Cost.Quality)
	}
	if got.Distortion.Retention != 48 {
		t.Errorf("distortion.retention = %d, want 48", got.Distortion.Retention)
	}
	if got.Distortion.Sensitivity != 2 {
		t.Errorf("distortion.sensitivity = %d, want 2", got.Distortion.Sensitivity)
	}
}
