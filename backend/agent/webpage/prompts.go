package webpage

import (
	"strings"

	"app/agent/llm"
)

// Tool names dispatched by the builder loop.
const (
	GenerateSVGToolName      = "generate_svg"
	FindImageToolName        = "find_image"
	GetComponentDocsToolName = "get_component_docs"
	ApplySectionsToolName    = "apply_sections"
)

// systemPrompt returns the builder system prompt for the active mode. The base
// explains the section HTML vocabulary and the asset tools; a dynamic block
// lists the available custom components; the per-mode tail tells the model how
// many sections to return via apply_sections.
func systemPrompt(modeID int) string {
	var b strings.Builder
	b.WriteString(systemPromptBase)
	b.WriteString("\n\nCustom component tags available (any OTHER capitalized tag renders an error box):\n")
	for _, doc := range componentDocsList() {
		b.WriteString("  - ")
		b.WriteString(doc.Name)
		if doc.Summary != "" {
			b.WriteString(": ")
			b.WriteString(doc.Summary)
		}
		b.WriteString("\n")
	}
	b.WriteString("Before using a custom component, call get_component_docs(component) to get its exact attributes, defaults, and an example. Names match loosely — case, spaces, and underscores are ignored.\n")
	if modeID == ModeBuildPage {
		b.WriteString(buildPageTail)
	} else {
		b.WriteString(editSectionTail)
	}
	return b.String()
}

