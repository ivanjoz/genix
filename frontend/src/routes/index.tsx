import { onMount } from "solid-js"
import PageBuilder from "~/pages/main"
import { Env, isLogin } from "~/shared/security"

export default function Home() {

  onMount(() => {
    if(isLogin() !== 2){
      Env.navigate("login")
    } else {
      // Env.navigate("admin-home")
    }
  })
  
  return <PageBuilder />
}
