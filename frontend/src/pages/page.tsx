import { For, JSX } from "solid-js"
import './components.css'
import { coponentsRenders } from "./page-components"
import { pageExample } from "./page-example"
import styles from './page.module.css'

export interface IPageSection {
  id?: number
  type?: number
  mode?: number
  style?: string
  content?: string
  contentArray?: string[]
  backgroundImage?: string
  content1?: string
  content2?: string
  content3?: string
  title?: string
  subtitle?: string
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
}

/*
  types: 1 = text, 2 = content, 3 = array of string, 4 = image, 5 = array of images
         6 = color, 7 = array of image with text
*/
export const PageSectionsDefs: ISectionParams[] = [
  { id: 1,  key: 'title', name: 'Título', type: 1 },
  { id: 2,  key: 'subtitle', name: 'Subtítulo', type: 1 },
  { id: 3,  key: 'content', name: 'Contenido', type: 2 },
  { id: 4,  key: 'contentArray', name: 'Contenidos', type: 3 },
  { id: 5,  key: 'backgroundImage', name: 'Image de Fondo', type: 4 },
  { id: 11, key: 'content1', name: 'Contenido 1', type: 2 },
  { id: 12, key: 'content2', name: 'Contenido 2', type: 2 },
  { id: 13, key: 'content3', name: 'Contenido 3', type: 2 },
  { id: 13, key: 'images', name: 'Imágenes', type: 5 },
  { id: 14, key: 'imagesGalery', name: 'Imágenes (Galería)', type: 7 },
]

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
  params: (number|[number,string])[]
  render: (e: any) => JSX.Element
}

interface IPageSectionRenderer {
  type: number
  args: IPageSection
}

export const PageSectionRenderer = (e: IPageSectionRenderer) => {
  for(const sec of coponentsRenders){
    if(e.type === sec.type){ return sec.render(e) }
  }
  return <div></div>
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