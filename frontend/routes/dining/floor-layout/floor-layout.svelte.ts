import type { IDiningLayout, IGridPoint } from '../dining.ts'
import { deleteShapes, duplicateShapes, rotateShapes } from './floor-layout.ts'

const MAX_HISTORY = 100

// The editor's state: the draft being drawn, the selection, the zoom and the undo history. The history
// holds whole snapshots of the draft: a plan is small, and a snapshot cannot drift from what it undoes.
export class FloorLayoutEditorState {
  draft = $state<IDiningLayout>()
  selectedShapeIDs = $state<number[]>([])
  cellSize = $state(16)
  // Where the viewport sits on the open sheet when the shapes do not push it (editorCanvasRegion).
  viewAnchor = $state<IGridPoint>({ x: 0, y: 0 })
  // The draft differs from what was loaded or saved.
  isDirty = $state(false)
  #history: string[] = []
  #historyIndex = $state(0)
  #historyLength = $state(0)

  canUndo = $derived(this.#historyIndex > 0)
  canRedo = $derived(this.#historyIndex < this.#historyLength - 1)

  // Starts editing a layout: a fresh history, nothing selected. A saved layout comes back with its
  // shapes trimmed to the margin; the anchor moves with them so the plan stays put on screen.
  load(layout: IDiningLayout, viewAnchor: IGridPoint = { x: 0, y: 0 }) {
    this.draft = structuredClone(layout)
    this.viewAnchor = viewAnchor
    this.selectedShapeIDs = []
    this.#history = [JSON.stringify(layout)]
    this.#historyIndex = 0
    this.#historyLength = 1
    this.isDirty = false
  }

  // Records the draft as it is now, after a change: a drag records once, on release.
  checkpoint() {
    if (!this.draft) return
    const snapshot = JSON.stringify($state.snapshot(this.draft))
    if (snapshot === this.#history[this.#historyIndex]) return
    this.#history = [...this.#history.slice(0, this.#historyIndex + 1), snapshot].slice(-MAX_HISTORY)
    this.#historyIndex = this.#history.length - 1
    this.#historyLength = this.#history.length
    this.isDirty = true
  }

  undo() { this.#restore(this.#historyIndex - 1) }
  redo() { this.#restore(this.#historyIndex + 1) }

  #restore(index: number) {
    if (index < 0 || index >= this.#history.length) return
    this.#historyIndex = index
    this.draft = JSON.parse(this.#history[index])
    // A restored draft may lack shapes that were selected.
    const shapeIDs = new Set((this.draft?.Shapes || []).map((shape) => shape.ShapeID))
    this.selectedShapeIDs = this.selectedShapeIDs.filter((shapeID) => shapeIDs.has(shapeID))
    this.isDirty = true
  }

  selectedShapes() {
    return (this.draft?.Shapes || []).filter((shape) => this.selectedShapeIDs.includes(shape.ShapeID))
  }

  // The actions on the selection, shared by the tool column and the plan's keyboard shortcuts.
  duplicateSelection() {
    if (!this.draft || !this.selectedShapeIDs.length) return
    this.selectedShapeIDs = duplicateShapes(this.draft, this.selectedShapeIDs)
    this.checkpoint()
  }

  rotateSelection() {
    if (!this.draft || !this.selectedShapeIDs.length) return
    rotateShapes(this.draft, this.selectedShapeIDs)
    this.checkpoint()
  }

  deleteSelection() {
    if (!this.draft || !this.selectedShapeIDs.length) return
    deleteShapes(this.draft, this.selectedShapeIDs)
    this.selectedShapeIDs = []
    this.checkpoint()
  }
}

// The layout last opened in the editor, per browser: a reload reopens it instead of an empty page.
const LAST_LAYOUT_STORAGE_KEY = 'floor_layout_last_layout_id'
export const lastOpenedLayoutID = (): number => Number(localStorage.getItem(LAST_LAYOUT_STORAGE_KEY)) || 0
export const rememberOpenedLayout = (layoutID: number) => { localStorage.setItem(LAST_LAYOUT_STORAGE_KEY, String(layoutID)) }
