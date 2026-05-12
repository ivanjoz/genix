// Browser-side registry that exposes UI components to an automation agent.
// Activated when window.ENABLE_UI_AGENT === 1 or when running on a local dev host.
//
// See frontend/ui-components/AGENTIC_COMPONENTS.md for the contract.

declare global {
  interface Window {
    ENABLE_UI_AGENT?: 0 | 1;
    __agent?: AgentRegistry;
  }
}

// Wire shape matches Go's agent.AgentOption (capitalized fields, no json tags).
export interface AgentOption {
  ID: number | string;
  Value: string | number;
}

// Methods are typed structurally; adding a new component type does not require
// touching this file. Only methods listed here are recognised by the bridge.
//
// `*Child` variants exist on container handles (currently `Table`) that route
// the call to a registered child by id rather than acting on themselves. They
// share the verb of the leaf method but live under a distinct name so callers
// always know whether they are addressing the container or one of its
// children.
export interface AgentMethodMap {
  search?: (text: string) => void;
  select?: (...ids: (number | string)[]) => void;
  remove?: (id: number | string) => void;
  setValue?: (value: string | number) => void;
  click?: () => void;
  open?: () => void;
  close?: () => void;
  deleted?: () => void;
  getOptions?: (max?: number) => AgentOption[];
  getValue?: () => AgentOption | AgentOption[] | undefined;
  // Child-routing variants (Table → cell). The first argument is the child id
  // (`<tableID>:<cellID>` resolves to its `<cellID>`); remaining args mirror
  // the leaf method.
  setValueChild?: (childID: number | string, value: string | number) => void;
  searchChild?: (childID: number | string, text: string) => void;
  getOptionsChild?: (childID: number | string, max?: number) => AgentOption[];
  clickChild?: (childID: number | string) => void;
}

export type AgentMethodName = keyof AgentMethodMap;

const AGENT_METHOD_NAMES: AgentMethodName[] = [
  "search",
  "select",
  "remove",
  "setValue",
  "click",
  "open",
  "close",
  "deleted",
  "getOptions",
  "getValue",
  "setValueChild",
  "searchChild",
  "getOptionsChild",
  "clickChild",
];

export interface AgentHandle extends AgentMethodMap {
  id: number;
  type: string;
  label: string;
}

// Summary entry returned by Agent.list — identity-only, no method surface.
export interface AgentComponentSummary {
  ID: number;
  Type: string;
  Label: string;
}

// Full entry returned by Agent.describe and embedded in the page snapshot.
export interface AgentPageComponent extends AgentComponentSummary {
  Methods: string[];
}

export interface AgentListFilter {
  type?: string;
  label?: string;
}

export interface AgentRegistry {
  register: (handle: AgentHandle) => () => void;
  list: (filter?: AgentListFilter) => AgentComponentSummary[];
  get: (id: number) => AgentHandle | undefined;
  describe: () => AgentPageComponent[];
}

export const isAgentEnabled = (): boolean => {
  if (typeof window === "undefined") { return false; }
  if (window.ENABLE_UI_AGENT === 1 || globalThis._isLocal) { return true; }
  return false;
};

// Shared map of registered handles. Exported so the WS bridge can resolve
// handle ids without going back through the public `Agent` facade.
export const agentHandles = new Map<number, AgentHandle>();

const methodsFor = (handle: AgentHandle): string[] => {
  const out: string[] = [];
  for (const name of AGENT_METHOD_NAMES) {
    if (typeof handle[name] === "function") { out.push(name); }
  }
  return out;
};

export const Agent: AgentRegistry & { getPageContent: () => Promise<{ Components: AgentPageComponent[]; HTML: string }> } = {
  register(handle) {
    if (!isAgentEnabled()) {
      return () => {};
    }
    agentHandles.set(handle.id, handle);
    return () => {
      const current = agentHandles.get(handle.id);
      // Only delete if the entry still points to this handle, in case HMR replaced it.
      if (current === handle) {
        agentHandles.delete(handle.id);
      }
    };
  },

  list(filter) {
    const result: AgentComponentSummary[] = [];
    const labelNeedle = filter?.label?.toLowerCase();
    for (const handle of agentHandles.values()) {
      if (filter?.type && handle.type !== filter.type) continue;
      if (labelNeedle && !handle.label.toLowerCase().includes(labelNeedle)) continue;
      result.push({ ID: handle.id, Type: handle.type, Label: handle.label });
    }
    return result;
  },

  get(id) {
    return agentHandles.get(id);
  },

  describe() {
    const out: AgentPageComponent[] = [];
    for (const handle of agentHandles.values()) {
      out.push({
        ID: handle.id,
        Type: handle.type,
        Label: handle.label,
        Methods: methodsFor(handle),
      });
    }
    return out;
  },

  // Returns the same payload the WS bridge sends to the backend (capitalized keys
  // to match Go field names). Available in devtools as `__agent.getPageContent()`.
  // Component values are read by the agent from the HTML snapshot via data-value /
  // selected attributes — this payload only carries identity and method surface.
  async getPageContent() {
    const [{ default: DOMPurify }, { default: normalize }] = await Promise.all([
      import("dompurify"),
      import("normalize-html-whitespace"),
    ]);

    const HTML = normalize(DOMPurify.sanitize(document.body.outerHTML, {
      FORBID_TAGS: ["script", "style"],
      ALLOW_DATA_ATTR: true,
    }));

    const Components: AgentPageComponent[] = [];
    for (const handle of agentHandles.values()) {
      Components.push({
        ID: handle.id,
        Type: handle.type,
        Label: handle.label,
        Methods: methodsFor(handle),
      });
    }

    return { Components, HTML };
  },
};

if (typeof window !== "undefined") {
  window.__agent = Agent;
}
