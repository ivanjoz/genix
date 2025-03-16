import { useLocation, useParams } from "@solidjs/router";
import { createEffect, createMemo, createSignal, For, JSX } from "solid-js";
import { Input } from "~/components/Input";
import { pageView, setIsRouteChanging } from "~/core/menu";
import { PageContainer } from "~/core/page";
import { componentsRenders, PageSectionRenderer } from "~/pages/page";
import { pageExample } from "~/pages/page-example";
import { useGaleriaImagesAPI } from "~/services/cms/galeria-images";
import { arrayToMapN } from "~/shared/main";
import { ImageGalery, ImageGalerySelector } from "./galeria-imagenes";
import * as styles from "./webpage.module.css";
import { IPageBlock, IPageParams, IPageSection } from "~/pages/page-components";

export const [pageViews, setPageViews] = createSignal({})

export default function CmsWebpage() {

  const location = useLocation()
  const [pageSections, setPageSections] = createSignal(pageExample)
  const [sectionSelected, setSectionSelected] = createSignal<IPageSection>()
  const [sectionParams, setSectionParams] = createSignal([] as IPageBlock[])
  const coponentsRendersMap = arrayToMapN(componentsRenders, 'type')
  const [galeriaImagenes] = useGaleriaImagesAPI()

  const params = useParams()

  createEffect(() => {
    import("./webpage.module.css")
    setIsRouteChanging(false)
    console.log("params::", params.webpage)
    console.log("location:: ",location.pathname)
  })

  const addSection = (seccionIdx: number, mode: 1 | 2) => {
    const newSections: IPageSection[] = []
    const blank = { type: 1, content: "demo", title: "demo" } as IPageSection
    const pageSections_ = pageSections()
    // console.log("args sections::",seccionIdx, mode)
    // debugger
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
    views={[[1,"Edición",],[2,"Imágenes"]]}
    pageStyle={{ 
      display: pageView() === 1 ? "grid" : "flex", 
      "grid-template-columns": "1.1fr 8px 4fr", 
      padding: '0', "background-color": "white" 
    }}
  >
    { 
    pageView() === 1 && <>
      <div class={`h100 px-08 py-08 p-rel flex-column ${styles.webpage_dev_card}`}
        style={{ "z-index": 51 }}
      >
        { !sectionSelected() &&
          <div>Seleccione una sección para editar su contenido1.</div>
        }
        { sectionSelected() &&
          <SeccionTypeSelector selected={sectionSelected()} 
            onSectionTypeSelect={pp => {
              const selected = {...sectionSelected()}
              selected.type = pp.type
              updateSectionSelected(selected)
            }}
          />
        }
        <For each={sectionParams()}>
          {e => {
            console.log("section params::", e)
            const applyUpdate = () => {
              const selected = {...sectionSelected()} as any
              updateSectionSelected(selected)
            }

            if(e.type === 1){
              return <Input css="mb-10"  inputCss="s5" label={e.name} 
                save={e.key as keyof IPageSection} saveOn={sectionSelected()} 
                onChange={() => {
                  applyUpdate()
                }}
              />
            } else if(e.type === 2){
              return <Input css="mb-10"  inputCss="s5" label={e.name} 
                save={e.key as keyof IPageSection} saveOn={sectionSelected()} 
                useTextArea={true}
                onChange={() => {
                  applyUpdate()
                }}
              />
            } else if(e.type === 5){
              return <ImageGalerySelector imagenes={galeriaImagenes()} 
                imageSelected={sectionSelected()[e.key] as string}
                onSelect={img => {
                  sectionSelected()[e.key] = img.Image as never
                  console.log("section updated::", sectionSelected())
                  applyUpdate()
                }}
              />
            }
            return <div>--</div>
          }}
        </For>
      </div>
      <div class="h100 w100" style={{ "background-color": "#fbaeae" }}></div>
      <div class={`h100 p-rel cms-1 ${styles.cms_content_layer}`}>
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
                const params = coponentsRendersMap.get(e.type)?.params || []
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
    </>
    }
    { 
    pageView() === 2 &&
    <ImageGalery imagenes={galeriaImagenes()}
      useUploader={true}
    />
    }
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
    let cN = `mt-06 mb-06 mr-06 ml-06 ${styles.seccion_card}`
    if(props.isSelected){ cN += " selected" }
    return cN
  }
  return <div class={getCN()} 
    onClick={ev => { ev.stopPropagation(); props.onSelect() }}>
    {props.params.name}
  </div>
}

export interface ISeccionTypeSelector {
  selected: IPageSection
  onSectionTypeSelect: (e: IPageParams) => void
}

const SeccionTypeSelector = (props: ISeccionTypeSelector) => {
  
  const [showLayer, setShowLayer] = createSignal(false)
  const coponentsRendersMap = arrayToMapN(componentsRenders, 'type')

  let inputRef: HTMLInputElement
  let avoidBlur = false
  let avoidClickUntil = 0

  return <div class="w100 p-rel mb-08">
    <button class={`flex ai-center w100 p-rel h3 ff-bold ${styles.btn_11}`}
      classList={{"open": showLayer() }}
      onClick={ev => {
        ev.stopPropagation()
        if(avoidClickUntil && avoidClickUntil > Date.now()){
          return
        }
        const showState = !showLayer()
        setShowLayer(showState)
        setTimeout(() => {
          if(showState && inputRef){ inputRef.focus() }
        },250)
      }}
    >
      {showLayer() ? <i class="icon-cancel"></i> : <i class="icon-menu"></i>}
      <div>{coponentsRendersMap.get(props.selected?.type)?.name}</div>
    </button>
    { showLayer() &&
      <>
        <input ref={inputRef} class={`p-abs ${styles.input_invisible}`}
          onBlur={ev => {
            ev.stopPropagation()
            if(avoidBlur){
              avoidBlur = false
              ev.target.focus()
            } else {
              setShowLayer(false)
              avoidClickUntil = Date.now() + 400
            }
          }}
        />
        <div class={`layer-angle1 w100 p-abs ${styles.layer_11a}`}></div>
        <div class={`p-abs ${styles.layer_11}`} 
          onMouseDown={ev => {
            ev.stopPropagation()
            avoidBlur = true
          }}
        >
          <div class="flex-wrap ac-baseline h100 w100">
            { componentsRenders.map(e => {
                const isSelected = createMemo(() => props.selected?.type === e.type)
                return <SeccionCard params={e} isSelected={isSelected()} 
                  onSelect={() => {
                    props.onSectionTypeSelect(e)
                    setShowLayer(false)
                  }}
                />
              })
            }
          </div>
        </div>
      </>
    }
  </div>
}