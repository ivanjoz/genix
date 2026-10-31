// Drag & drop of items between ordered lists, on the browser's native drag events, as Svelte
// attachments: `{@attach draggableItem(() => payload)}` on each item (which also carries
// `data-drag-id`) and `{@attach dropZone({ accepts, onDrop })}` on each list. Touch browsers fire no
// drag events: dragging is a desktop feature.
//
// Marks for the consumer's CSS (Tailwind `data-[drop-before]:…`): the zone under the pointer gets
// `data-drop-active`; the item the drop would land before gets `data-drop-before`, or the last item
// `data-drop-after`; the dragged item has `data-dragging`.

// The payload of the drag in progress. dataTransfer can't be read during dragover, so it lives here.
let draggedPayload: unknown = null

const clearDropMarks = () => {
  for (const markedElement of document.querySelectorAll('[data-drop-before],[data-drop-after],[data-drop-active]')) {
    const element = markedElement as HTMLElement
    delete element.dataset.dropBefore
    delete element.dataset.dropAfter
    delete element.dataset.dropActive
  }
}

export const draggableItem = (getPayload: () => unknown) => (element: HTMLElement) => {
  element.draggable = true
  const onDragStart = (dragEvent: DragEvent) => {
    draggedPayload = getPayload()
    // Firefox starts no drag without data.
    dragEvent.dataTransfer?.setData('text/plain', '')
    if (dragEvent.dataTransfer) { dragEvent.dataTransfer.effectAllowed = 'move' }
    element.dataset.dragging = ''
  }
  const onDragEnd = () => {
    draggedPayload = null
    delete element.dataset.dragging
    clearDropMarks()
  }
  element.addEventListener('dragstart', onDragStart)
  element.addEventListener('dragend', onDragEnd)
  return () => {
    element.removeEventListener('dragstart', onDragStart)
    element.removeEventListener('dragend', onDragEnd)
  }
}

// The neighbours a drop lands between, by their data-drag-id: '' at either end of the list.
export interface IDropPosition {
  previousID: string
  nextID: string
}

interface IDropZoneOptions<TPayload> {
  accepts: (payload: unknown) => payload is TPayload
  onDrop: (payload: TPayload, dropPosition: IDropPosition) => void
}

export const dropZone = <TPayload>(options: IDropZoneOptions<TPayload>) => (zoneElement: HTMLElement) => {
  // The zone's items in order, the dragged one left out: the drop lands between the others.
  const listItems = () => [...zoneElement.querySelectorAll<HTMLElement>('[data-drag-id]')]
    .filter((itemElement) => itemElement.dataset.dragging === undefined)

  // The first item whose middle is below the pointer: the drop goes before it.
  const findNextItem = (pointerY: number) => listItems().find((itemElement) => {
    const itemBox = itemElement.getBoundingClientRect()
    return pointerY < itemBox.top + itemBox.height / 2
  })

  const onDragOver = (dragEvent: DragEvent) => {
    if (!options.accepts(draggedPayload)) { return }
    dragEvent.preventDefault()
    clearDropMarks()
    zoneElement.dataset.dropActive = ''
    const nextItem = findNextItem(dragEvent.clientY)
    if (nextItem) {
      nextItem.dataset.dropBefore = ''
    } else {
      const lastItem = listItems().at(-1)
      if (lastItem) { lastItem.dataset.dropAfter = '' }
    }
  }

  const onDragLeave = (dragEvent: DragEvent) => {
    if (!zoneElement.contains(dragEvent.relatedTarget as Node | null)) { clearDropMarks() }
  }

  const onDrop = (dragEvent: DragEvent) => {
    const payload = draggedPayload
    if (!options.accepts(payload)) { return }
    dragEvent.preventDefault()
    const items = listItems()
    const nextItem = findNextItem(dragEvent.clientY)
    const nextIndex = nextItem ? items.indexOf(nextItem) : items.length
    clearDropMarks()
    options.onDrop(payload, {
      previousID: items[nextIndex - 1]?.dataset.dragId || '',
      nextID: nextItem?.dataset.dragId || '',
    })
  }

  zoneElement.addEventListener('dragover', onDragOver)
  zoneElement.addEventListener('dragleave', onDragLeave)
  zoneElement.addEventListener('drop', onDrop)
  return () => {
    zoneElement.removeEventListener('dragover', onDragOver)
    zoneElement.removeEventListener('dragleave', onDragLeave)
    zoneElement.removeEventListener('drop', onDrop)
  }
}
