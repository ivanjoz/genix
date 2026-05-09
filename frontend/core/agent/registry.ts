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

export interface AgentOption {
  id: number | string;
  value: string | number;
}

// Methods are typed structurally; adding a new component type does not require
// touching this file. Only methods listed here are recognised by the bridge.
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
];

export interface AgentHandle extends AgentMethodMap {
  id: number;
  type: string;
  label: string;
}

export interface AgentPageComponent {
  ID: number;
  Type: string;
  Label: string;
  Methods: string[];
}

export interface AgentRegistry {
  register: (handle: AgentHandle) => () => void;
  list: (filter?: { type?: string; label?: string }) => Array<{ id: number; type: string; label: string }>;
  get: (id: number) => AgentHandle | undefined;
  describe: () => Array<{ id: number; type: string; label: string; methods: string[] }>;
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
    const result: Array<{ id: number; type: string; label: string }> = [];
    const labelNeedle = filter?.label?.toLowerCase();
    for (const handle of agentHandles.values()) {
      if (filter?.type && handle.type !== filter.type) continue;
      if (labelNeedle && !handle.label.toLowerCase().includes(labelNeedle)) continue;
      result.push({ id: handle.id, type: handle.type, label: handle.label });
    }
    return result;
  },

  get(id) {
    return agentHandles.get(id);
  },

  describe() {
    const out: Array<{ id: number; type: string; label: string; methods: string[] }> = [];
    for (const handle of agentHandles.values()) {
      out.push({
        id: handle.id,
        type: handle.type,
        label: handle.label,
        methods: methodsFor(handle),
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
