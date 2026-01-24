import { clientOnly } from "@solidjs/start"
import { pageExample } from "./page-example"
const PageRenderer = clientOnly(() => import("./page"))

export default function PageBuilder() {
  console.log("renderizando page...")
  return <>
    <PageRenderer isEditable={false} sections={pageExample} />
  </>
}