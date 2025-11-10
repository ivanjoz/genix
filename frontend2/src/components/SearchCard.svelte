<script lang="ts" generics="T,E">
    import { untrack } from "svelte";
  import SearchSelect from "./SearchSelect.svelte";
    import { WeakSearchRef } from "../core/store.svelte";

  interface SearchSelectProps<T> {
    saveOn?: T;
    save?: keyof T;
    css?: string;
    options: E[];
    keyId: keyof E;
    keyName: keyof E;
    label?: string;
    cardCss?: string
    inputCss?: string
    onChange?: (e: (string|number)[]) => void;
  }

  const {
    saveOn = $bindable(),
    save,
    css = "",
    options = [],
    keyId,
    keyName,
    label,
    onChange,
    inputCss,
    cardCss
  }: SearchSelectProps<T> = $props();

  let selectedIDs = $state<(number|string)[]>([])

  const buildMap = () => {
    if(WeakSearchRef.has(options)){ return }
    const idToRecord: Map<string|number, E> = new Map()
    const valueToRecord: Map<string,E> = new Map()

    for(const e of options){
      idToRecord.set(e[keyId] as (string|number), e)
      valueToRecord.set((e[keyName] as string).toLowerCase(), e)
    }

    WeakSearchRef.set(options, { idToRecord, valueToRecord })
  }

  buildMap()

  $effect(() => {
    if(options?.length > 0){ buildMap() }
  })

  const doOnChange = () => {
    if(saveOn && save){
      saveOn[save] = selectedIDs as NonNullable<T>[keyof T]
    }
    if(onChange){ onChange(selectedIDs) }
    //console.log("Change selectedIDs:", $state.snapshot(selectedIDs), $state.snapshot(saveOn), saveOn, save)
  }

  let prevSaveOn: T | undefined

  $effect(() => {
    // console.log("Nuevo SaveOn:", $state.snapshot(saveOn))
    if(saveOn && save && saveOn !== prevSaveOn){
      prevSaveOn = saveOn
      buildMap()
      untrack(() => {
        // console.log("Asignando SaveOn:", $state.snapshot(saveOn))
        selectedIDs = saveOn[save] as (number|string)[] || []
      })
    }
  })

  const getOption = (id: string | number): E => {
    const idToRecord = WeakSearchRef.get(options)?.idToRecord || new Map()
    return idToRecord.get(id) as E || {[keyName]: `ID-${id}`, [keyId]: id } as E
  }

</script>


<div class={css}>
  <SearchSelect options={options} keyId={keyId} keyName={keyName} 
    clearOnSelect={true} avoidIDs={selectedIDs} placeholder={label} 
    css={"s1 "+inputCss}
    onChange={e => {
      if(!e){ return }
      const id = e[keyId] as number
      if(!selectedIDs.includes(id)){
        selectedIDs.push(id)        
        doOnChange()
      }
    }}
  />
  <div class="p-4 min-h-40 _2 flex flex-wrap {cardCss}">
    {#each selectedIDs as id }
      {@const el = getOption(id)}
      <div class="m-2 px-8 min-w-56 h-32 lh-10 flex _3">
        { el[keyName] as string }
        <button class="_4 absolute w-28 h-28 rounded right-2" aria-label="eliminar"
          onclick={ev => {
            ev.stopPropagation()
            selectedIDs = selectedIDs.filter(x => x !== id)
            doOnChange()
          }}
        >
          <i class="icon-trash"></i>
        </button>
      </div>
    {/each}
  </div>
</div>

<style>
  ._2 {
    background-color: var(--light-blue-1);
    border-radius: 5px;
    box-shadow: #5f7187a8 0 1px 3px -1px;
  }
  ._3 {
    background-color: #fff;
    justify-content: center;
    border-radius: 4px;
    border: 1px solid #dfe1ea;
    color: #3d497a;
    align-items: center;
    cursor: pointer;
    position: relative;
    user-select: none;
  }
  ._3:hover {
    border-color: rgb(236, 125, 125);
    color: rgb(209, 66, 66);
  }
  ._3:hover ._4 {
    opacity: 1;
    background-color: rgb(255, 221, 221);
    color: rgb(224, 61, 61);
  }
  ._3 ._4 {
    font-size: 14px;
    border-radius: 50%;
    opacity: 0;
  }
  ._3 ._4:hover {
    background-color: rgb(240, 102, 102);
    color: white;
  }
</style>