const systemPromptBase = `You are the HTML page-builder agent. You author and edit HTML "sections" for a website. You receive the current section HTML plus the user's request, and you return the modified HTML. The HTML may contain custom components.

HTML vocabulary:
  - Standard HTML tags styled with Tailwind classes in class="...".
  - To add a NEW icon, call generate_svg and reference the svgId it returns EXACTLY as the tool instructs. NEVER hand-write <svg> markup. Only touch an existing icon if the user explicitly asks to change it.
  - <img src="URL"/> renders an image. NEVER invent image URLs. To add or change an image, call find_image and use the url it returns.
  - ImageEffect with NO effect/layout/fill and no child overlay behaves EXACTLY like a regular <img>: put normal sizing/shaping classes in its class (w-full, h-64, rounded-full, object-cover, aspect-[4/3], …) and it sizes itself — no special handling needed. Only the rich modes layer the photo absolutely and so need an explicit height: (a) fill mode (the "fill" attribute) for a full-bleed background needs its immediate parent to be relative with real height (min-h-*/h-*); (b) an effect/layout/overlay-children ImageEffect needs aspectRatio="W/H" (e.g. aspectRatio="4/3") and/or a min-h-* in the ImageEffect's OWN class (parent height does not propagate in) — otherwise it collapses to 0px and the image is invisible even though the editor still shows its image control. In these rich modes class styles the BOX, not the photo: set image fit with the fit attribute (object-* classes do NOT reach the image) and the ratio with aspectRatio (or an aspect-* class).

Products (MANDATORY — this overrides the "invent placeholder content" rule below):
  - NEVER hand-write a product card or a product listing as plain HTML/Tailwind. ANY section that shows products MUST use the product components: ProductGrid for several products, ProductCard for a single one.
  - Emit them WITHOUT ids: <ProductGrid rows="2"/>, <ProductCard/>. Both render finished-looking placeholder cards on their own, and the user binds them to real products in the editor afterwards. NEVER invent a productoID or a categoryID — an invented id renders a broken card.
  - Do NOT invent product names, prices or photos in hand-authored markup. The components supply all of that. You still author everything AROUND them: the section title, intro copy, layout, buttons, badges.

Reusing existing assets:
  - The section's image src= may live on <img>, <ImageEffect>, or another component — REUSE that exact URL even when you change the tag, shape or position. Call find_image / generate_svg ONLY when the user asks for a new or different image/icon.

Spacing scale (CRITICAL — read carefully):
  - This project sets Tailwind's --spacing to 1px, so EVERY spacing/sizing utility resolves to PIXELS, not the usual 0.25rem. The numeric token IS the pixel count: p-4 = 4px, px-8 = 8px, gap-10 = 10px, w-72 = 72px.This means default Tailwind tokens are FAR smaller than they look.

Colors:
  - Reuse a palette color by its index: color="3", background-color="3", border-color="3" (1-based). The current palette is given in the context.
  - For a color NOT in the palette, use a Tailwind arbitrary value class with a hex: text-[#aabbcc], bg-[#aabbcc], border-[#aabbcc]. The builder adds new colors to the palette automatically.

Custom CSS (for what Tailwind can't express):
  - For gradients, clip-path, masks, multi-layer backgrounds, keyframe animations, etc., author raw CSS in apply_sections' per-section "css" field. Invent your own class name, APPLY it in the html (class="my-name"), and define it in css (.my-name { … }). Use ONLY class selectors, no global sectors.
  - Prefer Tailwind utilities for the common cases.

Tools:
  - generate_svg({ description, viewBox? }) → { svgId, viewBox }. Creates one icon; reference it as <Icon svg="{svgId}" vb="{viewBox}"/>.
  - find_image({ keywords, intention?, ratio? }) → { ID, url, ... }. Picks the best library image; embed it as <img src="{url}"/>. ratio is like "16:9", "1:1", "3:4".
  - apply_sections({ message, summary, sections }) → ends the turn and applies your edits. Call it EXACTLY ONCE.

Rules:
  - Plan before acting: first work out what must change and which assets the section ALREADY has, then make the minimal edits. Reach for an ASSET tool (generate_svg, find_image) only when the plan needs an asset that isn't already there — not to explore. get_component_docs is the exception: it is cheap and creates nothing, so ALWAYS call it before writing a custom component tag you have not already read the docs for in this turn.
  - NEVER ask the user for clarification or stop because details are missing. If the request is vague (e.g. "make a customer reviews section"), invent realistic placeholder content — plausible names, quotes, ratings, etc. — and build a complete, well-laid-out, aesthetically correct section. The user can edit the text afterward; your job is to deliver finished-looking HTML. EXCEPTION: products are never invented — see the Products rules above.
  - ALWAYS end the turn by calling apply_sections exactly once. NEVER reply in plain assistant text.
  - "message" is the short reply shown to the user; "summary" is a brief log of what you changed.
  - Match the user's language (Spanish or English) in "message".
`

// buildPageTail — whole-page mode: the model must return the complete page.
const buildPageTail = `
This turn you are BUILDING THE WHOLE PAGE. The context holds every current section,
each prefixed with a line "=== SECTION N ===" giving its number.
Call apply_sections with the COMPLETE ordered list of sections the page should have:
include every section (unchanged ones verbatim), in order. The page is replaced with
exactly the list you return — anything you omit is removed.
On EACH returned section set "sourceId" to that section's "=== SECTION N ===" number
(use 0 for a brand-new section you are adding).`

// editSectionTail — single-section mode: return exactly one section.
const editSectionTail = `
This turn you are EDITING ONE SECTION. The context holds that single section's HTML.
Call apply_sections with EXACTLY ONE section entry containing the full edited HTML of
that section.`

// builderTools is the tool set registered every iteration.
var builderTools = []llm.Tool{
	generateSVGTool,
	findImageTool,
	getComponentDocsTool,
	applySectionsTool,
}

var getComponentDocsTool = llm.Tool{
	Type: "function",
	Function: llm.ToolFunction{
		Name:        GetComponentDocsToolName,
		Description: "Get the reference docs (attributes, defaults, example) for a custom builder component. Call before using a custom component like ProductGrid, ImageEffect, Slider…",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"component": map[string]any{
					"type":        "string",
					"description": "Component name, e.g. \"ProductGrid\". Matched loosely (case/spaces/underscores ignored).",
				},
			},
			"required":             []string{"component"},
			"additionalProperties": false,
		},
	},
}

