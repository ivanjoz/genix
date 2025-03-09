import { BasicSection, BigScroll, Demo1, LayerImage21 } from "./components";
import './components.css';
import { Header1 } from "./headers";
import { IPageParams } from "./page";
import { pageExample } from "./page-example";
import { ProductosCuadrilla } from "./productos";

/*
  types: 1 = text, 2 = content, 3 = array of string, 4 = image, 5 = array of images
         6 = color, 7 = array of image with text
*/
export interface IPageBlock {
  id: number, name: string, type: number, key?: string, content?: string | string[]
}

export class PageBlock {
  constructor(id: number, name: string, type: number, key?: string){
    this.id = id
    this.type = type
    this.name = name
    this.key = key
  }
  id: number
  type: number
  name: string
  key: string
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

const pb = PageBlocks
for(const key in PageBlocks){ (PageBlocks as any)[key].key = key }

export const componentsRenders: IPageParams[] = [
  { type: 1, name: 'Basic Section', 
    render: e => <BasicSection args={e.args} />,
    params: [pb.Title, pb.Subtitle, pb.Content],
  },
  { type: 10, name: 'Header 1', 
    render: e => <Header1 args={e.args} />, 
    params: [pb.Title, pb.Subtitle, pb.Content],
  },
  { type: 21, name: 'Layer Image 21', 
    render: e => <LayerImage21 args={e.args} />,
    params: [pb.Title, pb.Subtitle, pb.Content, pb.Image],
  },
  { type: 41, name: 'Productos Cuadrilla', 
    render: e => <ProductosCuadrilla args={e.args} />,
    params: [pb.Content],
  },
  { type: 9998, name: 'Demo 1', render: e => <Demo1 args={e.args} />, 
    params: []
  },
  { type: 9999, name: 'Big Scroll', render: e => <BigScroll args={e.args} />, 
    params: []
  }
]
