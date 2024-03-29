import { createSignal } from "solid-js"
import { PageContainer } from "~/core/page"

export default function Home() {

  const [form, setfForm] = createSignal({})
  
  return <PageContainer title="Index">
    <h1>Sistema de Gestión de Mypes</h1>
  </PageContainer>
}