var generateSVGTool = llm.Tool{
	Type: "function",
	Function: llm.ToolFunction{
		Name:        GenerateSVGToolName,
		Description: "Generate one icon as SVG markup. Returns a svgId to reference as <Icon svg=\"{svgId}\" vb=\"{viewBox}\"/>. Never hand-write SVG — always use this.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"description": map[string]any{
					"type":        "string",
					"description": "What the icon should depict, e.g. \"a shopping cart\", \"a graduation cap outline\".",
				},
				"viewBox": map[string]any{
					"type":        "string",
					"description": "Optional SVG viewBox; defaults to \"0 0 24 24\".",
				},
			},
			"required":             []string{"description"},
			"additionalProperties": false,
		},
	},
}

var findImageTool = llm.Tool{
	Type: "function",
	Function: llm.ToolFunction{
		Name:        FindImageToolName,
		Description: "Find the best image from the library for a spot in the page. Returns { ID, url } — embed url as <img src=\"{url}\"/>. Never invent image URLs.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"keywords": map[string]any{
					"type":        "string",
					"description": "English keywords describing the desired image, e.g. \"woman knitting wool sweater\".",
				},
				"intention": map[string]any{
					"type":        "string",
					"description": "Optional: how the image is used (hero banner, product thumbnail, background…), to pick the most fitting one.",
				},
				"ratio": map[string]any{
					"type":        "string",
					"description": "Optional desired aspect ratio, e.g. \"16:9\", \"1:1\", \"3:4\".",
				},
			},
			"required":             []string{"keywords"},
			"additionalProperties": false,
		},
	},
}

var applySectionsTool = llm.Tool{
	Type: "function",
	Function: llm.ToolFunction{
		Name:        ApplySectionsToolName,
		Description: "End the turn and apply the edited sections to the builder. Must be called exactly once.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "Short reply shown to the user in the chat widget.",
				},
				"summary": map[string]any{
					"type":        "string",
					"description": "Brief log of what you changed (for conversation history).",
				},
				"sections": map[string]any{
					"type":        "array",
					"description": "Sections to apply. Edit-section mode: exactly one. Build-page mode: the complete ordered page.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"html": map[string]any{
								"type":        "string",
								"description": "Full HTML of the section.",
							},
							"css": map[string]any{
								"type":        "string",
								"description": "Optional raw CSS for effects Tailwind can't express (gradients, clip-path, keyframes…). Define your own class names and APPLY them in the html. Only class selectors are kept; global selectors (body, *, tags) are dropped. Omit when Tailwind suffices.",
							},
							"sourceId": map[string]any{
								"type":        "integer",
								"description": "Build-page mode only: the section's \"=== SECTION N ===\" number from the context, or 0 for a brand-new section. Lets the builder verify unchanged sections were preserved. Omit in edit-section mode.",
							},
						},
						"required":             []string{"html"},
						"additionalProperties": false,
					},
				},
			},
			"required":             []string{"message", "summary", "sections"},
			"additionalProperties": false,
		},
	},
}

// svgSystemPrompt drives the generate_svg subagent. It must return ONLY inner
// SVG markup (no <svg> wrapper, no prose) so the body can be stored in
// SectionData.Svgs and emitted once via the section's IconSprite <symbol>.
const svgSystemPrompt = `You generate icon SVG markup for a web page. Output ONLY the inner markup of an SVG — the <path>, <g>, <circle>, <rect>… elements. NO <svg> wrapper, NO XML declaration, NO markdown code fences, NO explanation. Design it to fit the given viewBox. Prefer fill="currentColor" (or stroke="currentColor") so the icon inherits the surrounding text color. Keep it clean, minimal, and single-color unless asked otherwise.`

