import { useLocation } from "@solidjs/router";
import { For, JSX, createEffect, createMemo, createSignal } from "solid-js";
import { PageContainer } from "~/core/page";
import { IPageParams, IPageSection, ISectionParams, PageSectionRenderer, PageSectionsDefs } from "~/pages/page";
import { pageExample } from "~/pages/page-example";
import styles from "./webpage.module.css"
import { Input } from "~/components/Input";
import { coponentsRenders } from "~/pages/page-components";
import { pageView } from "~/core/menu";
import { arrayToMapN } from "~/shared/main";

export const [pageViews, setPageViews] = createSignal({})

const sectionsParamsMap: Map<number,IPageParams> = new Map()

const getPageSectionParams = (type: number): IPageParams => {
  if(sectionsParamsMap.size === 0){
    for(let e of coponentsRenders){ sectionsParamsMap.set(e.type,e) }
  }
  return sectionsParamsMap.get(type)
}

export default function CmsWebpage() {

  const location = useLocation()
  const [pageSections, setPageSections] = createSignal(pageExample)
  const [sectionSelected, setSectionSelected] = createSignal<IPageSection>()
  const [sectionParams, setSectionParams] = createSignal([] as ISectionParams[])
  const pageSectionsDefsMap = arrayToMapN(PageSectionsDefs,'id')

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

  const updateSectionSelected = (selected: IPageSection) => {
    const newPageSections = []
    for(const ps of pageSections()){
      if(ps.id === selected.id){ newPageSections.push(selected) }
      else { newPageSections.push(ps) }
    }
    setSectionSelected(selected)
    setPageSections(newPageSections)
  }

  return <PageContainer title="Webpage" class=""
    views={[[1,"Editor",],[2,"Secciones"]]}
    pageStyle={{ display: 'grid', "grid-template-columns": "1.1fr 4fr", 
      padding: '0', "background-color": "white" }}
  >
    { 
    pageView() === 1 &&
    <div class={`h100 px-08 py-08 p-rel flex-column ${styles.webpage_dev_card}`}>
      { !sectionSelected() &&
        <div>Seleccione una sección para editar su contenido.</div>
      }
      <For each={sectionParams()}>
        {e => {
          if(e.type === 1){
            return <Input css="mb-10"  inputCss="s5" label={e.name} 
              save="content" saveOn={e} 
              onChange={() => {
                console.log(e.content)
                const selected = {...sectionSelected()}
                selected[e.key] = e.content as never
                updateSectionSelected(selected)
              }}
            />
          }
          return <div>--</div>
        }}
      </For>
    </div>
    }
    { 
    pageView() === 2 &&
    <div class={`h100 px-08 py-08 p-rel flex-column ${styles.webpage_dev_card}`}>
      { coponentsRenders.map(e => {
          const isSelected = createMemo(() => sectionSelected()?.type === e.type)
          return <SeccionCard params={e} isSelected={isSelected()} 
            onSelect={() => {
              if(sectionSelected()){
                const selected = {...sectionSelected()}
                selected.type = e.type
                updateSectionSelected(selected)
              }
            }}
          />
        })
      }
    </div>
    }
    <div class="h100 p-rel cms-1" style={{ 
      overflow: 'auto', "max-height": `calc(100vh - var(--header-height))`,
      padding: '2px', "z-index": 999, "margin-left": "-2px",
    }}>
      <For each={pageSections()}>
        {(e,i) => {
          const cN = () => {
            let cN_ = `p-rel w100 ${styles.cms_editable_card}`
            if(sectionSelected()?.id === e.id){ cN_ += ` ${styles.cms_editable_card_selected}`}
            return cN_
          }
          if(e.marginTop){ e.marginTop = undefined }
          if(e.marginBottom){ e.marginBottom = undefined }

          return <div class={cN()}
            onClick={ev => {
              ev.stopPropagation()
              setSectionSelected(e)
              const params: ISectionParams[] = []
              for(const param of (getPageSectionParams(e.type)?.params || [])){
                const id = typeof param === 'number' ? param : param[0] 
                const p = {...pageSectionsDefsMap.get(id)}
                const nameCustom = typeof param === 'number' ? "" : param[1]
                if(nameCustom){ p.name = nameCustom }

                if(e[p.key]){ p.content = e[p.key] }
                params.push(p)
              }
              console.log("params a renderizar::", params)
              setSectionParams(params)
            }}
          >
            <SeccionButtons mode={1} onAddSeccion={() => addSection(i(),1)}
              onMoveSeccion={() => moveSection(i(),1)}
            />
            <PageSectionRenderer args={e} type={e.type} />
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

interface ISeccionCard {
  params: IPageParams
  isSelected: boolean
  onSelect?: () => void
}

const SeccionCard = (props: ISeccionCard) => {
  const getCN = () => {
    let cN = `mt-04 mb-04 ${styles.seccion_card}`
    if(props.isSelected){ cN += " selected" }
    return cN
  }
  return <div class={getCN()} 
    onClick={ev => { ev.stopPropagation(); props.onSelect() }}>
    {props.params.name}
  </div>
}