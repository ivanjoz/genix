import { For, JSX, Show } from "solid-js"
import './components.css'
import { componentsRenders, IPageBlock } from "./page-components"
import styles from './page.module.css'
import { CartFloating, ProductoInfoLayer, productoSelected } from "./productos"

export interface IPageSection {
  id?: number
  type?: number
  mode?: number
  style?: string
  Content?: string
  contentArray?: string[]
  backgroundImage?: string
  content1?: string
  content2?: string
  content3?: string
  Title?: string
  Subtitle?: string
  images?: string[]
  imagesGalery?: string[]
  image1?: string
  image2?: string
  params?: number[]
  color1?: string
  color2?: string
  color3?: string
  marginTop?: string
  marginBottom?: string
  columnasCant?: number
  filasCant?: number
}


interface IPageRenderer {
  sections: IPageSection[]
  isEditable?: boolean
}

export interface ISectionParams {
  id: number
  key: keyof IPageSection
  name: string
  type: number
  content?: string | number | string[] | number[]
}

export interface IPageParams {
  type: number, name: string
  params: IPageBlock[]
  render: (e: { args: IPageSection }) => JSX.Element
}

interface IPageSectionRenderer {
  type: number
  args: IPageSection
}

export const PageSectionRenderer = (e: IPageSectionRenderer) => {
  for(const sec of componentsRenders){
    if(e.type === sec.type){ return sec.render(e) }
  }
  return <div></div>
}

export default function PageRenderer(props: IPageRenderer){
  
  return <>
    <For each={props.sections}>
      {e => {
        if(props.isEditable){
          return <div class={`p-rel w100 ${styles.cms_editable_card}`}>
            <div class={`flex ai-center p-abs ff-bold ${styles.cms_card_button_up}`}>
              <i class="icon-plus h6"></i>Sección
            </div>
            <PageSectionRenderer args={e} type={e.type} />
            <div class={`flex ai-center p-abs ff-bold ${styles.cms_card_button_down}`}>
              <i class="icon-plus h6"></i>Sección
            </div>
          </div>
        }
        return <PageSectionRenderer args={e} type={e.type} />
      }}
    </For>
    <CartFloating />
    <Show when={!!productoSelected()}>
      <ProductoInfoLayer />
    </Show>
  </>
}
