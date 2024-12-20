import { createEffect } from "solid-js";
import { IHeader1 } from "./headers";
import { IPageSection } from "./page";
import { useProductosCmsAPI } from "./productos-service";
import { ImageUploader } from "~/components/Uploaders";

export function ProductosCuadrilla(props: IHeader1) { // type: 10

  const [ productos ] = useProductosCmsAPI()

  createEffect(() => {
    console.log("productos obtenidos component::", productos())
  })

  return <div class="w100" style={{ "min-height": '80px' }}>
    <div class="grid w100" style={{ "grid-template-columns": "1fr 1fr 1fr 1fr 1fr" }}>
      { productos().productos.map(e => {
          return <div>
            {e.Nombre}
            { e.Images?.length > 0 &&
              <ImageUploader src={"img-productos/"+ e.Images[0].n + "-x2"}
                cardStyle={{ width: '100%' }}
                types={["avif","webp"]}
              />
            }
          </div>
        })
      }
    </div>
    <h2>Productos aqui = {productos().productos.length}</h2>
    <h2>hola mundo</h2>
  </div>
}
