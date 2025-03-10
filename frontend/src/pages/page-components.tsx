import { JSX } from 'solid-js';
import './components.css';

export interface IPageParams {
  type: number, name: string
  params: IPageBlock[]
  render: (e: { args: IPageSection }) => JSX.Element
}

export interface IPageSectionRenderer {
  type: number
  args: IPageSection
}

/*
  types: 1 = text, 2 = content, 3 = array of string, 4 = image, 5 = array of images
         6 = color, 7 = array of image with text
*/
export interface IPageBlock {
  id: number, name: string, type: number, 
  key?: keyof IPageSection, content?: string | string[]
}

export class PageBlock {
  constructor(id: number, name: string, type: number, key?: keyof IPageSection){
    this.id = id
    this.type = type
    this.name = name
    this.key = key
  }
  id: number
  type: number
  name: string
  key: keyof IPageSection
  content: string
  as = (name: string): PageBlock => {
    return new PageBlock(this.id, name, this.type, this.key)
  }
}

export const PageBlocks /*: {[e: string]: PageBlock} */ = {
  Title: new PageBlock(1,"Título", 1),
  Subtitle: new PageBlock(2,"Subtítulo", 1),
  Content: new PageBlock(3,"Contenido", 2),
  ContentArray: new PageBlock(4,"Contenidos", 3),
  BackgroundImage: new PageBlock(5,"Image de Fondo", 5),
  Content_1: new PageBlock(11,"Contenido 1", 2),
  Content_2: new PageBlock(12,"Contenido 2", 2),
  Content_3: new PageBlock(13,"Contenido 3", 2),
  Image: new PageBlock(15,"Imágen", 5),
  ImagesGalery: new PageBlock(16,"Imágenes (Galería)", 7),
  Categorias: new PageBlock(13,"Productos Categorías", 8),
  ColumnsCant: new PageBlock(31,"Nº Columnas", 2),
  RowsCant: new PageBlock(32,"Nº Filas", 2),
  Check_1: new PageBlock(41,"Check 1", 4),
  Check_2: new PageBlock(42,"Check 2", 4),
  Check_3: new PageBlock(43,"Check 3", 4),
  Check_4: new PageBlock(44,"Check 4", 4),
  Check_5: new PageBlock(45,"Check 5", 4),
}

for(const key in PageBlocks){ (PageBlocks as any)[key].key = key }

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
  Image?: string
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

export interface IPageRenderer {
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
