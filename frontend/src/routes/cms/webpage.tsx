import { useLocation } from "@solidjs/router";
import { For, JSX, createEffect, createSignal } from "solid-js";
import { PageContainer } from "~/core/page";
import { IPageParams, IPageSection, ISectionParams, PageSectionRenderer } from "~/pages/page";
import { pageExample } from "~/pages/page-example";
import styles from "./webpage.module.css"
import { Input } from "~/components/Input";
import { pageParams } from "~/pages/page-params";

export const [pageViews, setPageViews] = createSignal({})

const sectionsParamsMap: Map<number,IPageParams> = new Map()

const getPageSectionParams = (type: number): IPageParams => {
  if(sectionsParamsMap.size === 0){
    for(let e of pageParams){ sectionsParamsMap.set(e.id,e) }
  }
  return sectionsParamsMap.get(type)
}

export default function CmsWebpage() {

  const location = useLocation()
  const [pageSections, setPageSections] = createSignal(pageExample)
  const [sectionSelected, setSectionSelected] = createSignal(-1)
  const [sectionParams, setSectionParams] = createSignal([] as ISectionParams[])

  createEffect(() => {
    console.log(location.pathname)
  })

  const addSection = (seccionIdx: number, mode: 1 | 2) => {
    const newSections: IPageSection[] = []
    const blank = { type: 1, content: "demo", title: "demo" } as IPageSection
    const pageSections_ = pageSections()
    console.log("args sections::",seccionIdx, mode)
    debugger
    for(let i = 0; i < pageSections_.length; i++){
      if(mode === 1 && seccionIdx === i){ newSections.push(blank) }
      newSections.push(pageSections_[i])
      if(mode === 2 && seccionIdx === i){ newSections.push(blank) }
    }
    console.log("nuevas seccciones::", newSections)
    setPageSections(newSections)
  }

  const moveSection = (seccionIdx: number, mode: 1 | 2) => {
    let mantain = false
    if(seccionIdx === (pageSections().length - 1) && mode === 2){ mantain = true }
    else if(seccionIdx === 0 && mode === 1){ mantain = true }

    if(mantain){ return }

    const sec = pageSections()[seccionIdx]
    const newSections: IPageSection[] = pageSections().filter(x => x !== sec)

    if(mode === 1){ seccionIdx-- }
    else if(mode === 2){ seccionIdx++ }
    
    newSections.splice(seccionIdx,0,sec)
    setPageSections(newSections)
  }

  return <PageContainer title="Webpage" class=""
    views={[[2,"Creador",],[1,"Secciones"]]}
    pageStyle={{ display: 'grid', "grid-template-columns": "1.1fr 4fr", 
      padding: '0', "background-color": "white" }}
  >
    <div class={`h100 px-08 py-08 p-rel flex-column ${styles.webpage_dev_card}`}>
      <For each={sectionParams()}>
        {e => {
          if(e.type === 1){
            return <Input css="mb-10"  inputCss="s5" label={e.name} 
              save="content" saveOn={e} 
              onChange={() => {
                console.log(e.content)
                const sec = pageSections()[sectionSelected()]
                sec[e.key] = e.content as never
                // console.log(sec)
                pageSections()[sectionSelected()] = {...sec}
                setPageSections([...pageSections()])
              }}
            />
          }
          return <div>--</div>
        }}
      </For>
    </div>
    <div class="h100 p-rel cms-1" style={{ 
      overflow: 'auto', "max-height": `calc(100vh - var(--header-height))`,
      padding: '2px', "z-index": 999, "margin-left": "-2px",
    }}>
    <For each={pageSections()}>
      {(e,i) => {
        const cN = () => {
          let cN_ = `p-rel w100 ${styles.cms_editable_card}`
          if(sectionSelected() === i()){ cN_ += ` ${styles.cms_editable_card_selected}`}
          return cN_
        }

        return <div class={cN()}
          onClick={ev => {
            ev.stopPropagation()
            setSectionSelected(i())
            const paramsBase = getPageSectionParams(e.type)?.params || []
            const params: ISectionParams[] = []
            for(let p_ of paramsBase){
              const p = {...p_}
              if(e[p.key]){ p.content = e[p.key] }
              params.push(p)
            }
            setSectionParams(params)
          }}
        >
          <SeccionButtons mode={1} onAddSeccion={() => addSection(i(),1)}
            onMoveSeccion={() => moveSection(i(),1)}
          />
          <PageSectionRenderer args={e} type={e.type} 
            
          />
          <SeccionButtons mode={2} onAddSeccion={() => addSection(i(),2)}
            onMoveSeccion={() => moveSection(i(),2)}
          />
        </div>
      }}
    </For>
    </div>
  </PageContainer>
}

interface ISeccionButtons {
  mode: 1 | 2 /* up | down */
  onAddSeccion: () => void
  onMoveSeccion: () => void
}

const SeccionButtons = (props: ISeccionButtons) => {
  const style1: JSX.CSSProperties = {}
  if(props.mode === 1){ style1.top = "0" }
  if(props.mode === 2){ style1.bottom = "0" }

  return <div class={`flex p-abs ${styles.cms_card_buttons}`} style={style1}>
    <div class={`flex ai-center p-rel ff-bold mr-06 ${styles.cms_card_button}`}
      onClick={ev => {
        ev.stopPropagation()
        props.onAddSeccion()
      }}
    >
      <i class="icon-plus h6"></i>Sección
    </div>
    <div class={`flex ai-center p-rel ff-bold mr-06 ${styles.cms_card_button} ${styles.cs1}`}
      onClick={ev => {
        ev.stopPropagation()
        props.onMoveSeccion()
      }}
    >
      { props.mode === 1 && <i class="icon-up-1"></i> }
      { props.mode === 2 && <i class="icon-down-1"></i> }
    </div>
  </div>
}