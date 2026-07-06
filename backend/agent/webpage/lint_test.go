package webpage

import (
	"strings"
	"testing"
)

func lintOne(html string) []string {
	return staticLintSections([]SectionEdit{{HTML: html}})
}

func hasObs(obs []string, substr string) bool {
	for _, o := range obs {
		if strings.Contains(o, substr) {
			return true
		}
	}
	return false
}

func TestLint_PlainImageEffectIsClean(t *testing.T) {
	// No effect/layout/fill, no children → behaves like a regular <img>. No height needed.
	cases := []string{
		`<ImageEffect src="x.avif" fit="cover" class="w-full rounded-2xl"></ImageEffect>`,
		`<ImageEffect src="x.avif" class="w-[120px] h-[120px] rounded-full"></ImageEffect>`,
		`<div class="md:w-1/2"><ImageEffect src="x.avif" fit="cover" class="w-full"></ImageEffect></div>`,
	}
	for _, c := range cases {
		if obs := lintOne(c); len(obs) != 0 {
			t.Errorf("expected no observations for %q, got %v", c, obs)
		}
	}
}

func TestLint_RichModeMissingHeight(t *testing.T) {
	// The original bug: removed fill, min-h only on PARENT, none on the ImageEffect.
	html := `<div class="md:w-1/2 min-h-[400px]"><ImageEffect effect="overlay" src="x.avif" fit="cover"></ImageEffect></div>`
	obs := lintOne(html)
	if !hasObs(obs, "NO height source") {
		t.Fatalf("expected rich-mode height observation, got %v", obs)
	}
}

func TestLint_RichModeWithHeightIsClean(t *testing.T) {
	cases := []string{
		`<ImageEffect effect="overlay" aspectRatio="4/3" src="x.avif"></ImageEffect>`,
		`<ImageEffect layout="curve-right" class="min-h-[360px]" src="x.avif"></ImageEffect>`,
		`<ImageEffect effect="overlay" class="md:min-h-[500px]" src="x.avif"></ImageEffect>`, // responsive prefix
		`<ImageEffect effect="overlay" class="aspect-[4/3]" src="x.avif"></ImageEffect>`,      // aspect-* class now valid
	}
	for _, c := range cases {
		if obs := lintOne(c); hasObs(obs, "NO height source") {
			t.Errorf("did not expect height observation for %q, got %v", c, obs)
		}
	}
}

func TestLint_FillParentNoHeight(t *testing.T) {
	html := `<div class="md:w-1/2"><ImageEffect fill effect="overlay" src="x.avif"></ImageEffect></div>`
	if obs := lintOne(html); !hasObs(obs, "fill mode") {
		t.Fatalf("expected fill-parent observation, got %v", obs)
	}
	// Parent with real height → clean.
	ok := `<div class="md:w-1/2 min-h-[420px]"><ImageEffect fill effect="overlay" src="x.avif"></ImageEffect></div>`
	if obs := lintOne(ok); hasObs(obs, "fill mode") {
		t.Errorf("did not expect fill observation when parent has height, got %v", obs)
	}
}

func TestLint_ObjectFitClassInRichMode(t *testing.T) {
	html := `<ImageEffect effect="overlay" aspectRatio="4/3" class="object-contain" src="x.avif"></ImageEffect>`
	if obs := lintOne(html); !hasObs(obs, "object-fit utilities never reach") {
		t.Fatalf("expected object-fit observation, got %v", obs)
	}
	// object-* on a PLAIN image is fine (it reaches the <img>).
	plain := `<ImageEffect class="w-full object-cover" src="x.avif"></ImageEffect>`
	if obs := lintOne(plain); len(obs) != 0 {
		t.Errorf("did not expect observation for plain object-cover, got %v", obs)
	}
}

func TestLint_LooseTagMatch(t *testing.T) {
	html := `<image-effect effect="overlay" src="x.avif"></image-effect>`
	if obs := lintOne(html); !hasObs(obs, "NO height source") {
		t.Errorf("expected loose tag match to lint, got %v", obs)
	}
}
