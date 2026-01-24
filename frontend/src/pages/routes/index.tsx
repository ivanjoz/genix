import { onMount } from "solid-js"
import PageBuilder from "~/pages/main"
import { Env, getToken } from "~/shared/security"

export default function Home() {

  onMount(() => {
    if(!Env.getEmpresaID() || !getToken(true)){
      console.log("Empresa ID:",Env.getEmpresaID(),"| Token:", getToken(true))
      // Env.navigate("login")
    } else {
      // Env.navigate("admin-home")
    }
  })
  
  return <PageBuilder />
}
