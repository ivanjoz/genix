import { For } from "solid-js"
import { pageExample } from "./page-example"
import { Header1 } from "./headers"
import './components.css'
import { BigScroll, Demo1, LayerImage21 } from "./components"

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
}

export const PageRenderer = (props: IPageRenderer) => {

  return <For each={props.sections}>
    {e => {
      if(e.type === 10){ return <Header1 args={e} /> }
      else if(e.type === 21){ return <LayerImage21 args={e} /> }
      else if(e.type === 9998){ return <Demo1 args={e} /> }
      else if(e.type === 9999){ return <BigScroll args={e} /> }
      else {
        return <div></div>
      }
    }}
  </For>
}

export default function PageBuilder() {

  return <PageRenderer sections={pageExample} />

}