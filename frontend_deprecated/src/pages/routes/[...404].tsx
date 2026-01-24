import { PageContainer } from "~/core/page";

export default function NotFound() {
  return <PageContainer title="No Encontrado" fetchLoading={true}>
    <div class="h1">No se encontró la página buscada...</div>
  </PageContainer>
}
