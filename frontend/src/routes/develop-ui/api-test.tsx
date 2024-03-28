import { PageContainer } from '~/core/page';
import { GET } from "~/shared/http";
import  * as cbor from 'cbor-web'

export default function ApiTest() {
  GET({
    route: 'p-demo-serialization'
  }).then(result => {
    const base64 = result.base64
    //convert base64 into array of bytes
    const bytes = Uint8Array.from(atob(base64), c => c.charCodeAt(0))
    console.log(cbor)
    const decoded = cbor.decodeAllSync(bytes)
    console.log("resultado serializacion::",decoded);
  })

  return <PageContainer title="Table demo">
    <div>hola</div>
  </PageContainer>
}

