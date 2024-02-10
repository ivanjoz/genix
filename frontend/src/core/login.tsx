import { Loading } from "~/core/main";
import { createSignal } from "solid-js";
import { Input } from "~/components/Input";
import { sendUserLogin } from "~/services/admin/login";

interface ILoginPage {
  setLogin: (v: boolean) => void
}

const SendLogin = async (form: any) => {
  Loading.standard("Enviando Credenciales...")
  form.EmpresaID = 1
  const result = await sendUserLogin(form)
  console.log(result)
  
  if(result.error){}
  Loading.remove()
}

export default function LoginPage(props: ILoginPage) {

  const [loginFrom, setLoginForm] = createSignal({})

  return <div class="p-rel">
    <img class="bg-image-1" src="images/background-1.webp" alt="" />
    <div class="flex ai-center h100 login-bg-c1 p-rel">
      <div class="login-bg-1 w100">
        <div class="login-tt flex ai-center h2">
          Iniciar Sesión
        </div>
        <div class="login-logo-c p-rel mb-08">
          <img class="w100 h100" src="/images/genix_logo.svg" alt="Genix Logo" />
        </div>
        <div class="flex-center h2 ff-bold c-steel4 mb-16">
          Gestor Empresarial en la Nube para MyPes
        </div>
        <Input required={true} css="mb-12 w100 big-1" label="Usuario" 
          saveOn={loginFrom()} save="Usuario" type="text" 
          validator={v => !!v && String(v||"").length > 4}  
        />
        <Input required={true} css="mb-12 w100 big-1" label="Contraseña" 
          saveOn={loginFrom()} save="Password" type="password" 
          validator={v => !!v && String(v||"").length > 4}    
        />
        <div class="flex w100 jc-center ai-center"
          style={{ "margin-top": "1.4rem" }}
        >
          <button class="bn1 big d-blue" onClick={ev => {
            ev.stopPropagation()
            SendLogin(loginFrom())
          }}>
            <i class="icon-login"></i> Ingresar
          </button>
        </div>
      </div>
    </div>
  </div>
}