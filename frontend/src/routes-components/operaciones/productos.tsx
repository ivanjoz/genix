import FroalaEditor from 'froala-editor'
import 'froala-editor/js/plugins/colors.min.js'
import 'froala-editor/js/plugins/font_size.min.js'
import 'froala-editor/js/plugins/table.min.js'
import 'froala-editor/js/plugins/line_height.min.js'
import 'froala-editor/js/plugins/paragraph_style.min.js'
import 'froala-editor/js/plugins/align.min.js'
import 'froala-editor/css/plugins/table.min.css'
import { onMount, onCleanup, on, createEffect } from 'solid-js'
import s1 from '../operaciones/operaciones.module.css'
import './froala.css'
import { IProducto } from '~/services/operaciones/productos'

interface IProductoFichaEditor {
  producto: IProducto
}

export const ProductoFichaEditor = (props: IProductoFichaEditor) => {

  let refDiv: HTMLDivElement = null
  let htmlEditor: FroalaEditor = null

  onMount(() => {
    console.log("refdiv::",refDiv)
    htmlEditor = new FroalaEditor(refDiv, {
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
          props.producto.ContentHTML = htmlEditor.html.get()
          console.log("cambió el producto::", props.producto)
        },
        initialized: () => {
          htmlEditor.html.set(props.producto.ContentHTML||"")
        }
      }      
    })
  })

  createEffect(on(
    () => [props.producto],
    () => {
      if(props.producto && htmlEditor?.html){
        htmlEditor.html.set(props.producto.ContentHTML||"")
      }
    }
  ))

  return <div class={`mt-08 w100 ${s1.product_ficha_editor}`} ref={refDiv}>

  </div>

}