import { createSignal } from "solid-js"
import { Title } from "@solidjs/meta"
import { Counter } from "~/components/Counter"
import { PageContainer } from "~/core/page"

const options = [
  { id: 1, name: "hola mundo" },
  { id: 2, name: "dasdas dasdas" },
  { id: 3, name: "lalala dsd" },
  { id: 4, name: "musica pop" },
  { id: 5, name: "dasd mundo" },
  { id: 6, name: "ss 213 dca dasdas" },
  { id: 7, name: "laslala dsd" },
  { id: 8, name: "ds 412 ads d pop" },
  { id: 9, name: "hobadfla mundo" },
  { id: 10, name: "da dn bvsdsdas dasdas" },
  { id: 11, name: "ds dsd" },
  { id: 12, name: "mfsdgausica fadfasf" },
  { id: 13, name: "hola dasd" },
  { id: 14, name: "dasgggdas dasdas" },
  { id: 15, name: "lalsdd dd ala dsd" },
  { id: 16, name: "2ewadasf pop" },
]

export default function Home() {

  const [form, setfForm] = createSignal({})
  
  return <PageContainer title="Index">
    <Title>Hello World</Title>
    <h1>Hello world!</h1>
    <Counter />
    <div class="flex" style={{ width: '100%' }}>
  
    </div>
    <button onClick={ev => {
      ev.stopPropagation()
      console.log("form:: ", form)
      setfForm({})
    }}>
      Enviar
    </button>
    <p>
      Visit{" "}
      <a href="https://start.solidjs.com" target="_blank">
        start.solidjs.com
      </a>{" "}
      to learn how to build SolidStart apps. s
    </p>
  </PageContainer>
}
