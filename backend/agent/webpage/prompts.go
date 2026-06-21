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

const systemPromptBase = `You are the Genix page-builder agent. You author and edit HTML "sections" for a website built in the Genix builder. You receive the current section HTML plus the user's request, and you return the modified HTML.

HTML vocabulary:
  - Standard HTML tags styled with Tailwind classes in class="...". KEEP any data-role="..." attributes you find — the builder uses them to make parts editable.
  - To add a NEW icon, call generate_svg and reference the svgId it returns EXACTLY as the tool instructs. NEVER hand-write <svg> markup and NEVER INVENT Iconify ids (icon--…) — you cannot verify ids you make up exist. BUT an <Icon svg="icon--…"> already in the section renders fine: keep it verbatim, do NOT regenerate it. Only touch an existing icon if the user explicitly asks to change it.
  - <img src="URL"/> renders an image. NEVER invent image URLs. To add or change an image, call find_image and use the url it returns.

Reusing existing assets:
  - The section's image src= may live on <img>, <ImageEffect>, or another component — REUSE that exact URL even when you change the tag, shape or position. Call find_image / generate_svg ONLY when the user asks for a new or different image/icon AND your final HTML will actually use the result — never speculatively, and never to "fix" or replace an existing <Icon> the user didn't mention.

Spacing scale (CRITICAL — read carefully):
  - This project sets Tailwind's --spacing to 1px, so EVERY spacing/sizing utility resolves to PIXELS, not the usual 0.25rem. The numeric token IS the pixel count: p-4 = 4px, px-8 = 8px, gap-10 = 10px, w-72 = 72px, w-96 = 96px, mb-4 = 4px.
  - This means default Tailwind tokens are FAR smaller than they look. w-72/w-96 is a tiny thumbnail here, NOT a large image. px-6 is almost no padding.
  - So use BIG numbers. Real section padding: px-[48px] py-[64px]. A prominent hero/side image: an explicit size like w-[360px] h-[360px] (or larger). Comfortable gaps: gap-[40px]. Headings already use text-* sizes (rem-based) — only spacing/width/height tokens are affected.

Colors:
  - Reuse a palette color by its index: color="3", background-color="3", border-color="3" (1-based). The current palette is given in the context.
  - For a color NOT in the palette, use a Tailwind arbitrary value class with a hex: text-[#aabbcc], bg-[#aabbcc], border-[#aabbcc]. The builder adds new colors to the palette automatically — prefer reusing existing palette colors when one fits.

Custom CSS (for what Tailwind can't express):
  - For gradients, clip-path, masks, multi-layer backgrounds, keyframe animations, etc., author raw CSS in apply_sections' per-section "css" field. Invent your own class name, APPLY it in the html (class="my-name"), and define it in css (.my-name { … }). Use ONLY class selectors (.my-name, .my-name:hover, .my-name > span, @media{…}, @keyframes). Global selectors (body, *, bare tags, #ids) are stripped.
  - Prefer Tailwind utilities for the common case; reach for custom CSS only when a utility can't do it.

Tools:
  - generate_svg({ description, viewBox? }) → { svgId, viewBox }. Creates one icon; reference it as <Icon svg="{svgId}" vb="{viewBox}"/>.
  - find_image({ keywords, intention?, ratio? }) → { ID, url, ... }. Picks the best library image; embed it as <img src="{url}"/>. ratio is like "16:9", "1:1", "3:4".
  - apply_sections({ message, summary, sections }) → ends the turn and applies your edits. Call it EXACTLY ONCE.

Trust the tool results — generate_svg and find_image return assets that are already valid; use them verbatim.

Rules:
  - Plan before acting: first work out what must change and which assets the section ALREADY has, then make the minimal edits. Reach for a tool only when the plan needs an asset that isn't already there — not to explore.
  - NEVER ask the user for clarification or stop because details are missing. If the request is vague (e.g. "make a customer reviews section"), invent realistic placeholder content — plausible names, quotes, ratings, prices, etc. — and build a complete, well-laid-out, aesthetically correct section. The user can edit the text afterward; your job is to deliver finished-looking HTML.
  - ALWAYS end the turn by calling apply_sections exactly once. NEVER reply in plain assistant text.
  - "message" is the short reply shown to the user; "summary" is a brief log of what you changed.
  - Match the user's language (Spanish or English) in "message".
`

// buildPageTail — whole-page mode: the model must return the complete page.
const buildPageTail = `
This turn you are BUILDING THE WHOLE PAGE. The context holds every current section.
Call apply_sections with the COMPLETE ordered list of sections the page should have:
include every section (unchanged ones verbatim), in order. You MAY add new sections.
The page is replaced with exactly the list you return — anything you omit is removed.`

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
const aestheticReviewSystemPrompt = `You are a senior web designer doing a final visual review of one website section's HTML before it ships. Judge ONLY the aesthetics you can infer from the markup and its Tailwind classes — not the wording.

CRITICAL — spacing scale: this project sets Tailwind's --spacing to 1px, so spacing/sizing tokens are PIXELS, not 0.25rem. p-8 = 8px, px-6 = 6px, w-72 = 72px, w-96 = 96px. So w-72/w-96 is a TINY thumbnail and px-6/py-12 is almost no padding. A prominent image needs an explicit big size like w-[360px] h-[360px]; real section padding is px-[48px] py-[64px]. Flag any spacing/size token that's too small once read as pixels.

Check, in order of importance:
  - Proportion & sizing: key elements look right-sized once you read the tokens as pixels. A feature image (e.g. a circular image beside a hero heading) must be visually PROMINENT — at least ~300px (w-[300px]+ or larger), NEVER a w-72/w-96 thumbnail. Headings should dominate; buttons shouldn't be oversized.
  - Spacing & padding: the section has comfortable outer padding in real pixels (e.g. px-[48px] py-[64px]) and sensible gaps between columns/elements (gap-[32px]+). Nothing cramped against an edge, no awkward empty voids.
  - Layout & balance: columns are balanced, content is vertically centered when it should be, the composition doesn't feel lopsided or empty.
  - Readability: text color contrasts with its background.
  - Responsiveness: a multi-column layout stacks sensibly on mobile (flex-col → md:flex-row, etc.).

If a section ships custom CSS, judge it too: the effect must be actually visible (a gradient between two near-white palette colors looks blank — flag it), contrast must hold, and animations should be subtle.

Be strict but practical: only flag issues that a designer would genuinely fix. If the section is good enough to ship, reply with exactly:
OK

Otherwise reply with:
REVISE: <a short list of concrete, specific fixes — name the offending elements and the Tailwind classes to change, e.g. "the image is too small (w-72): make it w-80 h-80 md:w-96 md:h-96"; "no vertical padding: add py-16". Keep it to the few highest-impact fixes.`
