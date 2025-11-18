<script lang="ts" generics="T">
  import { onMount } from "svelte"
  import { browser } from "$app/environment"
  import {
    ChangeSource,
    clearFormat,
    createDomToModelContext,
    createDomToModelContextWithConfig,
    createEditor,
    domToContentModel,
    editTable,
    getFormatState,
    insertTable,
    setAlignment,
    setBackgroundColor,
    setFontName,
    setFontSize,
    setHeadingLevel,
    setTextColor,
    toggleBold,
    toggleBullet,
    toggleItalic,
    toggleNumbering,
    toggleStrikethrough,
    toggleUnderline
  } from "roosterjs"
  import { TableEditPlugin } from "roosterjs"
  import type {
    ContentModelDocument,
    ContentModelFormatState,
    EditorPlugin,
    IEditor,
    PluginEvent,
    TableOperation
  } from "roosterjs"

  type HeadingLevel = 0 | 1 | 2 | 3 | 4 | 5 | 6

  const { saveOn, save, css }: {
    saveOn: T
    save: keyof T
    css?: string
  } = $props()

  const getInitialValue = () =>
    (saveOn && save && ((saveOn[save] as string) ?? "")) || ""

  let editorRoot = $state<HTMLDivElement | null>(null)
  let editorContainer = $state<HTMLDivElement | null>(null)
  let editor = $state<IEditor | null>(null)
  let editorValue = $state<string>(getInitialValue())
  let formatState = $state<ContentModelFormatState>({})
  let isInTable = $state(false)
  let fontSizeOptions = $state<string[]>([
    "8pt",
    "9pt",
    "10pt",
    "11pt",
    "12pt",
    "14pt",
    "16pt",
    "18pt",
    "20pt",
    "24pt",
    "28pt",
    "32pt",
  ])
  let fontSizeSelection = $state<string>("16pt")
  let headingSelection = $state<HeadingLevel>(0)
  let textColorValue = $state<string>("#000000")
  let backgroundColorValue = $state<string>("#ffffff")
  let fontFamilySelection = $state<string>("Arial")
  let lastSyncedValue = getInitialValue()
  let pendingFormatFrame: number | null = null
  let showImageDialog = $state(false)
  let showTableDialog = $state(false)
  let imageUrl = $state("")
  let tableRows = $state(3)
  let tableCols = $state(3)

  const headingOptions: Array<{ label: string; value: HeadingLevel }> = [
    { label: "Paragraph", value: 0 },
    { label: "H1", value: 1 },
    { label: "H2", value: 2 },
    { label: "H3", value: 3 },
    { label: "H4", value: 4 },
    { label: "H5", value: 5 },
    { label: "H6", value: 6 }
  ]

  const fontFamilyOptions = [
    "Arial",
    "Helvetica",
    "Times New Roman",
    "Courier New",
    "Verdana",
    "Georgia",
    "Palatino",
    "Garamond",
    "Comic Sans MS",
    "Trebuchet MS",
    "Impact"
  ]

  const getEditorCore = () => {
    const current = editor as (IEditor & { core?: any }) | null
    return current?.core
  }

  const ensureFontSizeOption = (size: string) => {
    if (!size) {
      return
    }
    if (!fontSizeOptions.includes(size)) {
      fontSizeOptions = [...fontSizeOptions, size]
    }
  }

  const createModelFromHtml = (
    html: string,
    core?: ReturnType<typeof getEditorCore>
  ): ContentModelDocument | undefined => {
    if (!browser) {
      return undefined
    }

    const normalized = html?.trim() ? html : "<p></p>"
    const parser = new DOMParser()
    const doc = parser.parseFromString(normalized, "text/html")
    const body = doc.body
    if (!body) {
      return undefined
    }

    const context = core
      ? createDomToModelContextWithConfig(
          core.environment.domToModelSettings.calculated,
          core.api.createEditorContext(core, false)
        )
      : createDomToModelContext()

    return domToContentModel(body, context)
  }

  const sanitizeHtml = (html: string): string => {
    if (!browser) {
      return html
    }

    const parser = new DOMParser()
    const doc = parser.parseFromString(html, "text/html")
    
    // Remove unwanted styles from divs
    const divs = doc.querySelectorAll('div')
    divs.forEach(div => {
      const htmlDiv = div as HTMLElement
      htmlDiv.style.removeProperty('margin-top')
      htmlDiv.style.removeProperty('margin-bottom')
      htmlDiv.style.removeProperty('font-family')
      htmlDiv.style.removeProperty('font-size')
      htmlDiv.style.removeProperty('color')
    })

    // Remove unwanted styles from table cells
    const cells = doc.querySelectorAll('td, th')
    cells.forEach(cell => {
      const htmlCell = cell as HTMLElement
      htmlCell.style.removeProperty('width')
      htmlCell.style.removeProperty('height')
      htmlCell.style.removeProperty('border-width')
      htmlCell.style.removeProperty('border-style')
      htmlCell.style.removeProperty('border-color')
    })

    return doc.body.innerHTML
  }

  const applyHtmlToEditor = (html: string) => {
    const core = getEditorCore()
    if (!core) {
      return
    }

    // Sanitize HTML to remove unwanted styles before creating model
    const sanitizedHtml = sanitizeHtml(html)
    const model = createModelFromHtml(sanitizedHtml, core)
    if (!model) {
      return
    }

    core.api.setContentModel(core, model, { ignoreSelection: true })
    core.api.triggerEvent(
      core,
      {
        eventType: "contentChanged",
        source: ChangeSource.SetContent
      },
      true
    )
    
    // Also remove styles from the DOM after applying
    setTimeout(() => {
      if (editorRoot) {
        removeUnwantedStyles(editorRoot)
      }
    }, 0)
  }

  const removeUnwantedStyles = (element: HTMLElement) => {
    // Remove unwanted styles from divs
    const divs = element.querySelectorAll('div')
    divs.forEach(div => {
      const htmlDiv = div as HTMLElement
      htmlDiv.style.removeProperty('margin-top')
      htmlDiv.style.removeProperty('margin-bottom')
      htmlDiv.style.removeProperty('font-family')
      htmlDiv.style.removeProperty('font-size')
      htmlDiv.style.removeProperty('color')
    })

    // Remove unwanted styles from table cells
    const cells = element.querySelectorAll('td, th')
    cells.forEach(cell => {
      const htmlCell = cell as HTMLElement
      htmlCell.style.removeProperty('width')
      htmlCell.style.removeProperty('height')
      htmlCell.style.removeProperty('border-width')
      htmlCell.style.removeProperty('border-style')
      htmlCell.style.removeProperty('border-color')
    })
  }

  const syncEditorHtml = () => {
    if (!editorRoot) {
      return
    }

    // Remove unwanted inline styles before syncing
    removeUnwantedStyles(editorRoot)

    const nextValue = editorRoot.innerHTML
    if (nextValue !== editorValue) {
      editorValue = nextValue
    }
  }

  const checkIfInTable = () => {
    if (!editor || !editorRoot) {
      isInTable = false
      return
    }
    
    const selection = editor.getDOMSelection()
    if (!selection) {
      isInTable = false
      return
    }
    
    // Check if selection is a table selection
    if (selection.type === 'table') {
      isInTable = true
      return
    }
    
    // Check if range selection is inside a table
    if (selection.type === 'range') {
      const range = selection.range
      
      // Check both startContainer and commonAncestorContainer
      const nodesToCheck: Node[] = [
        range.startContainer,
        range.commonAncestorContainer
      ]
      
      for (const startNode of nodesToCheck) {
        let node: Node | null = startNode
        
        // Walk up the DOM tree to find if we're inside a table
        while (node && node !== editorRoot) {
          if (node.nodeType === Node.ELEMENT_NODE) {
            const element = node as HTMLElement
            const tagName = element.tagName?.toUpperCase()
            if (tagName === 'TABLE' || tagName === 'TD' || tagName === 'TH' || tagName === 'TR' || tagName === 'TBODY' || tagName === 'THEAD' || tagName === 'TFOOT') {
              isInTable = true
              return
            }
          }
          node = node.parentNode
        }
      }
    }
    
    isInTable = false
  }

  const refreshFormatState = () => {
    if (!editor) {
      formatState = {}
      isInTable = false
      return
    }
    formatState = getFormatState(editor) ?? {}
    const nextFontSize = formatState.fontSize
    if (nextFontSize) {
      ensureFontSizeOption(nextFontSize)
      fontSizeSelection = nextFontSize
    }
    const nextHeading = (formatState.headingLevel ?? 0) as HeadingLevel
    headingSelection = nextHeading
    const nextFontFamily = formatState.fontName
    if (nextFontFamily) {
      fontFamilySelection = nextFontFamily
    }
    checkIfInTable()
  }

  const scheduleFormatStateRefresh = () => {
    if (!browser) {
      refreshFormatState()
      return
    }
    if (pendingFormatFrame !== null) {
      cancelAnimationFrame(pendingFormatFrame)
    }
    pendingFormatFrame = requestAnimationFrame(() => {
      pendingFormatFrame = null
      refreshFormatState()
    })
  }

  const createSyncPlugin = (): EditorPlugin => {
    let pluginEditor: IEditor | null = null
    let mutationObserver: MutationObserver | null = null

    return {
      getName: () => "svelte-sync",
      initialize: (ed: IEditor) => {
        pluginEditor = ed
        editor = ed
        syncEditorHtml()
        refreshFormatState()

        // Set up MutationObserver to remove unwanted styles as they're added
        if (editorRoot && browser) {
          mutationObserver = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
              mutation.addedNodes.forEach((node) => {
                if (node.nodeType === Node.ELEMENT_NODE) {
                  const element = node as HTMLElement
                  removeUnwantedStyles(element)
                  
                  // Also check if the node itself is a div or table cell
                  if (element.tagName === 'DIV' || element.tagName === 'TD' || element.tagName === 'TH') {
                    if (element.tagName === 'DIV') {
                      element.style.removeProperty('margin-top')
                      element.style.removeProperty('margin-bottom')
                      element.style.removeProperty('font-family')
                      element.style.removeProperty('font-size')
                      element.style.removeProperty('color')
                    } else {
                      element.style.removeProperty('width')
                      element.style.removeProperty('height')
                      element.style.removeProperty('border-width')
                      element.style.removeProperty('border-style')
                      element.style.removeProperty('border-color')
                    }
                  }
                }
              })
            })
          })

          mutationObserver.observe(editorRoot, {
            childList: true,
            subtree: true,
            attributes: true,
            attributeFilter: ['style']
          })
        }
      },
      dispose: () => {
        if (mutationObserver) {
          mutationObserver.disconnect()
          mutationObserver = null
        }
        if (pluginEditor && editor === pluginEditor) {
          editor = null
        }
        pluginEditor = null
      },
      onPluginEvent: (event: PluginEvent) => {
        if (event.eventType === "contentChanged") {
          syncEditorHtml()
          // Remove unwanted styles after content changes
          if (editorRoot) {
            setTimeout(() => {
              removeUnwantedStyles(editorRoot as HTMLElement)
            }, 0)
          }
        }
        if (event.eventType === "contentChanged" || event.eventType === "selectionChanged") {
          // Immediately check table state on selection changes for better responsiveness
          if (event.eventType === "selectionChanged") {
            checkIfInTable()
          }
          scheduleFormatStateRefresh()
        }
      }
    }
  }

  const withEditor = (cb: (instance: IEditor) => void) => {
    if (!editor) {
      return
    }
    editor.focus()
    cb(editor)
    scheduleFormatStateRefresh()
  }

  const handleHeadingChange = (value: HeadingLevel) => {
    headingSelection = value
    withEditor(instance => setHeadingLevel(instance, value))
  }

  const handleFontSizeChange = (size: string) => {
    fontSizeSelection = size
    withEditor(instance => setFontSize(instance, size))
  }

  const handleFontFamilyChange = (family: string) => {
    fontFamilySelection = family
    withEditor(instance => setFontName(instance, family))
  }

  const handleTextColorChange = (color: string) => {
    textColorValue = color
    withEditor(instance => setTextColor(instance, color))
  }

  const handleBackgroundColorChange = (color: string) => {
    backgroundColorValue = color
    withEditor(instance => setBackgroundColor(instance, color))
  }

  const handleAlignment = (alignment: "left" | "center" | "right" | "justify") => {
    withEditor(instance => setAlignment(instance, alignment))
  }

  const handleInsertTable = () => {
    withEditor(instance => insertTable(instance, tableRows, tableCols))
    showTableDialog = false
    tableRows = 3
    tableCols = 3
  }

  const handleInsertImage = () => {
    if (imageUrl.trim() && editor) {
      const core = getEditorCore()
      if (core) {
        const img = document.createElement('img')
        img.src = imageUrl.trim()
        img.style.maxWidth = '100%'
        core.api.insertNode(core, img)
        showImageDialog = false
        imageUrl = ""
      }
    }
  }

  const handleInsertHR = () => {
    if (editor) {
      const core = getEditorCore()
      if (core) {
        const hr = document.createElement('hr')
        core.api.insertNode(core, hr)
      }
    }
  }

  onMount(() => {
    if (!browser || !editorRoot) {
      return
    }

    const initialModel = createModelFromHtml(editorValue)
    const syncPlugin = createSyncPlugin()
    // TableEditPlugin enables table resizing, moving, and other table editing features
    // Features enabled by default: TableResizer, TableMover, TableSelector, CellResizer, etc.
    // The plugin automatically detects tables and adds resize handles on hover
    const tableEditPlugin = new TableEditPlugin()
    const instance = createEditor(editorRoot, [syncPlugin, tableEditPlugin], initialModel)
    editor = instance

    // Add click listener for immediate table detection
    const handleClick = () => {
      // Use setTimeout to ensure selection is updated after click
      setTimeout(() => {
        checkIfInTable()
      }, 0)
    }

    editorRoot.addEventListener('click', handleClick)
    editorRoot.addEventListener('focus', handleClick)

    return () => {
      if (pendingFormatFrame !== null) {
        cancelAnimationFrame(pendingFormatFrame)
        pendingFormatFrame = null
      }
      editorRoot?.removeEventListener('click', handleClick)
      editorRoot?.removeEventListener('focus', handleClick)
      // Plugins are automatically disposed when editor is disposed
      instance.dispose()
      editor = null
    }
  })

  $effect(() => {
    if (!saveOn || !save) {
      return
    }

    if (editorValue !== lastSyncedValue) {
      lastSyncedValue = editorValue
      saveOn[save] = editorValue as NonNullable<T>[keyof T]
    }
  })

  $effect(() => {
    if (!browser || !saveOn || !save) {
      return
    }

    const incoming = ((saveOn[save] as string) ?? "") || ""
    if (incoming !== editorValue) {
      lastSyncedValue = incoming
      editorValue = incoming
      if (editor) {
        applyHtmlToEditor(incoming)
      }
    }
  })
