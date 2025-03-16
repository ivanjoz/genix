import { createEffect, createMemo, createSignal, For, JSX, Show } from "solid-js"
import { CardArrowSteps } from "~/components/Cards"
import { CartFloatingProducto, cartProductos } from "./productos"
import { formatN, getCmsTable, makeEcommerceDB } from "~/shared/main"
import s1 from "./components.module.css"
import { Input } from "~/components/Input"
import { CiudadSelector } from "~/core/components"
import { throttle } from "~/core/main"

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

  return <div class="w100" style={{ padding: '8px' }}>
    <CardArrowSteps selected={cartOption()}
      options={[ 
        { id: 1, name: 'Carrito', icon: "icon-basket" }, 
        { id: 2, name: 'Datos de Envío', icon: "icon-doc-inv-alt" }, 
        { id: 3, name: 'Pago', icon: "icon-shield" }, 
        { id: 4, name: 'Confirmación', icon: "icon-ok" }, 
      ]}
      onSelect={opt => {
        setCartOption(opt.id)
      }}
      optionRender={e => {
        let name = e.name as (string|JSX.Element);
        if(e.id === 2){ name = <span>Datos de <br />Envío</span>  }
        else if(e.id === 4){ name = <span>Confirmación</span>  }
        return <div class="flex ai-center mt-01 ff-semibold">
          <i class={`h3 ${e.icon} mr-02`} style={{ "margin-left": "-6px" }}></i>
          <div style={{ "line-height": '1.1', "text-align": 'left' }}
            class="mr-08">{name}</div>
        </div>
      }}
    />
    <div class="w100 mt-08"></div>
    <Show when={cartOption() === 1}>
      <div class="w100 flex mb-08">
        <div class="ff-bold-italic fs20">Total a pagar:</div>
        <div class="ml-04 ff-bold fs20 c-blue">s/. {formatN(precio()/100,2)}</div>
        <div class="mr-auto"></div>
        <button class={`mr-20 ${s1.cart_button_1}`} 
          onClick={ev => {
            ev.stopPropagation()
            setCartOption(2)
          }}
        >
          Continuar <i class="icon-right"></i>
        </button>
      </div>
      <div class="h100 w100 grid" 
        style={{ "grid-template-columns": "1fr 1fr", "column-gap": "12px" }}
      >
        <For each={Array.from(cartProductos().values())}>
        {e => {
          console.log("rendering productos::", cartProductos())
          return <CartFloatingProducto producto={e.producto} cant={e.cant} mode={2} />
        }}
        </For>
      </div>
    </Show>
    <Show when={cartOption() === 2}>
      <div class="w100 flex mb-08">
        <div class="ff-bold-italic fs20">Total a pagar:</div>
        <div class="ml-04 ff-bold fs20 c-blue">s/. {formatN(precio()/100,2)}</div>
        <div class="mr-auto"></div>
        <button class={`mr-20 ${s1.cart_button_1}`} 
          onClick={ev => {
            ev.stopPropagation()
            setCartOption(3)
          }}
        >
          Relizar Pago <i class="icon-right"></i>
        </button>
      </div>
      <div class="flex-wrap w100-10">
        <Input label="Nombres" saveOn={cartForm()} save="nombres"  
          css="w-12x mb-10" required={true}
          onChange={() => saveCartForm()}
        />
        <Input label="Apellidos" saveOn={cartForm()} save="apellidos"  
          css="w-12x mb-10" required={true}
          onChange={() => saveCartForm()}
        />
        <Input label="Correo Electrónico" saveOn={cartForm()} save="email"  
          css="w-12x mb-10" required={true}
          onChange={() => saveCartForm()}
        />
        <CiudadSelector css={["w-12x mb-10","w-12x mb-10","w-12x mb-10"]}
          saveOn={cartForm()} save="ciudadID" 
          onChange={() => saveCartForm()}
        />
        <Input label="Dirección" saveOn={cartForm()} save="direccion"  
          css="w-24x mb-10" required={true}
          onChange={() => saveCartForm()}
        />
        <Input label="Referencia" saveOn={cartForm()} save="referencia"  
          css="w-24x mb-10"
          onChange={() => saveCartForm()}
        />
      </div>
    </Show>
  </div>
}