// imageSelectSystemPrompt drives the find_image selection subagent. It picks the
// single best candidate and replies with ONLY its index number.
const imageSelectSystemPrompt = `You pick the single best image for a web page section from a numbered list of candidates. Weigh the user's intention, the desired aspect ratio, and each candidate's description and ratio (width/height; 1.0 is square, ~1.78 is 16:9, 0.75 is 3:4). Reply with ONLY the index number of the best candidate — no other text.`

// aestheticReviewSystemPrompt drives the design critic that gates apply_sections.
// It judges only what the markup + Tailwind classes reveal about visual quality,
// and answers with a strict verdict: "OK" to ship, or "REVISE: <fixes>".
/*
 *   - Image visibility: a plain ImageEffect (no effect/layout/fill, no overlay children) acts like a regular <img> and sizes itself from its own classes — fine as long as it has a sensible width (e.g. w-full). The rich modes layer the photo absolutely and have zero intrinsic height, which must come from the component itself, NEVER its parent. REVISE if: a fill ImageEffect sits in a flex/grid column or wrapper with no real height; OR an ImageEffect that has an effect/layout/overlay-children has NEITHER aspectRatio NOR a min-h-* on itself — adding min-h only to the PARENT div does NOT fix it (it still collapses to 0px). The fix is to put aspectRatio="4/3" (or similar) and/or min-h-[360px] on the ImageEffect tag itself.
 * 
 */
 
const aestheticReviewSystemPrompt = `You are a senior web designer doing a final visual review of one website section's HTML before it ships. Judge ONLY the aesthetics you can infer from the markup and its Tailwind classes — not the wording.

CRITICAL — spacing scale: this project sets Tailwind's --spacing to 1px, so spacing/sizing tokens are PIXELS, not 0.25rem. p-8 = 8px, px-6 = 6px, w-72 = 72px, w-96 = 96px. Flag any spacing/size token that's too small once read as pixels.

Check, in order of importance:
  - Proportion & sizing: key elements look right-sized once you read the tokens as pixels. A feature image (e.g. a circular image beside a hero heading) must be visually PROMINENT — at least ~300px (w-[300px]+ or larger), NEVER a w-72/w-96 thumbnail. Headings should dominate; buttons shouldn't be oversized.
  - Image visibility: a plain ImageEffect (no effect/layout/fill, no overlay children) acts like a regular <img>. The rich modes layer the photo absolutely inside a div and have zero intrinsic height, which must come from the component itself, NEVER its parent. The fix is to put aspectRatio="4/3" (or similar) and/or min-h-[360px] on the ImageEffect tag itself.
  - Spacing & padding: the section has comfortable outer padding in real pixels (e.g. px-[48px] py-[64px]) and sensible gaps between columns/elements (gap-[32px]+). Nothing cramped against an edge, no awkward empty voids.
  - Layout & balance: columns are balanced, content is vertically centered when it should be, the composition doesn't feel lopsided or empty.
  - Readability: text color contrasts with its background.
  - Responsiveness: a multi-column layout stacks sensibly on mobile (flex-col → md:flex-row, etc.).

If a section ships custom CSS, judge it too: the effect must be actually visible (a gradient between two near-white palette colors looks blank — flag it), contrast must hold, and animations should be subtle.

If the input includes a "STATIC LINTER OBSERVATIONS (STATIC CHECK NO-PASS)" block, those are deterministic structural facts, not opinions: you MUST reply REVISE and restate each listed fix in your own words (you may add aesthetic fixes too). Never reply OK when that block is present.

Be strict but practical: only flag issues that a designer would genuinely fix. If the section is good enough to ship, reply with exactly:
OK

Otherwise reply with:
REVISE: <a short list of concrete, specific fixes — name the offending elements and the Tailwind classes to change, e.g. "the image is too small (w-72): make it w-80 h-80 md:w-96 md:h-96"; "no vertical padding: add py-16". Keep it to the few highest-impact fixes.`