</script>

<div class={`rooster-wrapper ${css ?? ""}`}>
  {#if browser}
    <div class="rooster-toolbar">
      <div class="rooster-group">
        <button
          type="button"
          disabled={!editor}
          aria-label="Undo"
          onclick={() => {
            if (editor) {
              const core = getEditorCore()
              if (core) {
                core.api.undo(core)
                scheduleFormatStateRefresh()
              }
            }
          }}>
          ↶
        </button>
        <button
          type="button"
          disabled={!editor}
          aria-label="Redo"
          onclick={() => {
            if (editor) {
              const core = getEditorCore()
              if (core) {
                core.api.redo(core)
                scheduleFormatStateRefresh()
              }
            }
          }}>
          ↷
        </button>
      </div>

      <div class="rooster-group">
        <label class="rooster-select">
          <span>Size</span>
          <select
            bind:value={fontSizeSelection}
            disabled={!editor}
            onchange={event =>
              handleFontSizeChange((event.currentTarget as HTMLSelectElement).value)}>
            {#each fontSizeOptions as size}
              <option value={size}>{size}</option>
            {/each}
          </select>
        </label>
      </div>

      <div class="rooster-group">
        <button
          type="button"
          class:active={!!formatState.isBold}
          disabled={!editor}
          aria-label="Bold"
          onclick={() => withEditor(toggleBold)}>
          <strong>B</strong>
        </button>
        <button
          type="button"
          class:active={!!formatState.isItalic}
          disabled={!editor}
          aria-label="Italic"
          onclick={() => withEditor(toggleItalic)}>
          <em>I</em>
        </button>
        <button
          type="button"
          class:active={!!formatState.isUnderline}
          disabled={!editor}
          aria-label="Underline"
          onclick={() => withEditor(toggleUnderline)}>
          <u>U</u>
        </button>
        <button
          type="button"
          class:active={!!formatState.isStrikeThrough}
          disabled={!editor}
          aria-label="Strikethrough"
          onclick={() => withEditor(toggleStrikethrough)}>
          <s>S</s>
        </button>
      </div>

      <div class="rooster-group">
        <label class="rooster-color" aria-label="Text color">
          <span>Text</span>
          <input
            type="color"
            bind:value={textColorValue}
            disabled={!editor}
            oninput={event =>
              handleTextColorChange((event.currentTarget as HTMLInputElement).value)} />
        </label>
        <label class="rooster-color" aria-label="Background color">
          <span>Fill</span>
          <input
            type="color"
            bind:value={backgroundColorValue}
            disabled={!editor}
            oninput={event =>
              handleBackgroundColorChange((event.currentTarget as HTMLInputElement).value)} />
        </label>
      </div>

      <div class="rooster-group">
        <button
          type="button"
          class:active={formatState.textAlign === "left" || !formatState.textAlign}
          disabled={!editor}
          aria-label="Align left"
          onclick={() => handleAlignment("left")}>
          ⬅
        </button>
        <button
          type="button"
          class:active={formatState.textAlign === "center"}
          disabled={!editor}
          aria-label="Align center"
          onclick={() => handleAlignment("center")}>
          ⬍
        </button>
        <button
          type="button"
          class:active={formatState.textAlign === "right"}
          disabled={!editor}
          aria-label="Align right"
          onclick={() => handleAlignment("right")}>
          ➡
        </button>
        <button
          type="button"
          class:active={formatState.textAlign === "justify"}
          disabled={!editor}
          aria-label="Justify"
          onclick={() => handleAlignment("justify")}>
          ☰
        </button>
      </div>

      <div class="rooster-group">
        <button
          type="button"
          class:active={!!formatState.isBullet}
          disabled={!editor}
          aria-label="Bulleted list"
          onclick={() => withEditor(toggleBullet)}>
          ••
        </button>
        <button
          type="button"
          class:active={!!formatState.isNumbering}
          disabled={!editor}
          aria-label="Numbered list"
          onclick={() => withEditor(toggleNumbering)}>
          1.
        </button>
      </div>

      <div class="rooster-group">
        <button
          type="button"
          disabled={!editor}
          aria-label="Insert horizontal rule"
          onclick={handleInsertHR}>
          ─
        </button>
        <button
          type="button"
          disabled={!editor}
          aria-label="Insert image"
          onclick={() => showImageDialog = true}>
          🖼
        </button>
        <button
          type="button"
          disabled={!editor}
          aria-label="Insert table"
          onclick={() => showTableDialog = true}>
          ⊞
        </button>
        <button
          type="button"
          disabled={!editor}
          aria-label="Clear format"
          onclick={() => withEditor(clearFormat)}>
          Clear
        </button>
      </div>

      {#if isInTable}
        <div class="rooster-group">
          <button
            type="button"
            disabled={!editor}
            aria-label="Add row above"
            onclick={() => withEditor(instance => editTable(instance, 'insertAbove'))}>
            ⬆ Row
          </button>
          <button
            type="button"
            disabled={!editor}
            aria-label="Add row below"
            onclick={() => withEditor(instance => editTable(instance, 'insertBelow'))}>
            ⬇ Row
          </button>
          <button
            type="button"
            disabled={!editor}
            aria-label="Add column left"
            onclick={() => withEditor(instance => editTable(instance, 'insertLeft'))}>
            ⬅ Col
          </button>
          <button
            type="button"
            disabled={!editor}
            aria-label="Add column right"
            onclick={() => withEditor(instance => editTable(instance, 'insertRight'))}>
            ➡ Col
          </button>
          <button
            type="button"
            disabled={!editor}
            aria-label="Delete row"
            onclick={() => withEditor(instance => editTable(instance, 'deleteRow'))}>
            🗑 Row
          </button>
          <button
            type="button"
            disabled={!editor}
            aria-label="Delete column"
            onclick={() => withEditor(instance => editTable(instance, 'deleteColumn'))}>
            🗑 Col
          </button>
        </div>
      {/if}
    </div>

    {#if showImageDialog}
      <div 
        class="rooster-dialog-overlay" 
        role="button"
        tabindex="0"
        onclick={() => showImageDialog = false}
        onkeydown={(e) => {
          if (e.key === 'Escape' || e.key === 'Enter') {
            showImageDialog = false
          }
        }}>
        <div 
          class="rooster-dialog" 
          role="dialog"
          tabindex="-1"
          onclick={(e) => e.stopPropagation()}
          onkeydown={(e) => {
            if (e.key === 'Escape') {
              showImageDialog = false
            }
          }}>
          <h3>Insert Image</h3>
          <input
            type="text"
            bind:value={imageUrl}
            placeholder="Image URL"
            class="rooster-dialog-input"
            onkeydown={(e) => {
              if (e.key === 'Enter') {
                handleInsertImage()
              }
            }} />
          <div class="rooster-dialog-actions">
            <button type="button" onclick={handleInsertImage} disabled={!imageUrl.trim()}>
              Insert
            </button>
            <button type="button" onclick={() => { showImageDialog = false; imageUrl = "" }}>
              Cancel
            </button>
          </div>
        </div>
      </div>
        {/if}

    {#if showTableDialog}
      <div 
        class="rooster-dialog-overlay" 
        role="button"
        tabindex="0"
        onclick={() => showTableDialog = false}
        onkeydown={(e) => {
          if (e.key === 'Escape' || e.key === 'Enter') {
            showTableDialog = false
          }
        }}>
        <div 
          class="rooster-dialog" 
          role="dialog"
          tabindex="-1"
          onclick={(e) => e.stopPropagation()}
          onkeydown={(e) => {
            if (e.key === 'Escape') {
              showTableDialog = false
            }
          }}>
          <h3>Insert Table</h3>
          <div class="rooster-dialog-input-group">
            <label>
              Rows:
              <input
                type="number"
                bind:value={tableRows}
                min="1"
                max="20"
                class="rooster-dialog-input" />
            </label>
            <label>
              Columns:
              <input
                type="number"
                bind:value={tableCols}
                min="1"
                max="20"
                class="rooster-dialog-input" />
            </label>
          </div>
          <div class="rooster-dialog-actions">
            <button type="button" onclick={handleInsertTable}>
              Insert
            </button>
            <button type="button" onclick={() => { showTableDialog = false; tableRows = 3; tableCols = 3 }}>
              Cancel
            </button>
          </div>
        </div>
      </div>
    {/if}

    <div class="rooster-editor-container" bind:this={editorContainer} style="position: relative;">
      <div
        bind:this={editorRoot}
        class="rooster-editor"
        role="textbox"
        aria-label="Rich text editor"
        aria-multiline="true"
        contenteditable="true"
        data-placeholder="Empieza a escribir contenido enriquecido…"></div>
    </div>
  {:else}
    <div class="rooster-placeholder">
      El editor de texto enriquecido se cargará cuando estés en el navegador.
    </div>
  {/if}
  </div>

<style>
  .rooster-wrapper {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .rooster-toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
    padding: 0.5rem;
    border: 1px solid #e2e8f0;
    border-radius: 0.75rem;
    background: #f8fafc;
  }

  .rooster-group {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding-right: 0.5rem;
    border-right: 1px solid #e2e8f0;
  }

  .rooster-group:last-child {
    border-right: none;
    padding-right: 0;
  }

  .rooster-toolbar button {
    border: 1px solid #cbd5f5;
    background: white;
    border-radius: 0.4rem;
    padding: 0.35rem 0.65rem;
    font-size: 0.85rem;
    cursor: pointer;
    color: #0f172a;
    transition: border-color 0.2s, color 0.2s, background 0.2s;
  }

  .rooster-toolbar button:hover:not(:disabled) {
    border-color: #3b82f6;
    color: #1d4ed8;
  }

  .rooster-toolbar button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .rooster-toolbar button.active {
    background: #dbeafe;
    border-color: #3b82f6;
    color: #1d4ed8;
  }

  .rooster-select {
    display: flex;
    flex-direction: column;
    font-size: 0.75rem;
    color: #475569;
  }

  .rooster-select select {
    margin-top: 0.2rem;
    border: 1px solid #cbd5f5;
    border-radius: 0.4rem;
    padding: 0.25rem 0.5rem;
    font-size: 0.85rem;
    background: white;
    color: #0f172a;
    min-width: 6rem;
  }

  .rooster-select select:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .rooster-color {
    display: flex;
    flex-direction: column;
    font-size: 0.75rem;
    color: #475569;
  }

  .rooster-color input[type="color"] {
    margin-top: 0.2rem;
    width: 2.5rem;
    height: 2rem;
    border: 1px solid #cbd5f5;
    border-radius: 0.4rem;
    padding: 0.1rem;
    background: white;
  }

  .rooster-editor-container {
    border: 1px solid #e2e8f0;
    border-radius: 0.75rem;
    background: white;
    min-height: 14rem;
    box-shadow: inset 0 1px 2px rgba(15, 23, 42, 0.05);
  }

  .rooster-editor {
    min-height: 14rem;
    padding: 1rem;
    font-size: 1rem;
    line-height: 1.6;
    color: #0f172a;
    outline: none;
  }

  .rooster-editor:empty:before {
    content: attr(data-placeholder);
    color: #94a3b8;
    pointer-events: none;
  }

  .rooster-editor :global(p) {
    margin: 0 0 0.75rem;
  }

  .rooster-editor :global(div) {
    margin-top: 0 !important;
    margin-bottom: 0 !important;
    font-family: inherit !important;
    font-size: inherit !important;
    color: inherit !important;
  }

  .rooster-editor :global(table) {
    width: 100%;
    border-collapse: collapse;
    margin: 0.75rem 0;
    position: relative;
  }

  .rooster-editor :global(table td),
  .rooster-editor :global(table th) {
    border: 1px solid #cbd5f5;
    padding: 0.5rem;
    width: auto !important;
    height: auto !important;
    border-width: 1px !important;
    border-style: solid !important;
    border-color: #cbd5f5 !important;
    vertical-align: top;
    box-sizing: border-box;
  }

  .rooster-editor :global(img) {
    max-width: 100%;
    height: auto;
    margin: 0.75rem 0;
  }

  .rooster-editor :global(hr) {
    border: none;
    border-top: 1px solid #cbd5f5;
    margin: 1rem 0;
  }

  .rooster-placeholder {
    border: 2px dashed #cbd5f5;
    border-radius: 0.75rem;
    padding: 1rem;
    text-align: center;
    color: #475569;
    background: #f8fafc;
  }

  .rooster-dialog-overlay {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .rooster-dialog {
    background: white;
    border-radius: 0.75rem;
    padding: 1.5rem;
    min-width: 300px;
    max-width: 90vw;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
  }

  .rooster-dialog h3 {
    margin: 0 0 1rem 0;
    font-size: 1.25rem;
    color: #0f172a;
  }

  .rooster-dialog-input {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #cbd5f5;
    border-radius: 0.4rem;
    font-size: 0.9rem;
    margin-bottom: 1rem;
  }

  .rooster-dialog-input-group {
    display: flex;
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .rooster-dialog-input-group label {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    font-size: 0.9rem;
    color: #475569;
  }

  .rooster-dialog-actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
  }

  .rooster-dialog-actions button {
    padding: 0.5rem 1rem;
    border: 1px solid #cbd5f5;
    border-radius: 0.4rem;
    background: white;
    color: #0f172a;
    cursor: pointer;
    font-size: 0.9rem;
    transition: all 0.2s;
  }

  .rooster-dialog-actions button:hover:not(:disabled) {
    background: #3b82f6;
    color: white;
    border-color: #3b82f6;
  }

  .rooster-dialog-actions button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
