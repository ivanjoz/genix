import { BackgroundParallax } from "~/components/Layers";
import { IHeader1 } from "./headers";
import { IPageSection } from "./page-components";
import { Image } from "../components/Uploaders"

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
//  <div style={{ height: "150vh", width: "100%", "background-image": "linear-gradient(0deg, #FA8BFF 0%, #2BD2FF 52%, #2BFF88 90%)" }}></div>
export function LayerImage21(props: IHeader1) { // type: 21

  return <div class="layer-im1 w100" style={{ "margin-top": props.args.marginTop }}>
    <BackgroundParallax offsetTop={0} keyUpdated={props.args.Image||"_"}>
      { props.args.Image ?
        <Image size={8} src={`img-galeria/${props.args.Image}`} class="w100" 
          style= {{ height: 'auto' }} types={["avif","webp"]}
        />
        : <div class="w100 h100"></div>
      }
    </BackgroundParallax>
  </div>
}

export function BasicSection(props: { args: IPageSection }) { // type: 21

  return <div class="w100" style={{ "height": "12rem" }}>
    <div><h1>{props.args.Title}</h1></div>
    <div>{props.args.Content}</div>
  </div>
}
