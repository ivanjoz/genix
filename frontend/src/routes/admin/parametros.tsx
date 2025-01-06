import { Input } from "~/components/Input"
import { Loading, Notify } from "~/core/main"
import { PageContainer } from "~/core/page"
import { postEmpresaParametros, useParametrosEmpresaAPI } from "~/services/admin/empresas"
import css from "../../css/layout.module.css"

export default function AdminParametros() {

  const [empresa, setEmpresa] = useParametrosEmpresaAPI()

  console.log("empresa:", empresa())
  
  const saveEmpresa = async () => {
    const form = empresa()
    if((form.ruc||"").length === 0 || (form.nombre||"").length === 0 || 
      (form.razonSocial||"").length === 0 || (form.email||"").length === 0 ){
      Notify.failure("Faltan datos a guardar.")
      return
    }

    Loading.standard("Guardando registros...")
    try {
      postEmpresaParametros(form)
    } catch (error) {
      console.warn(error); return
    }
    Loading.remove()
  }

  return <PageContainer title="Parámetros" fetchLoading={true}>
    <div class="flex jc-between ai-center mb-08">
      <div><div class="h3 ff-bold">Parámetros de la Empresa</div></div>
      <div class="flex ai-center">
        <button class="bn1 b-blue" onclick={ev => {
          ev.stopPropagation()
          saveEmpresa()
        }}>
          <i class="icon-floppy"></i>
          Guardar
        </button>
      </div>
    </div>
    <div class="w100 grid" style={{ "grid-template-columns": "4fr 3fr" }}>
      <div class="flex-wrap w100-10">
        <Input css="w-14x mb-10" label="Nombre" save="nombre" saveOn={empresa()}
          required={true} />
        <Input css="w-10x mb-10" label="RUC" save="ruc" saveOn={empresa()} 
          required={true} />
        <Input css="w-14x mb-10" label="Razón Social" save="razonSocial" 
          saveOn={empresa()} required={true} />
        <Input css="w-10x mb-10" label="Teléfono" save="telefono" saveOn={empresa()} />
        <Input css="w-14x mb-10" label="Correo Electrónico" save="email" 
          saveOn={empresa()} required={true} />
        <Input css="w-10x mb-10" label="Representante" save="representante" saveOn={empresa()} />
        <Input css="w-14x mb-10" label="Dirección Legal" save="direccion" 
          saveOn={empresa()} />
        <Input css="w-10x mb-10" label="Ciudad" save="ciudad" saveOn={empresa()} />
      </div>
      <div class="" style={{ "margin-left": "16px" }}>
        <div  class={`w100 ${css.layer_propiedades} py-10 px-12`}>
          <div class="w100 mb-08">Parámetros SMTP para notificaciones</div>
          <div class="w100-10 flex-wrap">
            <Input css="w-12x mb-10" inputCss="s3" label="Host" save="host" saveOn={empresa().smtp} />
            <Input css="w-12x mb-10" inputCss="s3" label="Port" save="port" saveOn={empresa().smtp} type="number" />
            <Input css="w-12x mb-10" inputCss="s3" label="Usuario" save="user" saveOn={empresa().smtp} />
            <Input css="w-12x mb-10" inputCss="s3" label="Password" save="pwd" saveOn={empresa().smtp} />
            <Input css="w-24x mb-10" inputCss="s3" label="Email" save="email" saveOn={empresa().smtp} />
          </div>
        </div>
      </div>
    </div>
  </PageContainer>
}
