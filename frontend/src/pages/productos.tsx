import { createEffect } from "solid-js";
import { IHeader1 } from "./headers";
import { IPageSection } from "./page";
import { useProductosCmsAPI } from "./productos-service";
import { ImageCtn, ImageUploader } from "~/components/Uploaders";
import s1 from "./components.module.css"
import { formatN } from "~/shared/main";

export function ProductosCuadrilla(props: IHeader1) { // type: 10

  const [ productos ] = useProductosCmsAPI()

  createEffect(() => {
    console.log("productos obtenidos component::", productos())
  })

  return <div class={"w100 " + s1.product_cuadrilla_ctn}>
    <div class="w100 flex-wrap jc-center">
      { productos().productos.map(e => {
          return <div class={`p-rel flex-column ai-center ${s1.product_card}`}>
            { e.Images?.length > 0 &&
              <ImageCtn src={"img-productos/"+ e.Images[0].n} size={2}
                class={s1.product_card_image}
                types={["avif","webp"]}
              />
            }
            <div class={`flex jc-start w100 mt-08 ${s1.product_name_ctn}`}>
              <div class="grow-1">
                <div>{e.Nombre}</div>
                <div class="ff-bold h3">s/. {formatN(e.PrecioFinal,2)}</div>
              </div>
              <div class={`flex ai-center jc-center ${s1.product_cart_button}`}>
                <i class="icon-basket"></i>
                <div class={s1.product_cart_bn_text}>
                  Agregar <br /> al Carrito
                </div>
              </div>
            </div>
          </div>
        })
      }
    </div>
    <h2>Productos aqui = {productos().productos.length}</h2>
    <h2>hola mundo</h2>
  </div>
}
