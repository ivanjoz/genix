<script lang="ts" generics="T">

  let { 
    options, selected, keyField, valueField, buttonCss, onSelect
  }: { 
    options: T[], 
    selected: number,
    keyField?: keyof T,
    valueField?: keyof T,
    buttonCss?: string,
    onSelect: (e: T) => void
  } = $props()

  const getClass = (e: T) => {
    let cn = "flex items-center ff-bold _1"
    const id = Array.isArray(e) ? e[0] : (e?.[keyField] || 0)
    if(id === selected){ cn += " _2" }
    if(buttonCss){ cn += " " + buttonCss }
    return cn
  }

  const getValue = (e: T): string => {
    if(Array.isArray(e)){
      return e[1] as string
    } else if(valueField){
      return e[valueField] as string
    } else {
      return ""
    }
  }

</script>

<div class="pb-4 md:pb-0 flex items-center shrink-0 max-w-[100%] overflow-x-auto overflow-y-hidden">
  {#each options as opt }
    <button class={getClass(opt)} onclick={ev => {
      ev.stopPropagation()
      onSelect(opt)
    }}>
      { getValue(opt) }
    </button>
  {/each}
</div>

<style>
  ._1 {
    padding: 0 6px 2px 6px;
    margin: 0 4px;
    min-width: 114px;
   /* font-family: bold; */
    color: #9d9dac;
    border-bottom: 4px solid rgba(0, 0, 0, 0.1);
    user-select: none;
    cursor: pointer;
    display: flex;
    text-align: center;
    justify-content: center;
    align-items: end;
  }
  ._2 {
    color: #4343ad;
    border-bottom: 4px solid rgb(117 108 233);
  }

  @media (max-width: 750px) {
    ._1 {
      height: 38px;
    }
  }
</style>