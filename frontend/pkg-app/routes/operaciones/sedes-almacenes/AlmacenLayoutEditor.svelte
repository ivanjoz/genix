<script lang="ts">
import Input from '$ui/components/Input';
  import type { IAlmacen, IAlmacenLayout } from "./sedes-almacenes.svelte"

  interface Props {
    almacen: IAlmacen
  }

  let { almacen = $bindable() }: Props = $props()

  const layouts = $derived(almacen.Layout || [])

  const addLayout = () => {
    const maxID = layouts.length > 0 
      ? Math.max(...layouts.map(x => x.ID || 0)) 
      : 0
    layouts.push({ RowCant: 2, ColCant: 3, Name: "", ID: maxID + 1, Bloques: [] })
    almacen.Layout = [...layouts]
  }

  const removeLayout = (idx: number) => {
    almacen.Layout = layouts.filter((_, i) => i !== idx)
  }

  const updateLayout = () => {
    almacen.Layout = [...layouts]
  }
</script>

<div class="w-full h-full relative">
  <div class="flex justify-between w-full mb-8 px-12 pt-12">
    <div></div>
    <div class="flex items-center">
      <button class="bx-green" aria-label="Agregar Layout" onclick={addLayout}>
        <i class="icon-plus"></i>
      </button>
    </div>
  </div>
  
  <div class="overflow-auto px-4" style="max-height: calc(100% - 60px)">
    {#if layouts.length === 0}
      <div class="bg-red-100 text-red-700 p-8 rounded">
        No hay espacios en al almacén. Agregue uno pulsando en (+)
      </div>
    {/if}
    
    {#each layouts as _, idx (layouts[idx].ID)}
      {@const layout = layouts[idx]}
      {@const heads = Array.from({ length: layout.ColCant || 1 }, (_, i) => String(i + 1))}
      {@const rows = Array.from({ length: layout.RowCant || 1 }, (_, i) => String(i + 1))}
      
      <div class="_1 bg-white rounded-lg shadow-sm p-8 mb-12">
        <div class="w-full flex items-center justify-between px-8 py-8">
          <div class="flex items-center">
            <Input bind:saveOn={layouts[idx]} save="Name" 
              css="shadow-small bg-solid no-border w-220 mr-12" inputCss="text-sm" required={true}
            />
            <span class="ff-bold text-slate-600">Filas</span>
            <Input bind:saveOn={layouts[idx]} save="RowCant" 
              css="shadow-small bg-solid no-border w-60 mx-4" inputCss="text-sm" type="number"
              onChange={updateLayout}
            />
            <span class="ff-bold text-slate-600">Niveles</span>
            <Input bind:saveOn={layouts[idx]} save="ColCant" 
              css="shadow-small bg-solid no-border w-60 mx-4" inputCss="text-sm" type="number"
              onChange={updateLayout}
            />
          </div>
          <button class="bnr4 b-red" aria-label="Eliminar Layout"
            style="margin-top: -12px; margin-right: -4px"
            onclick={() => removeLayout(idx)}
          >
            <i class="icon-trash"></i>
          </button>
        </div>
        
        <table class="w-full">
          <thead>
            <tr>
              <th style="width: 3rem">-</th>
              {#each heads as head}
                <th style="width: calc(92% / {heads.length})">{head}</th>
              {/each}
            </tr>
          </thead>
          <tbody>
            {#each rows as row}
              <tr>
                <td class="text-center">{row}</td>
                {#each heads as col}
                  <td class="relative py-2 px-2" style="height: 2.6rem">
                    <Input label="" bind:saveOn={layouts[idx]} save={`xy_${row}_${col}`}
                      css="shadow-small bg-solid no-border w-full" inputCss="text-sm text-center"
                    />
                  </td>
                {/each}
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/each}
  </div>
</div>

<style>
  ._1 {
    background-color: var(--light-blue-1);
    border-radius: 5px;
    box-shadow: #5f7187a8 0 1px 3px -1px;
    min-height: 50px;
  }
</style>