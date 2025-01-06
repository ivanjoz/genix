import { onMount } from "solid-js"
import { Env, isLogin } from "~/shared/security"

export default function Home() {

  onMount(() => {
    if(isLogin() !== 2){
      Env.navigate("login")
    } else {
      Env.navigate("home")
    }
  })
  
  return <div></div>
}
