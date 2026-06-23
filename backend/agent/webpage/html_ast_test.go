package webpage

import "testing"

func TestParseHTMLToASTKeepsCustomSiblings(t *testing.T) {
	nodes, err := ParseHTMLToAST(`<section><ImageEffect src="/hero.webp" /><h2>Title</h2></section>`)
	if err != nil {
		t.Fatalf("ParseHTMLToAST returned error: %v", err)
	}
	if len(nodes) != 1 || nodes[0].TagName != "section" {
		t.Fatalf("top-level nodes = %#v, want one section", nodes)
	}

	children := nodes[0].Children
	if len(children) != 2 {
		t.Fatalf("section child count = %d, want 2; children=%#v", len(children), children)
	}
	if children[0].TagName != "ImageEffect" || children[0].Attributes["src"] != "/hero.webp" {
		t.Fatalf("first child = %#v, want ImageEffect with src", children[0])
	}
	if children[1].TagName != "h2" || len(children[1].Children) != 1 || children[1].Children[0].Text != "Title" {
		t.Fatalf("second child = %#v, want h2 text Title", children[1])
	}
}

func TestParseHTMLToASTCapturesIconAndMixedText(t *testing.T) {
	nodes, err := ParseHTMLToAST(`
		<h1>
			<Icon svg="icon--mdi-account-school" vb="0 0 24 24"></Icon>
			<span> Style   That Speaks </span>
		</h1>
	`)
	if err != nil {
		t.Fatalf("ParseHTMLToAST returned error: %v", err)
	}

	if len(nodes) != 1 || nodes[0].TagName != "h1" {
		t.Fatalf("top-level nodes = %#v, want one h1", nodes)
	}
	if len(nodes[0].Children) != 2 {
		t.Fatalf("h1 child count = %d, want icon + span", len(nodes[0].Children))
	}

	icon := nodes[0].Children[0]
	if icon.TagName != "Icon" || icon.Attributes["svg"] != "icon--mdi-account-school" {
		t.Fatalf("icon node = %#v, want preserved svg attribute", icon)
	}

	span := nodes[0].Children[1]
	if span.TagName != "span" || len(span.Children) != 1 || span.Children[0].Text != "Style That Speaks" {
		t.Fatalf("span node = %#v, want normalized text child", span)
	}
}

func TestParseHTMLToASTHandlesVoidImage(t *testing.T) {
	nodes, err := ParseHTMLToAST(`<div><img src="/a.jpg"><p>After</p></div>`)
	if err != nil {
		t.Fatalf("ParseHTMLToAST returned error: %v", err)
	}

	children := nodes[0].Children
	if len(children) != 2 {
		t.Fatalf("div child count = %d, want image + paragraph", len(children))
	}
	if children[0].TagName != "img" || children[0].Attributes["src"] != "/a.jpg" {
		t.Fatalf("image node = %#v, want img src", children[0])
	}
	if children[1].TagName != "p" || children[1].Children[0].Text != "After" {
		t.Fatalf("paragraph node = %#v, want text After", children[1])
	}
}

func mustExtract(t *testing.T, htmlText string) SectionContent {
	t.Helper()
	nodes, err := ParseHTMLToAST(htmlText)
	if err != nil {
		t.Fatalf("ParseHTMLToAST(%q) error: %v", htmlText, err)
	}
	return ExtractSectionContent(nodes)
}

func TestExtractSectionContent(t *testing.T) {
	got := mustExtract(t, `<section>
		<img src="/a.jpg"/>
		<ImageEffect src="/b.webp"/>
		<h2>Hello <Icon svg="icon--cart" vb="0 0 24 24"></Icon> World</h2>
	</section>`)

	if len(got.Images) != 2 || got.Images[0] != "/a.jpg" || got.Images[1] != "/b.webp" {
		t.Fatalf("images = %#v, want [/a.jpg /b.webp]", got.Images)
	}
	if len(got.Icons) != 1 || got.Icons[0] != "icon--cart" {
		t.Fatalf("icons = %#v, want [icon--cart]", got.Icons)
	}
	// Text nodes are collected as a multiset (each text run separately).
	if len(got.Texts) != 2 || got.Texts[0] != "Hello" || got.Texts[1] != "World" {
		t.Fatalf("texts = %#v, want [Hello World]", got.Texts)
	}
}

func TestVerifyKeepDetectsRemovalAndAddition(t *testing.T) {
	old := mustExtract(t, `<div><img src="/hero.jpg"/><Icon svg="icon--star"></Icon><p>Keep me</p></div>`)

	// Dropped the icon and reworded the text — both must be flagged under KEEP.
	changed := mustExtract(t, `<div><img src="/hero.jpg"/><p>Different words</p></div>`)
	v := VerifySectionContent(old, changed, ContentPolicy{Text: PolicyKeep, Images: PolicyKeep, Icons: PolicyKeep})
	if len(v) == 0 {
		t.Fatalf("expected violations for removed icon + changed text, got none")
	}

	// Only the image changed; text+icons preserved → no violation when images=modify.
	imgOnly := mustExtract(t, `<div><img src="/new.jpg"/><Icon svg="icon--star"></Icon><p>Keep me</p></div>`)
	v = VerifySectionContent(old, imgOnly, ContentPolicy{Text: PolicyKeep, Images: PolicyModify, Icons: PolicyKeep})
	if len(v) != 0 {
		t.Fatalf("expected no violations (only image changed, images=modify), got %v", v)
	}
}

func TestVerifyAddAllowsAdditionsNotRemovals(t *testing.T) {
	old := mustExtract(t, `<ul><li>One</li><li>Two</li></ul>`)
	added := mustExtract(t, `<ul><li>One</li><li>Two</li><li>Three</li></ul>`)
	if v := VerifySectionContent(old, added, ContentPolicy{Text: PolicyAdd, Images: PolicyAdd, Icons: PolicyAdd}); len(v) != 0 {
		t.Fatalf("ADD should allow new text, got violations %v", v)
	}
	removed := mustExtract(t, `<ul><li>One</li></ul>`)
	if v := VerifySectionContent(old, removed, ContentPolicy{Text: PolicyAdd, Images: PolicyAdd, Icons: PolicyAdd}); len(v) == 0 {
		t.Fatalf("ADD should reject removal of existing text")
	}
}
