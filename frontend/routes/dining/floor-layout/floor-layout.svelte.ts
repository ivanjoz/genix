import type { IDiningLayout } from '../dining.ts'
import type { LayoutTool } from './floor-layout.ts'

const MAX_HISTORY = 100

// The editor's state: the draft being drawn, the selection, the tool and the undo history. The history
// holds whole snapshots of the draft: a plan is small, and a snapshot cannot drift from what it undoes.
export class FloorLayoutEditorState {
  draft = $state<IDiningLayout>()
  selectedShapeIDs = $state<number[]>([])
  tool = $state<LayoutTool>('select')
  cellSize = $state(16)
  // The draft differs from what was loaded or saved.
  isDirty = $state(false)
  #history: string[] = []
  #historyIndex = $state(0)
  #historyLength = $state(0)

  canUndo = $derived(this.#historyIndex > 0)
  canRedo = $derived(this.#historyIndex < this.#historyLength - 1)

  // Starts editing a layout: a fresh history, nothing selected.
  load(layout: IDiningLayout) {
    this.draft = structuredClone(layout)
    this.selectedShapeIDs = []
    this.tool = 'select'
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
}
