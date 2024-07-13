import { BasicSection, BigScroll, Demo1, LayerImage21 } from "./components";
import './components.css';
import { Header1 } from "./headers";
import { IPageParams } from "./page";

export const coponentsRenders: IPageParams[] = [
  { type: 1, name: 'Basic Section', 
    render: e => <BasicSection args={e.args} />,
    params: [1,2,3],
  },
  { type: 10, name: 'Header 1', 
    render: e => <Header1 args={e.args} />, 
    params: [1,2,3],
  },
  { type: 21, name: 'Layer Image 21', 
    render: e => <LayerImage21 args={e.args} />,
    params: [1,2,3],
  },
  { type: 9998, name: 'Demo 1', render: e => <Demo1 args={e.args} />, 
    params: []
  },
  { type: 9999, name: 'Big Scroll', render: e => <BigScroll args={e.args} />, 
    params: []
  }
]
