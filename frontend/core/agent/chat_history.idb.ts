// Local persistence for the in-app agent chat. Backed by Dexie/IndexedDB so
// the widget can re-render past messages instantly on open without waiting
// for a backend round-trip. Server-side history (agent_messages table in
// ScyllaDB) is the source of truth — this is a UI-only cache scoped to the
// browser tab (rows are keyed by the same TabID the WS uses).

import Dexie from 'dexie'
import { Env } from '$core/env'

const LOG_PREFIX = '[agent-chat:idb]'
const AGENT_CHAT_DB_PREFIX = 'agent_chat'
const AGENT_CHAT_DB_VERSION = 1

// Role values mirror the backend's agent.RoleUser / agent.RoleAgent. Status
// rows (role=3) are a frontend-only concept — the backend doesn't persist
// progress traces in ScyllaDB, but the widget keeps them in IndexedDB so the
// user can see the full action history after reload.
export const AGENT_ROLE_USER = 1
export const AGENT_ROLE_AGENT = 2
export const AGENT_ROLE_STATUS = 3

export type AgentRole =
  | typeof AGENT_ROLE_USER
  | typeof AGENT_ROLE_AGENT
  | typeof AGENT_ROLE_STATUS

// AgentChatRow is one persisted message. `id` is auto-incremented by Dexie;
// `tabID` partitions rows so each browser tab gets its own visible history
// (matching the sessionStorage scope of the TabID itself). `timestamp` is
// the server's reply timestamp when known, otherwise the local send time.
export interface AgentChatRow {
  id?: number
  tabID: string
  role: AgentRole
  message: string
  summary?: string
  timestamp: number
  // Pending = waiting for the backend's agentReply (used to render a spinner
  // next to the optimistic user row). Cleared once the row is finalized.
  pending?: boolean
}

class AgentChatDatabase extends Dexie {
  messages!: Dexie.Table<AgentChatRow, number>

  constructor(databaseName: string) {
    super(databaseName)
    this.version(AGENT_CHAT_DB_VERSION).stores({
      // Compound index `[tabID+timestamp]` lets us slice one tab's history in
      // chronological order without scanning the whole table.
      messages: '++id, tabID, [tabID+timestamp]',
    })
  }
}

const databasesByName = new Map<string, AgentChatDatabase>()

const makeDatabaseName = (companyID: number, env: string): string => {
  return `${companyID || 0}_${AGENT_CHAT_DB_PREFIX}_${env || '000000'}`
}

const getDatabase = (): AgentChatDatabase => {
  const name = makeDatabaseName(Env.getEmpresaID(), Env.enviroment || 'main')
  const existing = databasesByName.get(name)
  if (existing) return existing
  const created = new AgentChatDatabase(name)
  databasesByName.set(name, created)
  return created
}

export const loadAgentChatHistory = async (tabID: string): Promise<AgentChatRow[]> => {
  if (!tabID) return []
  try {
    const rows = await getDatabase().messages
      .where('[tabID+timestamp]')
      .between([tabID, Dexie.minKey], [tabID, Dexie.maxKey])
      .toArray()
    return rows
  } catch (error) {
    console.warn(`${LOG_PREFIX} load failed`, error)
    return []
  }
}

export const appendAgentChatMessage = async (row: AgentChatRow): Promise<number> => {
  try {
    return (await getDatabase().messages.add(row)) as number
  } catch (error) {
    console.warn(`${LOG_PREFIX} append failed`, error)
    return 0
  }
}

export const updateAgentChatMessage = async (
  id: number,
  patch: Partial<AgentChatRow>,
): Promise<void> => {
  if (!id) return
  try {
    await getDatabase().messages.update(id, patch)
  } catch (error) {
    console.warn(`${LOG_PREFIX} update failed`, error)
  }
}

export const clearAgentChatHistory = async (tabID: string): Promise<void> => {
  if (!tabID) return
  try {
    await getDatabase().messages.where('tabID').equals(tabID).delete()
  } catch (error) {
    console.warn(`${LOG_PREFIX} clear failed`, error)
  }
}
