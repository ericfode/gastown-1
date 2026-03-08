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

func TestFidelityLosslessIsTop(t *testing.T) {
	top := Lossless()
	other := Fidelity(5)
	if !other.Le(top) {
		t.Error("non-top should be ≤ Lossless")
	}
	if !top.Le(top) {
		t.Error("Lossless should be ≤ itself (reflexive)")
	}
}

func TestFidelitySeqDPI(t *testing.T) {
	a := Fidelity(2)
	b := Fidelity(5)
	got := a.Seq(b)
	if !got.Le(a) {
		t.Errorf("Seq(%d, %d) = %d, should be ≤ %d (DPI)", a, b, got, a)
	}
}

func TestFidelityParBestPath(t *testing.T) {
	a := Fidelity(5)
	b := Fidelity(2)
	got := a.Par(b)
	if !a.Le(got) {
		t.Errorf("Par(%d, %d) = %d, %d should be ≤ result (best-path)", a, b, got, a)
	}
}

func TestFidelitySeqLosslessIdentity(t *testing.T) {
	top := Lossless()
	a := Fidelity(3)
	if top.Seq(a) != a {
		t.Errorf("Lossless.Seq(a) = %d, want %d", top.Seq(a), a)
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
