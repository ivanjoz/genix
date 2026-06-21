// Global agent "modes". A mode tells the agent the user's current intent so it
// can answer in the right context (plain search, build a page, edit a section…).
// Pages register their own modes (e.g. the webpage builder) and the chat widget
// reads the active mode for its placeholder and sends the mode ID with each
// message. Naming note: "Mode" (not "Context") — the user picks how the agent
// operates; "Model" is reserved for the LLM model (see models.svelte.ts).
export interface IAgentMode {
	ID: number
	// Bilingual "English|Spanish" strings, resolved with tr() at render time.
	Placeholder: string
	Name: string
	Icon: string
}

// Mode 1 is always available: the default free-form search/ask mode.
export const DEFAULT_AGENT_MODE: IAgentMode = {
	ID: 1,
	Placeholder: 'Search or Ask Genix...|Busca o Pregúntale a Genix...',
	Name: 'Ask|Preguntar',
	Icon: 'icon-[fa--commenting-o]',
}

// A page-supplied function that returns extra context (e.g. the builder's
// sections serialized to HTML) for the active mode. Returns '' when the active
// mode needs no context. Called by the chat widget at send time.
export type AgentContextProvider = (modeID: number) => string

// Payload the backend's page-builder loop sends down the stream (agentSections,
// mirrors backend/agent/chat_ws.go ChatAgentSections). The page applies it back
// into its own model via the registered AgentSectionsApplier.
export interface AgentSectionsPayload {
	ModeID: number
	Sections: { html: string; css?: string }[]
	Svgs: Record<string, string>
	Message: string
	Summary: string
	Timestamp: number
}

// A page-supplied function that applies the agent's edited sections back into
// the page (e.g. parse the HTML into the builder's AST). Registered like the
// context provider; absent on pages that don't author sections.
export type AgentSectionsApplier = (payload: AgentSectionsPayload) => void

// Reactive singleton: the chat widget binds to it, pages mutate it.
class AgentModesState {
	// Extra modes registered by the current page (the default mode is implicit).
	registered = $state<IAgentMode[]>([])
	// ID of the mode the user is currently in.
	activeID = $state<number>(DEFAULT_AGENT_MODE.ID)
	// Optional context provider registered by the current page (plain ref, not
	// reactive — only read imperatively when a message is sent).
	contextProvider: AgentContextProvider | null = null
	// Optional applier for agent-returned sections (plain ref, not reactive —
	// only invoked imperatively when an agentSections event arrives).
	sectionsApplier: AgentSectionsApplier | null = null

	// Full list shown in the selector: default first, then page-registered modes.
	all = $derived<IAgentMode[]>([DEFAULT_AGENT_MODE, ...this.registered])
	// The active mode, falling back to the default if the active ID is gone.
	active = $derived<IAgentMode>(this.all.find((m) => m.ID === this.activeID) ?? DEFAULT_AGENT_MODE)

	// A page registers its modes and the ID it wants active by default. Called
	// from an $effect so it re-runs when the page's modes change.
	set(modes: IAgentMode[], activeID?: number) {
		this.registered = modes
		this.activeID = activeID ?? modes[0]?.ID ?? DEFAULT_AGENT_MODE.ID
	}

	// Switch the active mode (user clicked a selector button).
	select(id: number) {
		this.activeID = id
	}

	// A page registers a provider that supplies context HTML for the active mode.
	setContextProvider(provider: AgentContextProvider) {
		this.contextProvider = provider
	}

	// Resolve the context for the active mode, '' when none is available.
	getActiveContext(): string {
		return this.contextProvider?.(this.activeID) ?? ''
	}

	// A page registers how to apply agent-returned sections back into its model.
	setSectionsApplier(applier: AgentSectionsApplier) {
		this.sectionsApplier = applier
	}

	// Apply agent-returned sections via the page's applier; no-op when none.
	applySections(payload: AgentSectionsPayload) {
		this.sectionsApplier?.(payload)
	}

	// Restore the default-only state when a page unmounts.
	clear() {
		this.registered = []
		this.activeID = DEFAULT_AGENT_MODE.ID
		this.contextProvider = null
		this.sectionsApplier = null
	}
}

export const agentModes = new AgentModesState()
