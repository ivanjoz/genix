package webpage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"app/agent/llm"
	"app/business"
	"app/core"
)

// dispatchTool runs one non-terminating tool call (the apply_sections
// terminator is handled in the loop) and returns its result as a JSON string
// for the model. Errors come back as JSON {"error":"..."} so the model can
// recover instead of seeing a raw Go error.
func (t *builderTurn) dispatchTool(ctx context.Context, call llm.ToolCall) string {
	switch call.Function.Name {
	case GenerateSVGToolName:
		return t.generateSVG(ctx, call)
	case FindImageToolName:
		return t.findImage(ctx, call)
	case GetComponentDocsToolName:
		return componentDocsResult(call)
	default:
		return toolErrorJSON(fmt.Errorf("herramienta desconocida %q — solo generate_svg, find_image, get_component_docs y apply_sections están disponibles", call.Function.Name))
	}
}

// componentDocsResult returns the reference docs for one custom component. On a
// miss it returns the available names so the agent can correct itself. Pure
// lookup — no subagent, no LLM call.
func componentDocsResult(call llm.ToolCall) string {
	var args struct {
		Component string `json:"component"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return toolErrorJSON(fmt.Errorf("decode args: %w", err))
	}
	doc, ok := componentDocByName(args.Component)
	if !ok {
		return toolJSON(map[string]any{
			"error":     fmt.Sprintf("componente %q no encontrado", args.Component),
			"available": componentNamesList(),
		})
	}
	core.Log("agent.webpage get_component_docs component::", doc.Name)
	return toolJSON(map[string]any{"name": doc.Name, "docs": doc.Body})
}

// generateSVG spawns the SVG subagent, stores the returned inner markup in the
// turn's Svgs map keyed by a fresh sprite id, and returns just that id +
// viewBox to the main agent (the body never re-enters the main conversation).
func (t *builderTurn) generateSVG(ctx context.Context, call llm.ToolCall) string {
	var args struct {
		Description string `json:"description"`
		ViewBox     string `json:"viewBox"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return toolErrorJSON(fmt.Errorf("decode args: %w", err))
	}
	if strings.TrimSpace(args.Description) == "" {
		return toolErrorJSON(errors.New("description is required"))
	}
	viewBox := strings.TrimSpace(args.ViewBox)
	if viewBox == "" {
		viewBox = "0 0 24 24"
	}

	body, err := t.runSubagent(ctx, "generate_svg", subagentNoReasoning, svgSystemPrompt, fmt.Sprintf("Description: %s\nviewBox: %s", args.Description, viewBox))
	if err != nil {
		return toolErrorJSON(fmt.Errorf("generate svg: %w", err))
	}
	body = cleanSVGBody(body)
	if body == "" {
		return toolErrorJSON(errors.New("el subagente SVG devolvió contenido vacío"))
	}

	t.svgSeq++
	id := fmt.Sprintf("genix-svg-%d", t.svgSeq)
	t.svgs[id] = body
	core.Log("agent.webpage generate_svg id::", id, " body_bytes::", len(body))

	return toolJSON(map[string]any{
		"svgId":   id,
		"viewBox": viewBox,
		"usage":   fmt.Sprintf(`Reference this icon as <Icon svg="%s" vb="%s"/>`, id, viewBox),
	})
}

// findImage queries the library for candidates and, when there is more than
// one, asks the selection subagent to pick the best fit for the intention +
// ratio. Always returns one image (the search has a first-images fallback).
func (t *builderTurn) findImage(ctx context.Context, call llm.ToolCall) string {
	var args struct {
		Keywords  string `json:"keywords"`
		Intention string `json:"intention"`
		Ratio     string `json:"ratio"`
	}
	if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
		return toolErrorJSON(fmt.Errorf("decode args: %w", err))
	}

	candidates, err := business.FindImageCandidates(args.Keywords, 10)
	if err != nil {
		return toolErrorJSON(fmt.Errorf("buscar imágenes: %w", err))
	}
	if len(candidates) == 0 {
		return toolErrorJSON(errors.New("no hay imágenes disponibles en la biblioteca"))
	}

	chosen := candidates[0]
	if len(candidates) > 1 {
		if idx := t.pickImage(ctx, candidates, args.Intention, args.Ratio); idx >= 0 && idx < len(candidates) {
			chosen = candidates[idx]
		}
	}
	core.Log("agent.webpage find_image keywords::", args.Keywords, " candidates::", len(candidates), " chosen_id::", chosen.ID)

	return toolJSON(map[string]any{
		"ID":          chosen.ID,
		"url":         chosen.URL,
		"description": chosen.Description,
		"ratio":       chosen.Ratio,
		"usage":       fmt.Sprintf(`Embed this image as <img src="%s"/>`, chosen.URL),
	})
}

// pickImage asks the selection subagent for the best candidate index. On any
// failure it falls back to index 0 (the top text-search match).
func (t *builderTurn) pickImage(ctx context.Context, candidates []business.AgentImageCandidate, intention, ratio string) int {
	var b strings.Builder
	fmt.Fprintf(&b, "Desired intention: %s\nDesired aspect ratio: %s\n\nCandidates:\n", intention, ratio)
	for i, candidate := range candidates {
		shownRatio := candidate.Ratio
		if shownRatio == 0 {
			shownRatio = 1 // 0 ⇒ unknown, treated as 1:1
		}
		fmt.Fprintf(&b, "%d) ratio=%.3f desc=%s\n", i, shownRatio, candidate.Description)
	}
	b.WriteString("\nReply with ONLY the index number of the best candidate.")

	out, err := t.runSubagent(ctx, "find_image_select", subagentNoReasoning, imageSelectSystemPrompt, b.String())
	if err != nil {
		core.Log("agent.webpage find_image select subagent error::", err)
		return 0
	}
	return parseFirstInt(out)
}

// cleanSVGBody strips a markdown code fence and an accidental <svg> wrapper, so
// what we store in Svgs is the bare inner markup the IconSprite <symbol> expects.
func cleanSVGBody(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if nl := strings.IndexByte(s, '\n'); nl >= 0 {
			s = s[nl+1:]
		}
		s = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(s), "```"))
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "<svg") {
		if open := strings.IndexByte(s, '>'); open >= 0 {
			if end := strings.LastIndex(lower, "</svg>"); end > open {
				s = strings.TrimSpace(s[open+1 : end])
			}
		}
	}
	return s
}

// parseFirstInt returns the first non-negative integer found in s, or -1.
func parseFirstInt(s string) int {
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			if start < 0 {
				start = i
			}
		} else if start >= 0 {
			n, _ := strconv.Atoi(s[start:i])
			return n
		}
	}
	if start >= 0 {
		n, _ := strconv.Atoi(s[start:])
		return n
	}
	return -1
}
