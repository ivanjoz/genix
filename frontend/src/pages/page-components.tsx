import { BasicSection, BigScroll, Demo1, LayerImage21 } from "./components";
import './components.css';
import { Header1 } from "./headers";
import { IPageParams } from "./page";
import { ProductosCuadrilla } from "./productos";

/*
  types: 1 = text, 2 = content, 3 = array of string, 4 = image, 5 = array of images
         6 = color, 7 = array of image with text
*/
export interface IPageBlock {
  id: number, name: string, type: number, key?: string, content?: string | string[]
}

export const PageBlocks: {[e: string]: IPageBlock} = {
  Title: { id: 1, name: 'Título', type: 1 },
  Subtitle: { id: 2, name: 'Subtítulo', type: 1 },
  Content: { id: 3, name: 'Contenido', type: 2 },
  ContentArray: { id: 4, name: 'Contenidos', type: 3 },
  BackgroundImage: { id: 5, name: 'Image de Fondo', type: 4 },
  Content_1: { id: 11, name: 'Contenido 1', type: 2 },
  Content_2: { id: 12, name: 'Contenido 2', type: 2 },
  Content_3: { id: 13, name: 'Contenido 3', type: 2 },
  Images: { id: 14, name: 'Imágenes', type: 5 },
  ImagesGalery: { id: 14, name: 'Imágenes (Galería)', type: 7 },
  Categorias: { id: 13, name: 'Productos Categorías', type: 8 },
  ColumnsCant: { id: 31, name: 'Nº Columnas', type: 2 },
  RowsCant: { id: 32, name: 'Nº Filas', type: 2 },
}

const pb = PageBlocks
for(const key in PageBlocks){ (PageBlocks as any)[key].key = key }

export const coponentsRenders: IPageParams[] = [
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
    params: [pb.Title, pb.Subtitle, pb.Content],
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
