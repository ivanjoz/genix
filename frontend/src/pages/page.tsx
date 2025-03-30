import { For, Show } from "solid-js";
import { BasicSection, BigScroll, Demo1, LayerImage21 } from "./components";
import './components.css';
import { Header1 } from "./headers";
import { IPageParams, IPageRenderer, IPageSectionRenderer, PageBlocks } from "./page-components";
import styles from './page.module.css';
import { CartFloating, ProductoInfoLayer, ProductosCuadrilla, productoSelected } from "./productos";

const pb = PageBlocks

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
    <ProductoInfoLayer />
  </>
}
