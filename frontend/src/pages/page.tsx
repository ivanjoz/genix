import { For } from "solid-js"
import { pageExample } from "./page-example"
import { Header1 } from "./headers"
import './components.css'
import { BasicSection, BigScroll, Demo1, LayerImage21 } from "./components"
import styles from './page.module.css'

export interface IPageSection {
  type?: number
  mode?: number
  style?: string
  content?: string
  contentArray?: string[]
  backgroundImage?: string
  title?: string
  subtitle?: string
  images?: string[]
  image1?: string
  image2?: string
  params?: number[]
  color1?: string
  color2?: string
  color3?: string
  marginTop?: string
}

interface IPageRenderer {
  sections: IPageSection[]
  isEditable?: boolean
}

export interface ISectionParams {
  key: keyof IPageSection
  name: string
  type: number
  content?: string | number | string[] | number[]
}

export interface IPageParams {
  id: number, name: string
  params: ISectionParams[]
}

interface IPageSectionRenderer {
  type: number
  args: IPageSection
}

export const PageSectionRenderer = (e: IPageSectionRenderer) => {
  if(e.type === 1){ return <BasicSection args={e.args} /> }
  else if(e.type === 10){ return <Header1 args={e.args} /> }
  else if(e.type === 21){ return <LayerImage21 args={e.args} /> }
  else if(e.type === 9998){ return <Demo1 args={e.args} /> }
  else if(e.type === 9999){ return <BigScroll args={e.args} /> }
  else {
    return <div></div>
  }
}

export const PageRenderer = (props: IPageRenderer) => {

  return <For each={props.sections}>
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
}

export function PageViewer() {

  return <PageRenderer sections={pageExample} />

}

export default function PageBuilder() {

  return <PageRenderer isEditable={true} sections={pageExample} />

}