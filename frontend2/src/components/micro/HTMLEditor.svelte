<script lang="ts" generics="T">
  import { onDestroy, onMount, untrack } from "svelte"
  import { browser } from "$app/environment"
  import type FroalaEditor from "froala-editor"
  import "froala-editor/css/plugins/table.min.css"
  import "./froala.css"

  const { saveOn, save, css }: {
    saveOn: T
    save: keyof T
    css: string
  } = $props()

  let divElement: HTMLDivElement
  let htmlEditor: FroalaEditor | undefined

  let prevSaveOn: T | undefined
  $effect(() => {
    // console.log("Nuevo SaveOn:", $state.snapshot(saveOn))
    if(saveOn && save && saveOn !== prevSaveOn && htmlEditor){
      prevSaveOn = saveOn
      const editor = htmlEditor
      untrack(() => {
        editor.html.set((saveOn[save] ||"") as string)
      })
    }
  })

  onMount(async () => {
    if (!browser || !divElement) {
      return
    }

    const FroalaModule = (await import("froala-editor")) as {
      default: typeof import("froala-editor").default
    }

    await Promise.all([
      import("froala-editor/js/plugins/colors.min.js"),
      import("froala-editor/js/plugins/font_size.min.js"),
      import("froala-editor/js/plugins/table.min.js"),
      import("froala-editor/js/plugins/line_height.min.js"),
      import("froala-editor/js/plugins/paragraph_style.min.js"),
      import("froala-editor/js/plugins/align.min.js")
    ])

    const Froala = FroalaModule.default

    const editor = new Froala(divElement, {
      toolbarButtons: [
        'bold', 'italic', 'fontSize', 'textColor', 'backgroundColor',
        'align',
        'outdent','indent', 'underline', 'insertHR', 'lineHeight', 
        'formatOLSimple','insertTable', 'html'
      ],
      fontSize: ['14','16','18','20','24','28'],
      fontSizeSelection: true,
      fontFamilySelection: true,
      colorsBackground: [
        '#15E67F', '#E3DE8C', '#D8A076', '#D83762', '#76B6D8', 'REMOVE',
        '#1C7A90', '#249CB8', '#4ABED9', '#FBD75B', '#FBE571', '#FFFFFF'
      ],
      colorsStep: 6,
      colorsText: [
        '#15E67F', '#E3DE8C', '#D8A076', '#D83762', '#76B6D8', 'REMOVE',
        '#1C7A90', '#249CB8', '#4ABED9', '#FBD75B', '#FBE571', '#FFFFFF'
      ],
      events: {
        contentChanged: () => {
          if(saveOn && save){
            saveOn[save] = editor.html.get() as NonNullable<T>[keyof T]
          }
          console.log("Se asignó el HTML::", $state.snapshot(saveOn))
        },
        initialized: () => {
          if(saveOn && save){
            editor.html.set((saveOn[save] ||"") as string)
          }
        }
      }
    })
    htmlEditor = editor
  })

  onDestroy(() => {
    htmlEditor?.destroy()
    htmlEditor = undefined
  })

</script>

<div bind:this={divElement} class="{css}">

</div>