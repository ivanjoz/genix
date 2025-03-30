import { createEffect, createMemo, createSignal, For, JSX, Show } from "solid-js"
import { CardArrowSteps } from "~/components/Cards"
import { CartFloatingProducto, cartProductos } from "./productos"
import { formatN, getCmsTable, makeEcommerceDB } from "~/shared/main"
import s1 from "./components.module.css"
import { Input } from "~/components/Input"
import { CiudadSelector } from "~/core/components"
import { throttle } from "~/core/main"
import { isMobile } from "~/app"
import { Env } from "~/env"

export const [cartOption, setCartOption] = createSignal(1)

export interface IEcommerceCart {

}

export interface ICartForm {
  nombres: string
  apellidos: string
  email: string
  direccion?: string
  referencia?: string
  ciudadID?: string
}

export const [cartForm, setCartForm] = createSignal({} as ICartForm)

const savedForm = {} as ICartForm

export const saveCartForm = () => {
  throttle(async () => {
    const form = cartForm()
    const valuesToSave = []
  
    for(const key of Object.keys(form) as (keyof ICartForm)[]){
      const currentValue = savedForm[key]||""
      const value = form[key] || ""
      if(currentValue !== value){
        savedForm[key] = value
        valuesToSave.push({ formID: 1, key, value })
      }
    }
  
    if(valuesToSave.length > 0){
      const dbTable = await getCmsTable("forms")
      dbTable.bulkPut(valuesToSave)
    }
  },500)
}

