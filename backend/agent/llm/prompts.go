package llm

// System prompt + tool schemas for the in-app agent loop. The loop wires the
// page-driving tools (`get_page`, `get_menu`, `navigate`, `invoke_batch`)
// into the existing /ws/agent bridge so the model can both *see* the current
// page and *act* on it. The model still has to call `finish` to end a turn —
// any other tool just collects more information or mutates the page and
// then the loop re-asks the model what to do next.

// SystemPromptChat opens every chat turn. Tight: every byte goes out on
// every iteration. The model is told:
//  1. who it is + product context
//  2. how to *read* the page (get_page / get_menu)
//  3. how to *act* on the page (navigate / invoke_batch)
//  4. how to end the turn (finish, exactly once)
//
// We rely on the prompt — not `tool_choice: "required"` — to discipline the
// model into calling `finish`, because the tencent/hy3-preview provider
// routing rejects "required" (see openrouter.go).
const SystemPromptChat = `You are Genix, an in-app assistant for the Genix ERP/e-commerce platform.
The user is chatting with you through a widget in the app while looking at a real page.

You have tools to inspect and operate that page:

  - get_page() → returns the current page's interactive components and a cleaned HTML snapshot.
      Call this whenever the user asks "what's on this page", "what am I looking at",
      "how do I…", or before you act on something whose id/method you don't already know.
      ALSO call it before mutating a form whose current state you don't already know —
      checking the page first prevents you from re-filling values that are already set.
  - get_menu() → returns TSV columns: group_id, group_name, option_name, route, description.
      Useful when the user asks "where is X" or "take me to Y" and you don't know the route.
  - navigate({ route }) → moves the SPA to "route" (e.g. "/comercial/sale-orders").
      Only use a route you got from get_menu() — never invent routes.
      Prefer returnPageContent=true whenever you will need to inspect or act on the destination page next;
      this avoids a separate get_page() call.
  - invoke_batch({ invocations: [{ ID, Method, Args }] }) → runs one or more
      method calls on registered components from get_page(). ID is the value of
      the id="..." attribute from the snapshot (plain "38" or composite "38:100");
      the browser runs invocations sequentially and stops on the first failure.
      Use this to fill inputs, click buttons, select rows, etc.
      Prefer returnPageContent=true for clicks, selects, opens, saves, or any action that may change visible UI;
      this avoids a separate get_page() call.
  - finish({ message, summary }) → end the turn. Required to terminate.

About the conversation history you receive each turn:
  - Past ` + "`user`" + ` and ` + "`assistant`" + ` messages are the spoken exchange.
  - Some assistant turns are followed by a ` + "`system`" + ` note that starts with
    "[Acciones ya realizadas en esta conversación]". That note lists the actual page
    actions the assistant performed (which fields were filled, which routes navigated,
    which records were saved). These actions are PERMANENT for the rest of the session.
  - On a follow-up turn, treat those actions as already done. Do NOT call setValue/
    click/save again for items listed there. If unsure whether something is still set,
    call get_page() to inspect — don't re-mutate as a precaution.

Rules:
  - ALWAYS end the turn by calling ` + "`finish`" + ` exactly once. Never reply in plain assistant text.
  - When filling a form, ALWAYS ask user before save or send.
  - The ` + "`summary`" + ` you pass to ` + "`finish`" + ` MUST be a concrete log of the page
    actions you took this turn — e.g. "Llené Nombre, Precio Base, Precio Final, Moneda y
    guardé el producto" or "Navegué a /comercial/sale-orders y no realicé otras acciones".
    Do NOT use the summary as a second copy of the reply. Future turns will rely on it
    to avoid repeating work.
  - Prefer to call get_page() before answering questions about the visible UI; do not
    guess what's on the page from memory.
  - Never invent component ids, methods, or routes. If get_page()/get_menu() doesn't
    have it, say so in the ` + "`finish`" + ` message.
  - Keep replies short unless the user explicitly asks for detail.
  - Match the user's language (Spanish or English) in the ` + "`finish`" + ` message.`

// Tool name constants — referenced by the loop dispatcher.
const (
	FinishToolName      = "finish"
	GetPageToolName     = "get_page"
	GetMenuToolName     = "get_menu"
	NavigateToolName    = "navigate"
	InvokeBatchToolName = "invoke_batch"
)

