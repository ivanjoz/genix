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
    if((form.RUC||"").length === 0 || (form.Nombre||"").length === 0 || 
      (form.RazonSocial||"").length === 0 || (form.Email||"").length === 0 ){
      Notify.failure("Faltan datos a guardar.")
      return
    }

    Loading.standard("Guardando registros...")
    try {
      await postEmpresaParametros(form)
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
        <Input css="w-14x mb-10" label="Nombre" save="Nombre" saveOn={empresa()}
          required={true} />
        <Input css="w-10x mb-10" label="RUC" save="RUC" saveOn={empresa()} 
          required={true} />
        <Input css="w-14x mb-10" label="Razón Social" save="RazonSocial" 
          saveOn={empresa()} required={true} />
        <Input css="w-10x mb-10" label="Teléfono" save="Telefono" saveOn={empresa()} />
        <Input css="w-14x mb-10" label="Correo Electrónico" save="Email" 
          saveOn={empresa()} required={true} />
        <Input css="w-10x mb-10" label="Representante" save="Representante" saveOn={empresa()} />
        <Input css="w-14x mb-10" label="Dirección Legal" save="Direccion" 
          saveOn={empresa()} />
        <Input css="w-10x mb-10" label="Ciudad" save="Ciudad" saveOn={empresa()} />
        <div class="w100 mb-06"><div class="h3 ff-bold">Ecommerce</div></div>
        <Input css="w-12x mb-10" label="Llave Pública (Pruebas)"
          save="LlavePubDev" saveOn={empresa().CulquiConfig} />
        <Input css="w-12x mb-10" label="Llave Privada (Pruebas)"
          save="LlaveDev" saveOn={empresa().CulquiConfig} />
        <Input css="w-12x mb-10" label="Llave Pública (Live)"
          save="LlavePubLive" saveOn={empresa().CulquiConfig} />
        <Input css="w-12x mb-10" label="Llave Privada (Live)"
          save="LlaveLive" saveOn={empresa().CulquiConfig} />
        <Input css="w-12x mb-10" label="Culqui LLave RSA ID" 
          save="RsaKeyID" saveOn={empresa().CulquiConfig} />
        <Input css="w-12x mb-10" label="Culqui LLave RSA" 
          save="RsaKey" saveOn={empresa().CulquiConfig} />
      </div>
      <div class="" style={{ "margin-left": "16px" }}>
        <div  class={`w100 ${css.layer_propiedades} py-10 px-12`}>
          <div class="w100 mb-08">Parámetros SMTP para notificaciones</div>
          <div class="w100-10 flex-wrap">
            <Input css="w-12x mb-10" inputCss="s3" label="Host" save="Host" saveOn={empresa().SmtpConfig} />
            <Input css="w-12x mb-10" inputCss="s3" label="Port" save="Port" saveOn={empresa().SmtpConfig} type="number" />
            <Input css="w-12x mb-10" inputCss="s3" label="Usuario" save="User" saveOn={empresa().SmtpConfig} />
            <Input css="w-12x mb-10" inputCss="s3" label="Password" save="Password" saveOn={empresa().SmtpConfig} />
            <Input css="w-24x mb-10" inputCss="s3" label="Email" save="Email" saveOn={empresa().SmtpConfig} />
          </div>
        </div>
      </div>
    </div>
  </PageContainer>
}
