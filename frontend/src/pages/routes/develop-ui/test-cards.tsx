import { CardsList } from '~/components/Cards';
import { PageContainer } from '~/core/page';

export default function TestCards() {

  const data = makeData()

  return <PageContainer title="Virtualize Scroll">
  <div style={{ padding: '1rem' }}>
      <div style={{ "margin-bottom": '1rem' }}> <h2>VIRTUALIZE CARDS</h2> </div>
      <CardsList data={data} 
        render={e => {
          return <div style={{ padding: '8px' }}>
            <div style={{ width: '100%', height: '50px', "background-color":'#aeaef8' }}>
              {e.nombre}
            </div>
          </div>
        }}
      />
    </div>
  </PageContainer>
}

const makeData = (): any[] => {
  console.log("generando data:: ")

  const records = []

  for(let i = 0; i< 50000; i++){

    const record = {
      id: makeid(12),
      edad: Math.floor(Math.random() * 100),
      nombre: makeid(18),
      apellidos: makeid(23),
    }  
    records.push(record)
  }
  return records
}

function makeid(length: number) {
  let result = '';
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  const charactersLength = characters.length;
  let counter = 0;
  while (counter < length) {
    result += characters.charAt(Math.floor(Math.random() * charactersLength));
    counter += 1;
  }
  return result;
}