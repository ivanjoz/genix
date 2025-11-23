<script lang="ts" generics="T">
  import { browser } from "$app/environment";
  import { parseSVG } from "$lib/helpers";
  import {
    editorBackgroundColors,
    editorTextColors,
    editorTextSizes,
    TextBackgroudColor,
    TextColorIcon,
    TextSizeIcon
  } from "./HTMLEditor"
  import type {
    ContentModelDocument,
    ContentModelFormatState,
    EditorPlugin,
    IEditor,
    PluginEvent
  } from "roosterjs";
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
    TableEditPlugin,
    toggleBold,
    toggleBullet,
    toggleItalic,
    toggleNumbering,
    toggleStrikethrough,
    toggleUnderline
  } from "roosterjs";
  import { onMount } from "svelte";

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
    "8pt", "9pt", "10pt", "11pt", "12pt", "14pt",
    "16pt", "18pt", "20pt", "24pt", "28pt", "32pt"
  ])
  let fontSizeSelection = $state<string>("16pt")
  let headingSelection = $state<HeadingLevel>(0)
  let textColorValue = $state<string>("#000000")
  let backgroundColorValue = $state<string>("#ffffff")
  let fontFamilySelection = $state<string>("Arial")
  let lastSyncedValue = getInitialValue()
  let pendingFormatFrame: number | null = null
  
  // Layers state
  let showTableLayer = $state(false)
  let showTextColorLayer = $state(false)
  let showBackgroudColorLayer = $state(false)
  let showTextSizeLayer = $state(false)
  
  // Table state
  let selectedRows = $state(0)
  let selectedCols = $state(0)

  const getEditorCore = () => {
    const current = editor as (IEditor & { core?: any }) | null
    return current?.core
  }

  const ensureFontSizeOption = (size: string) => {
    if (!size) return
    if (!fontSizeOptions.includes(size)) {
      fontSizeOptions = [...fontSizeOptions, size]
    }
  }

  const createModelFromHtml = (
    html: string,
    core?: ReturnType<typeof getEditorCore>
  ): ContentModelDocument | undefined => {
    if (!browser) return undefined

    const normalized = html?.trim() ? html : "<p></p>"
    const parser = new DOMParser()
    const doc = parser.parseFromString(normalized, "text/html")
    const body = doc.body
    if (!body) return undefined

    const context = core
      ? createDomToModelContextWithConfig(
          core.environment.domToModelSettings.calculated,
          core.api.createEditorContext(core, false)
        )
      : createDomToModelContext()

    return domToContentModel(body, context)
  }

  let layerInput = $state<HTMLInputElement>()
  let avoidCloseOnBlur = false

  const sanitizeHtml = (html: string): string => {
    if (!browser) return html
    const parser = new DOMParser()
    const doc = parser.parseFromString(html, "text/html")
    
    const removeStyles = (elements: NodeListOf<HTMLElement>, props: string[]) => {
      elements.forEach(el => props.forEach(p => el.style.removeProperty(p)))
    }

    removeStyles(doc.querySelectorAll('div'), 
      ['margin-top', 'margin-bottom', 'font-family' /*, 'font-size', 'color' */])
    
    removeStyles(doc.querySelectorAll('td, th'),
      ['width', 'height', 'border-width', 'border-style', 'border-color'])

    return doc.body.innerHTML
  }

  const applyHtmlToEditor = (html: string) => {
    const core = getEditorCore()
    if (!core) return

    const sanitizedHtml = sanitizeHtml(html)
    const model = createModelFromHtml(sanitizedHtml, core)
    if (!model) return

    core.api.setContentModel(core, model, { ignoreSelection: true })
    core.api.triggerEvent(
      core,
      { eventType: "contentChanged", source: ChangeSource.SetContent },
      true
    )
    
    setTimeout(() => {
      if (editorRoot) removeUnwantedStyles(editorRoot)
    }, 0)
  }

  const removeUnwantedStyles = (element: HTMLElement) => {
    const removeStyles = (elements: NodeListOf<HTMLElement>, props: string[]) => {
      elements.forEach(el => props.forEach(p => el.style.removeProperty(p)))
    }

    removeStyles(element.querySelectorAll('div'), 
      ['margin-top', 'margin-bottom', 'font-family', 'font-size', 'color'])

    removeStyles(element.querySelectorAll('td, th'),
      ['width', 'height', 'border-width', 'border-style', 'border-color'])
  }

  const syncEditorHtml = () => {
    if (!editorRoot) return
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
    
    if (selection.type === 'table') {
      isInTable = true
      return
    }
    
    if (selection.type === 'range') {
      const range = selection.range
      const nodesToCheck: Node[] = [range.startContainer, range.commonAncestorContainer]
      
      for (const startNode of nodesToCheck) {
        let node: Node | null = startNode
        while (node && node !== editorRoot) {
          if (node.nodeType === Node.ELEMENT_NODE) {
            const tagName = (node as HTMLElement).tagName?.toUpperCase()
            if (['TABLE', 'TD', 'TH', 'TR', 'TBODY', 'THEAD', 'TFOOT'].includes(tagName)) {
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
    if (formatState.fontSize) {
      ensureFontSizeOption(formatState.fontSize)
      fontSizeSelection = formatState.fontSize
    }
    headingSelection = (formatState.headingLevel ?? 0) as HeadingLevel
    if (formatState.fontName) {
      fontFamilySelection = formatState.fontName
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

        if (editorRoot && browser) {
          mutationObserver = new MutationObserver((mutations) => {
            mutations.forEach((mutation) => {
              mutation.addedNodes.forEach((node) => {
                if (node.nodeType === Node.ELEMENT_NODE) {
                  const element = node as HTMLElement
                  removeUnwantedStyles(element)
                  
                  if (element.tagName === 'DIV') {
                    element.style.removeProperty('margin-top')
                    element.style.removeProperty('margin-bottom')
                    element.style.removeProperty('font-family')
                    //element.style.removeProperty('font-size')
                    //element.style.removeProperty('color')
                  } else if (element.tagName === 'TD' || element.tagName === 'TH') {
                    element.style.removeProperty('width')
                    element.style.removeProperty('height')
                    element.style.removeProperty('border-width')
                    element.style.removeProperty('border-style')
                    element.style.removeProperty('border-color')
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
          if (editorRoot) {
            setTimeout(() => removeUnwantedStyles(editorRoot as HTMLElement), 0)
          }
        }
        if (event.eventType === "contentChanged" || event.eventType === "selectionChanged") {
          if (event.eventType === "selectionChanged") {
            checkIfInTable()
          }
          scheduleFormatStateRefresh()
        }
      }
    }
  }

  const withEditor = (cb: (instance: IEditor) => void) => {
    if (!editor) return
    editor.focus()
    cb(editor)
    scheduleFormatStateRefresh()
  }

  const handleAlignment = (alignment: "left" | "center" | "right" | "justify") => {
    withEditor(instance => setAlignment(instance, alignment))
  }

  const handleFontSizeChange = (size: string) => {
    fontSizeSelection = size
    withEditor(instance => setFontSize(instance, size))
  }

  const handleInsertTable = () => {
    if (selectedRows > 0 && selectedCols > 0) {
      withEditor(instance => insertTable(instance, selectedCols, selectedRows))
      showTableLayer = false
      selectedRows = 0
      selectedCols = 0
    }
  }

  const handleRowSelect = (rows: number) => {
    selectedRows = rows
    if (selectedRows > 0 && selectedCols > 0) handleInsertTable()
  }

  const handleColSelect = (cols: number) => {
    selectedCols = cols
    if (selectedRows > 0 && selectedCols > 0) handleInsertTable()
  }

  const closeTableDialog = () => {
    showTableLayer = false
    selectedRows = 0
    selectedCols = 0
  }

  const handleInsertHR = () => {
    if (editor) {
      const core = getEditorCore()
      if (core) {
        core.api.insertNode(core, document.createElement('hr'))
      }
    }
  }
  
  const handleTextColorChange = (color: string) => {
    textColorValue = color
    withEditor(instance => setTextColor(instance, color))
  }

  const handleBackgroundColorChange = (color: string) => {
    backgroundColorValue = color
    withEditor(instance => setBackgroundColor(instance, color))
  }

  onMount(() => {
    if (!browser || !editorRoot) return

    const initialModel = createModelFromHtml(editorValue)
    const syncPlugin = createSyncPlugin()
    const tableEditPlugin = new TableEditPlugin()
    const instance = createEditor(editorRoot, [syncPlugin, tableEditPlugin], initialModel)
    editor = instance

    const handleClick = () => setTimeout(checkIfInTable, 0)

    editorRoot.addEventListener('click', handleClick)
    editorRoot.addEventListener('focus', handleClick)

    return () => {
      if (pendingFormatFrame !== null) {
        cancelAnimationFrame(pendingFormatFrame)
        pendingFormatFrame = null
      }
      editorRoot?.removeEventListener('click', handleClick)
      editorRoot?.removeEventListener('focus', handleClick)
      instance.dispose()
      editor = null
    }
  })

  const showLayer = $derived(showTableLayer || showTextColorLayer || showBackgroudColorLayer || showTextSizeLayer)

  $effect(() => {
    if(showLayer){ layerInput?.focus() }
  })

  $effect(() => {
    if (!saveOn || !save) return
    if (editorValue !== lastSyncedValue) {
      lastSyncedValue = editorValue
      saveOn[save] = editorValue as NonNullable<T>[keyof T]
    }
  })

  $effect(() => {
    if (!browser || !saveOn || !save) return
    const incoming = ((saveOn[save] as string) ?? "") || ""
    if (incoming !== editorValue) {
      lastSyncedValue = incoming
      editorValue = incoming
      if (editor) applyHtmlToEditor(incoming)
    }
  })

  // Button Configuration
  const toolbarItems = $derived([
    { label: 'Bold', icon: '<strong>B</strong>', action: () => withEditor(toggleBold), active: !!formatState.isBold },
    { label: 'Italic', icon: '<em>I</em>', action: () => withEditor(toggleItalic), active: !!formatState.isItalic },
    { 
      label: 'Text Size', 
      icon: `<img class="h-24 w-24" src="${parseSVG(TextSizeIcon)}" alt="" />`, 
      action: () => { showTextSizeLayer = !showTextSizeLayer }, 
      active: showTextSizeLayer,
      isLayer: true
    },
    { 
      label: 'Insert table', 
      icon: '⊞', 
      action: () => { showTableLayer = !showTableLayer }, 
      active: showTableLayer,
      isLayer: true
    },
    {
      label: 'Text color',
      html: `<img class="h-20 w-20 ml-4" src="${parseSVG(TextColorIcon)}" alt="" />
             <div class="absolute bottom-2 left-2 w-[calc(100%-4px)] h-12 border border-black/70 rounded-[2px]" style="background-color: ${textColorValue};"></div>`,
      action: () => { showTextColorLayer = !showTextColorLayer },
      active: showTextColorLayer,
      isLayer: true,
      className: 'pb-12'
    },
    {
      label: 'Background color',
      html: `<img class="h-20 w-20" src="${parseSVG(TextBackgroudColor)}" alt="" />
             <div class="absolute bottom-2 left-2 w-[calc(100%-4px)] h-12 border border-black/70 rounded-[2px]" style="background-color: ${backgroundColorValue};"></div>`,
      action: () => { showBackgroudColorLayer = !showBackgroudColorLayer },
      active: showBackgroudColorLayer,
      isLayer: true,
      className: 'pb-12'
    },
    { label: 'Align left', icon: '⬅', action: () => handleAlignment("left"), active: formatState.textAlign === "left" || !formatState.textAlign },
    { label: 'Align center', icon: '⬍', action: () => handleAlignment("center"), active: formatState.textAlign === "center" },
    { label: 'Align right', icon: '➡', action: () => handleAlignment("right"), active: formatState.textAlign === "right" },
    { label: 'Underline', icon: '<u>U</u>', action: () => withEditor(toggleUnderline), active: !!formatState.isUnderline },
    { label: 'Strikethrough', icon: '<s>S</s>', action: () => withEditor(toggleStrikethrough), active: !!formatState.isStrikeThrough },
    { label: 'Justify', icon: '☰', action: () => handleAlignment("justify"), active: formatState.textAlign === "justify" },
    { label: 'Bulleted list', icon: '••', action: () => withEditor(toggleBullet), active: !!formatState.isBullet },
    { label: 'Numbered list', icon: '1.', action: () => withEditor(toggleNumbering), active: !!formatState.isNumbering },
    { label: 'Insert horizontal rule', icon: '─', action: handleInsertHR },
    { label: 'Clear format', icon: 'Clear', action: () => withEditor(clearFormat) },
    { 
      label: 'Undo', 
      icon: '↶', 
      action: () => { if (editor && getEditorCore()) { getEditorCore()?.api.undo(getEditorCore()!); scheduleFormatStateRefresh() } } 
    },
    { 
      label: 'Redo', 
      icon: '↷', 
      action: () => { if (editor && getEditorCore()) { getEditorCore()?.api.redo(getEditorCore()!); scheduleFormatStateRefresh() } } 
    },
  ])

  const tableButtons = [
    { label: 'Add row above', icon: '⬆ Row', action: () => withEditor(instance => editTable(instance, 'insertAbove')) },
    { label: 'Add row below', icon: '⬇ Row', action: () => withEditor(instance => editTable(instance, 'insertBelow')) },
    { label: 'Add column left', icon: '⬅ Col', action: () => withEditor(instance => editTable(instance, 'insertLeft')) },
    { label: 'Add column right', icon: '➡ Col', action: () => withEditor(instance => editTable(instance, 'insertRight')) },
    { label: 'Delete row', icon: '🗑 Row', action: () => withEditor(instance => editTable(instance, 'deleteRow')) },
    { label: 'Delete column', icon: '🗑 Col', action: () => withEditor(instance => editTable(instance, 'deleteColumn')) },
  ]

  const buttonCss = "mr-4 mb-4 w-32 h-28 flex items-center justify-center rounded bg-indigo-100/50 cursor-pointer text-[13px] text-slate-900 transition-all hover:not(.selected):bg-indigo-100 hover:not(.selected):border-indigo-500 hover:not(.selected):text-indigo-700"
</script>

<div class={`flex flex-col ${css ?? ""}`}>
  {#if browser}
    <div class="flex flex-wrap gap-2 items-center p-6 border border-slate-200 rounded-t-[6px] bg-slate-50 relative"
      role="toolbar"
      aria-label="Editor toolbar"
      tabindex="-1"
      onclick={(e) => {
        if (showTableLayer && !(e.target as HTMLElement).closest('._10')) {
          // closeTableDialog()
        }
      }}
      onkeydown={(e) => {
        if (e.key === 'Escape' && showTableLayer) {
          // closeTableDialog()
        }
      }}
    >
      {#each toolbarItems as item}
        <button type="button"  disabled={!editor}
          class:_6={item.isLayer && item.active}
          class="_4 {item.active ? 'bg-blue-100 border-blue-500 text-blue-700' : ''} {item.className ?? ''}"
          aria-label={item.label}
          onclick={item.action}
        >
          {@html item.html || item.icon}
        </button>
      {/each}

      {#if showLayer}
        <div class="absolute top-[47px] w-[calc(100%-20px)] left-[10px] border border-[#ab9efc] min-h-[60px] bg-white rounded-lg z-50 shadow-lg p-12 _10" 
          role="dialog"
          aria-label="Editor popup"
          tabindex="-1"
          onmousedown={() => avoidCloseOnBlur = true}
          onclick={(e) => e.stopPropagation()}
          onkeydown={(e) => { if (e.key === 'Escape') closeTableDialog() }}
        >
          <input type="text" bind:this={layerInput}
            inputmode="none" autocomplete="off" 
            class="opacity-0 h-2 w-1 absolute z-[-1]"
            onblur={(ev => {
              // return
              if(avoidCloseOnBlur){
                avoidCloseOnBlur = false;
                (ev.target as HTMLInputElement).focus()
                return
              }
              showTableLayer = false
              showTextColorLayer = false
              showBackgroudColorLayer = false
              showTextSizeLayer = false
            })}
          >
          {#if showTableLayer}
            <div>
              <div class="flex items-start flex-col md:flex-row md:items-center">
                <div class="w-80 text-sm font-semibold text-slate-600 mb-8">Columnas</div>
                <div class="flex flex-wrap items-center">
                  {#each Array(12) as _, i} 
                    {@const num = i + 1}
                    <button type="button"
                      class="{buttonCss} {selectedCols >= num ? 'bg-indigo-500 text-white font-semibold' : ''}"
                      class:selected={selectedCols >= num}
                      onclick={() => handleColSelect(num)}
                      aria-label="Select {num} columns"
                    >
                      {num}
                    </button>
                  {/each}
                </div>
              </div>
              <div class="flex items-start flex-col md:flex-row md:items-center">
                <div class="w-80 text-sm font-semibold text-slate-600 mb-8">Filas</div>
                <div class="flex flex-wrap items-center">
                  {#each Array(12) as _, i}
                    {@const num = i + 1}
                    <button type="button"
                      class="{buttonCss} {selectedRows >= num ? 'bg-indigo-500 text-white font-semibold' : ''}"
                      class:selected={selectedRows >= num}
                      onclick={() => handleRowSelect(num)}
                      aria-label="Select {num} rows"
                    >
                      {num}
                    </button>
                  {/each}
                </div>
              </div>
            </div>
          {/if}
          {#if showBackgroudColorLayer || showTextColorLayer}
            {@const colors = showBackgroudColorLayer ? editorBackgroundColors : editorTextColors }
            <div class="flex flex-wrap w-full">
              {#each colors as color }
                <!-- svelte-ignore a11y_click_events_have_key_events -->
                <!-- svelte-ignore a11y_no_static_element_interactions -->
                <div class="h-20 w-32 m-4 border border-black/60 cursor-pointer hover:outline hover:outline-1 hover:outline-black/70" 
                  style="background-color: {color};"
                  onclick={() => {
                    if(showBackgroudColorLayer) handleBackgroundColorChange(color)
                    else handleTextColorChange(color)
                  }}
                ></div>
              {/each}
            </div>
          {/if}
          {#if showTextSizeLayer}
            <div class="flex flex-wrap w-full gap-8">
              {#each editorTextSizes as e }
                <button type="button" 
                  class="px-12 py-4 hover:bg-slate-100 rounded border border-slate-200"
                  onclick={() => { 
                    handleFontSizeChange(`${e.id}px`); showTextSizeLayer = false; 
                  }}
                >
                  {e.name}
                </button>
              {/each}
            </div>
          {/if}
        </div>
      {/if}

      {#if isInTable}
        {#each tableButtons as item}
          <button
            type="button"
            disabled={!editor}
            class="_4"
            aria-label={item.label}
            onclick={item.action}
          >
            {item.icon}
          </button>
        {/each}
      {/if}
    </div>

    <div class="border border-slate-200 rounded-b-[6px] bg-white min-h-[14rem] shadow-inner" bind:this={editorContainer} style="position: relative;">
      <div
        bind:this={editorRoot}
        class="rooster-editor min-h-[14rem] p-16 text-base leading-relaxed text-slate-900 outline-none empty:before:content-[attr(data-placeholder)] empty:before:text-slate-400 empty:before:pointer-events-none"
        role="textbox"
        aria-label="Rich text editor"
        aria-multiline="true"
        contenteditable="true"
        data-placeholder="Empieza a escribir contenido enriquecido…"></div>
    </div>
  {:else}
    <div class="border-2 border-dashed border-slate-300 rounded-xl p-16 text-center text-slate-600 bg-slate-50">
      El editor de texto enriquecido se cargará cuando estés en el navegador.
    </div>
  {/if}
</div>

<style>
  ._4 {
    width: 42px;
    height: 38px;
    border: 1px solid #cbd5e1;
    background-color: white;
    border-radius: 4px;
    font-size: 14px;
    cursor: pointer;
    color: #0f172a;
    transition-property: color, background-color, border-color;
    transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
    transition-duration: 150ms;
    display: flex;
    align-items: center;
    justify-content: center;
    position: relative;
  }
  
  ._4:hover:not(:disabled) {
    border-color: #3b82f6;
    color: #1d4ed8;
  }
  
  ._4:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  ._6 {
    margin-bottom: -8px;
    border: 2px solid #996dff;
    background-color: #faf9ff;
    border-bottom: none;
    border-radius: 4px 4px 0 0;
    z-index: 80;
  }

  /* Global editor styles that are hard to tailwind-ize inside the contenteditable div */
  div :global(.rooster-editor p) {
    margin: 0 0 6px;
  }

  div :global(.rooster-editor table) {
    width: 100%;
    border-collapse: collapse;
    margin: 6px 0;
    position: relative;
  }

  div :global(.rooster-editor table td),
  div :global(.rooster-editor table th) {
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

  div :global(.rooster-editor img) {
    max-width: 100%;
    height: auto;
    margin: 0.75rem 0;
  }

  div :global(.rooster-editor hr) {
    border: none;
    border-top: 1px solid #cbd5f5;
    margin: 1rem 0;
  }
</style>