export const EcommerceCart = (props: IEcommerceCart) => {

  const precio = createMemo(() => {
    let precio = 0
    for(const pr of cartProductos().values()){
      precio += (pr.cant * pr.producto.PrecioFinal)
    }
    return precio
  })

  getCmsTable("forms").then(table => {
    table.where({ formID: 1 }).toArray().then(formRecords => {
      const form = {} as ICartForm
      for(const e of formRecords){
        form[e.key as keyof ICartForm] = e.value
      }
      setCartForm(form)
    })
  })

  const paymentMethods = {
    tarjeta: true,
    yape: true,
    billetera: true,
    bancaMovil: true,
  }

  const publicKey = Env.empresa.CulqiLlave || ""

  const config = {
    publicKey,
    settings: {
      title: 'Culqi  store 2',
      currency: 'PEN',
      amount: 8000,
      order: 'ord_live_d1P0Tu1n7Od4nZdp',
      xculqirsaid: 'Inserta aquí el id de tu llave pública RSA',
      rsapublickey: 'Inserta aquí tu llave pública RSA',
    },
    options: {
      lang: 'auto',
      installments: true,
      modal: false,
      container: "#culqi-container", // Opcional
      paymentMethods: paymentMethods,
      paymentMethodsSort: Object.keys(paymentMethods), // las opciones se ordenan según se configuren en paymentMethods
    },
    appearance: {
      hiddenBanner: true,
      menuType: isMobile() ? "sliderTop" : "", // "sliderTop",
      rules: {
        ".Culqi-Input": { "margin-bottom": "8px" },
      }
    },
    client: {
      email: 'test2@demo.com',
    }
  }

  createEffect(() => {
    if(cartOption() === 3){
      setTimeout(() => {
        const handleCulqiAction = () => {
          if (Culqi.token) {
            const token = Culqi.token.id;
            console.log('Se ha creado un Token: ', token);
          } else if (Culqi.order) {
            const order = Culqi.order;
            console.log('Se ha creado el objeto Order: ', order);
          } else {
            console.log('Errorrr : ', Culqi.error);
          }
        }
        
        const Culqi = new window.CulqiCheckout(publicKey, config)
        Culqi.culqi = handleCulqiAction
        Culqi.open()
      },100)
    }
  })

  const ciudadCss = "sm:w-full w-1/2 mb-10 ps-form"
  //id="culqi-js"

  return <div class={`w100 flex flex-col ac-baseline h100 p-rek ${s1.menu_cart_layer_container}`}>
    <CardArrowSteps selected={cartOption()}
      columnsTemplate={isMobile() ? "1fr 1fr 1fr 0.7fr" : ""}
      options={[ 
        { id: 1, name: 'Carrito', icon: "icon-basket" }, 
        { id: 2, name: 'Datos de Envío', icon: "icon-doc-inv-alt" }, 
        { id: 3, name: 'Pago', icon: "icon-shield" }, 
        { id: 4, 
          name: 'Confirmación', 
          icon: "icon-ok" 
        },
      ]}
      onSelect={opt => {
        setCartOption(opt.id)
      }}
      optionRender={e => {
        let name = e.name as (string|JSX.Element);
        if(e.id === 2){ name = <span>Datos de <br />Envío</span>  }
        else if(e.id === 4){
          if(isMobile()){ name = <i class="icon-ok"></i>  }
          else {
            name = <span>Confirmación</span> 
          } 
        }
        return <div class={`flex ai-center mt-01 ff-semibold ${s1.menu_cart_layer_header_button}`}>
          <i class={`h3 ${e.icon} mr-02`} style={{ "margin-left": "-6px" }}></i>
          <div class={s1.menu_cart_layer_header_button_name}>{name}</div>
        </div>
      }}
    />
    <div class="w100 mt-08"></div>
    <Show when={cartOption() === 1}>
      <div class="h100 p-rel flex flex-col overflow-s4">
        <CarritoSubtotal precio={precio()} accion="Continuar" 
          onAccion={() => setCartOption(2) }
        />
        <div class={`h100 w100 grid ac-baseline ${s1.menu_cart_layer_products}`}>
          <For each={Array.from(cartProductos().values())}>
          {e => {
            console.log("rendering productos::", cartProductos())
            return <CartFloatingProducto producto={e.producto} cant={e.cant} mode={2} />
          }}
          </For>
        </div>
      </div>
    </Show>
    <Show when={cartOption() === 2}>
      <div class="h100 p-rel flex flex-col overflow-s4">
        <CarritoSubtotal precio={precio()} accion="Pagar" 
          onAccion={() => setCartOption(3) }
        />
        <div class="flex-wrap w100-10">
          <Input label="Nombres" saveOn={cartForm()} save="nombres"  
            css="w-1/2 mb-10 ps-form" required={true}
            onChange={() => saveCartForm()}
          />
          <Input label="Apellidos" saveOn={cartForm()} save="apellidos"  
            css="w-1/2 mb-10 ps-form" required={true}
            onChange={() => saveCartForm()}
          />
          <Input label="Correo Electrónico" saveOn={cartForm()} save="email"  
            css="sm:w-1/1 w-1/2 mb-10 ps-form" required={true}
            onChange={() => saveCartForm()}
          />
          <CiudadSelector css={[ciudadCss, ciudadCss, ciudadCss]}
            saveOn={cartForm()} save="ciudadID" 
            onChange={() => saveCartForm()}
          />
          <Input label="Dirección" saveOn={cartForm()} save="direccion"  
            css="w-full ps-form mb-10" required={true}
            onChange={() => saveCartForm()}
          />
          <Input label="Referencia" saveOn={cartForm()} save="referencia"  
            css="w-full ps-form mb-10"
            onChange={() => saveCartForm()}
          />
        </div>
      </div>
    </Show>
    <Show when={cartOption() === 3}>
      <div class="h100 w100 p-rel" id="culqi-container" style={{ position: 'relative', height: "40rem" }}>

      </div>
    </Show>
  </div>
}

export interface ICarritoSubtotal {
  precio: number
  accion: string
  onAccion: () => void
}

export const CarritoSubtotal = (props: ICarritoSubtotal) => {

  return <div class="w100 flex mb-08">
    <div class="mr-auto">
      <div class={`flex items-center ${s1.cart_total_text}`} >
        <div class="ff-bold-italic fs20 mr-06">Total a pagar:</div>
        <div class="ff-bold c-blue mb-04">s/.</div>
        <div class="ml-04 ff-bold fs20 c-blue"> 
          {formatN(props.precio/100,2)}
        </div>
      </div>
    </div>
    <button class={`mr-20 ${s1.cart_button_1}`} 
      onClick={ev => {
        ev.stopPropagation()
        props.onAccion()
      }}
    >
      {props.accion} <i class="icon-right"></i>
    </button>
  </div>
}