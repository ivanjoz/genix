import { Loading, Notify } from "~/core/main"
import { createSignal } from "solid-js"
import { Input } from "~/components/Input"
import { Modal, setOpenModals } from "~/components/Modals"
import { QTable } from "~/components/QTable"
import { SearchCard, SearchSelect } from "~/components/SearchSelect"
import { formatTime, throttle } from "~/core/main"
import { PageContainer } from "~/core/page"
import { IUsuario, postUsuario, useEmpresasAPI, usePerfilesAPI, useUsuariosAPI } from "~/services/admin/empresas"

export default function UsuariosPage() {
  
  const [empresas] = useEmpresasAPI()
  const [usuarios, setUsuarios] = useUsuariosAPI()
  const [perfiles] = usePerfilesAPI()
  const [filterText, setFilterText] = createSignal("")

  const [usuarioForm, setUsuarioForm] = createSignal({} as IUsuario)

  const saveUsuario = async (isDelete?: boolean) => {
    const form = usuarioForm()
    if((form.usuario?.length||0) < 4 || (form.nombres?.length||0) < 4){
      Notify.failure("El usuario y el nombre deben tener al menos 4 caracteres.")
      return
    }
    if(form.password1) form.password1 = form.password1.trim()
    if(form.password2) form.password2 = form.password2.trim()

    if(!form.id || form.password1){
      let err = ""
      if((form.password1.length||0) < 6){
        err = "El password tiene menos de 6 caracteres."
      } else if(form.password1 !== form.password2){
        err = "Los password no coinciden."
      }
      if(err){ Notify.failure(err); return }
    }

    Loading.standard("Creando /Actualizando Usuario...")
    try {
      var result = await postUsuario(form)
    } catch (error) {
      Notify.failure(error as string); Loading.remove(); return
    }

    let usuarios_ = [...usuarios().usuarios]

    if(form.id){
      const selected = usuarios().usuarios.find(x => x.id === form.id)
      if(selected){ Object.assign(selected, form) }
      if(isDelete){ usuarios_ = usuarios_.filter(x => x.id !== form.id) }
    } else {
      form.id = result.id
      usuarios_.unshift(form)
    }
    Loading.remove()
    
    setUsuarios({...usuarios(), usuarios: usuarios_ })
    setOpenModals([])
  }

  return <PageContainer title="Usuarios" fetchLoading={true}>
    <div class="h100">
      <div class="flex ai-center jc-between mb-06">
        <div class="search-c4 mr-16 w16rem">
          <div><i class="icon-search"></i></div>
          <input class="w100" autocomplete="off" type="text" onKeyUp={ev => {
            ev.stopPropagation()
            throttle(() => {
              setFilterText(((ev.target as any).value||"").toLowerCase().trim())
            },150)
          }}/>
        </div>
        <div class="flex ai-center">
          <button class="bn1 b-green" onClick={ev => {
            ev.stopPropagation()
            setUsuarioForm({ ss: 1 } as IUsuario)
            setOpenModals([1])
          }}>
            <i class="icon-plus"></i>
          </button>
        </div>
      </div>
      <QTable data={usuarios()?.usuarios||[]} css="w100"
        maxHeight="calc(80vh - 13rem)" style={{ "height": '100%' }}
        columns={[
          { header: "ID", headerStyle: { width: '3.4rem' }, css: "c",
            getValue: e => e.id
          },
          { header: "Usuario", cardCss: "h3 ff-bold c-purple",
            getValue: e => e.usuario, cardColumn: [1,1]
          },
          { header: "Nombres", cardColumn: [2,1],
            getValue: e => e.nombres +" "+ e.apellidos
          },
          { header: "Email", cardColumn: [3,1], cardCss: "h5 c-steel",
            getValue: e => e.email
          },
          { header: "Estado", cardColumn: [3,2],
            getValue: e => e.ss, cardCss: "h5 c-steel",
          },
          { header: "Actualizado", headerStyle: { width: '9rem' },
            css: 'nowrap',
            getValue: e => formatTime(e.upd,"Y-m-d h:n") as string
          },
          { header: "...", headerStyle: { width: '2.6rem' }, css: "t-c",
            cardColumn: [1,2],
            render: (e,i) => {
              const onclick = (ev: MouseEvent) => {
                ev.stopPropagation()
                setOpenModals([1])
                setUsuarioForm({...e})
              }
              return <button class="bnr2 b-blue b-card-1" onClick={onclick}>
                <i class="icon-pencil"></i>
              </button>
            }
          }
        ]}    
      />
    </div>
    <Modal id={1} css="w56-78 in-s2"
      title={(usuarioForm()?.id > 0 ? "Actualizar" : "Guardar") + " Usuario"}
      onSave={()=> {
        console.log("usuario a guardar::", usuarioForm())
        saveUsuario()
      }}
      onDelete={()=> {
        saveUsuario(true)
      }}
    >
      <div class="flex-wrap ai-start w100-10">
        <Input saveOn={usuarioForm()} save="usuario" 
          css="w-12x mb-10" label="Usuario" required={true}
          disabled={usuarioForm()?.id > 0}
        />
        <Input saveOn={usuarioForm()} save="nombres" 
          css="w-12x mb-10" label="Nombres" required={true}
        />
        <Input saveOn={usuarioForm()} save="apellidos" 
          css="w-12x mb-10" label="Apellidos"
        />
        <Input saveOn={usuarioForm()} save="documentoNro" 
          css="w-12x mb-10" label="Nº Documento"
        />
        <Input saveOn={usuarioForm()} save="cargo" 
          css="w-12x mb-10" label="Cargo"
        />
        <Input saveOn={usuarioForm()} save="email" 
          css="w-12x mb-10" label="Email"
        />
        <SearchCard saveOn={usuarioForm()} save="perfilesIDs" 
          css="w100 mb-10 ai-start" label="" placeholder="Perfiles"
          options={perfiles().perfiles} keys="id.nombre" inputCss="w-07x"
        />
        <Input saveOn={usuarioForm()} save="password1" 
          css="w-12x mb-10" label="Password" type="password"
          required={!usuarioForm().id}
          placeholder={usuarioForm().id > 0 ? "SIN CAMBIAR" : ""}
        />
        <Input saveOn={usuarioForm()} save="password2" required={!usuarioForm().id}
          css="w-12x mb-10" label="Password (Repetir)" type="password"
        />
      </div>
    </Modal>
  </PageContainer>
}
