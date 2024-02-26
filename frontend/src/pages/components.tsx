import { IHeader1 } from "./headers";
import { IPageSection } from "./page";

export function BigScroll(props: IHeader1) { // type: 10

  return <div class="w100" style={{ height: '160vh' }}>

  </div>
}

export function Demo1(props: IHeader1) { // type: 10

  return <div class="w100" style={{ height: '80px' }}>
    <h2>title ---------</h2>
    <h2>hola mundo</h2>
  </div>
}

export function LayerImage21(props: IHeader1) { // type: 21

  return <div class="layer-im1 w100" style={{ "margin-top": props.args.marginTop }}>
    
  </div>
}

export function BasicSection(props: { args: IPageSection }) { // type: 21

  return <div class="w100" style={{ "height": "12rem" }}>
    <div><h1>{props.args.title}</h1></div>
    <div>{props.args.content}</div>
  </div>
}