// PageSnapshotGrammar is prepended to every tool result that carries a parsed
// page snapshot (get_page; navigate / invoke_batch when returnPageContent is
// set). Smaller models otherwise misread the compact tag conventions —
// confusing `[id] label` for free text, ignoring `options-count`, calling
// `select` with the label instead of the id. Keeping the grammar attached
// to the snapshot (not the system prompt) means we only pay the tokens when
// the model is actually going to read it.
const PageSnapshotGrammar = `SNAPSHOT GRAMMAR — apply to the html / page.html field in this result:
  <Type id="N" label="..." value="..." methods="m1,m2"/>
    - id: pass verbatim as ID to invoke_batch. May be plain ("38") or composite ("38:100"); composite ids route automatically.
    - methods: comma-separated legal Method names for this handle.
    - value: uses "[id] text" when it references another id (selected option/row).
    - selected (bare attr): marks the active item on Option / TableRow / Checkbox.

  Composite ids appear on items whose parent is a container:
    - TableRow / CellInput / CellSelect inside Table or CardList.
    - Option inside OptionsStrip, PageViews (parent tag is bare — no id/methods).
  Always pass the full composite to invoke_batch; never the parent id alone, never the child suffix alone.

  Examples — call the verb shown in methods= with the id verbatim:
    <TableRow id="38:100" selected methods="select"/>             → invoke_batch [{ID:"38:100", Method:"select"}]
    <CellInput id="38:101" value="..." methods="setValue"/>       → invoke_batch [{ID:"38:101", Method:"setValue", Args:["new value"]}]
    <Option id="30:2" methods="select">Categorías</Option>        → invoke_batch [{ID:"30:2", Method:"select"}]

  <Select> carries two extra attrs:
    - options-count="N"      total option count.
    - options="[1] A|[2] B"  pipe-separated "[id] label" list. ONLY present when full list <=12
                             (exhaustive — call select with one of the listed ids). When absent
                             (count > 12), call search(text) — it returns the matches directly,
                             so a follow-up getOptions is not needed.

  search and getOptions return their option list as a TSV string with header "ID\tValue"
  (one option per line). Read the ID column and pass it to select.
`

// FinishTool is the only tool that ends a turn. Arguments map directly to
// the fields persisted on the agent's AgentMessage row.
var FinishTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        FinishToolName,
		Description: "End the current turn and deliver the reply to the user. Must be called exactly once.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"message": map[string]any{
					"type":        "string",
					"description": "The reply the user will see in the chat widget.",
				},
				"summary": map[string]any{
					"type": "string",
					"description": "Concrete log of every PAGE ACTION you took this turn — which fields you " +
						"filled with which values, which routes you navigated to, which records you saved/deleted. " +
						"Future turns receive this verbatim as a system note so they know not to redo work. " +
						"If you performed no actions this turn (only answered), write \"Sin acciones; solo respondí\".",
				},
			},
			"required":             []string{"message", "summary"},
			"additionalProperties": false,
		},
	},
}

// GetPageTool fetches the components registry + cleaned HTML for the page
// the user is currently looking at. No arguments — the loop already knows
// the TabID.
var GetPageTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        GetPageToolName,
		Description: "Get the current page's interactive components and a cleaned HTML snapshot. Call before answering questions about visible UI.",
		Parameters: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
	},
}

// GetMenuTool returns TSV rows with side-menu groups and routes the user has
// access to. Used to discover routes before calling `navigate`.
var GetMenuTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        GetMenuToolName,
		Description: "Get TSV rows with side-menu groups, option names, SPA routes, and descriptions. Use to find a route before calling navigate.",
		Parameters: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		},
	},
}

// NavigateTool changes the SPA route.
var NavigateTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        NavigateToolName,
		Description: "Change the SPA route. Pass a route obtained from get_menu — never invent routes. Prefer returnPageContent=true when you will inspect or act on the destination page next, instead of calling get_page separately.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"route": map[string]any{
					"type":        "string",
					"description": "SPA path, e.g. /comercial/sale-orders.",
				},
				"returnPageContent": map[string]any{
					"type":        "boolean",
					"description": "Prefer true when the next step needs the destination page snapshot. The browser waits for navigation/DOM updates and returns page content in this same call.",
				},
			},
			"required":             []string{"route"},
			"additionalProperties": false,
		},
	},
}

// InvokeBatchTool runs a list of method calls on components returned by
// get_page. The shape matches the existing /agent HTTP contract so the
// browser-side dispatcher needs no changes.
var InvokeBatchTool = Tool{
	Type: "function",
	Function: ToolFunction{
		Name:        InvokeBatchToolName,
		Description: "Invoke one or more methods on registered components. The browser runs them sequentially and stops on the first failure. Prefer returnPageContent=true after clicks, selects, opens, saves, or any UI-changing action you need to inspect next, instead of calling get_page separately.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"invocations": map[string]any{
					"type":        "array",
					"description": "Sequence of invocations to run.",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"ID": map[string]any{
								"type":        "string",
								"description": "id from the snapshot's id=\"...\" attribute. Plain ('38') for top-level handles, composite ('38:100') for rows/cells inside a Table or CardList. The bridge routes composite ids to the parent automatically.",
							},
							"Method": map[string]any{
								"type":        "string",
								"description": "Method name shown in the component's methods=... attribute.",
							},
							"Args": map[string]any{
								"type":        "array",
								"description": "Arguments to pass to the method, in order.",
								"items":       map[string]any{},
							},
						},
						"required":             []string{"ID", "Method"},
						"additionalProperties": false,
					},
				},
				"returnPageContent": map[string]any{
					"type":        "boolean",
					"description": "Prefer true when the batch may change visible UI and the next step needs the updated page snapshot. The browser waits for action/DOM updates and returns page content in this same call.",
				},
			},
			"required":             []string{"invocations"},
			"additionalProperties": false,
		},
	},
}

// ChatTools is the full tool set registered for every turn. Order is not
// significant to the model but kept readable for humans.
var ChatTools = []Tool{
	GetPageTool,
	GetMenuTool,
	NavigateTool,
	InvokeBatchTool,
	FinishTool,
}
