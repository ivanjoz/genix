<script lang="ts">
  import type { ITextLine } from '../renderer/EcommerceRenderer.svelte';
  import { twMerge } from 'tailwind-merge';

  let { 
    textLines = $bindable([]), 
    editable = false 
  }: { 
    textLines: ITextLine[], 
    editable?: boolean 
  } = $props();

  let activeHoverIndex = $state(-1);

  const toggleClass = (index: number, className: string, pattern?: RegExp) => {
    let currentCss = textLines[index].css || '';
    if (pattern) {
      // Remove classes matching the pattern before adding the new one
      currentCss = currentCss.split(' ').filter(c => !pattern.test(c)).join(' ');
    }
    
    if (currentCss.includes(className) && !pattern) {
      textLines[index].css = twMerge(currentCss.replace(className, ''));
    } else {
      textLines[index].css = twMerge(currentCss, className);
    }
  };

  const sizes = [
    { label: 'XS', class: 'text-xs' },
    { label: 'SM', class: 'text-sm' },
    { label: 'MD', class: 'text-base' },
    { label: 'LG', class: 'text-lg' },
    { label: 'XL', class: 'text-xl' },
    { label: '2XL', class: 'text-2xl' }
  ];

  const colors = [
    { label: 'B', class: 'text-black', color: '#000' },
    { label: 'W', class: 'text-white', color: '#fff' },
    { label: 'R', class: 'text-red-500', color: 'red' },
    { label: 'G', class: 'text-green-500', color: 'green' },
    { label: 'B', class: 'text-blue-500', color: 'blue' },
    { label: 'G1', class: 'text-[__COLOR:1__]' },
    { label: 'G2', class: 'text-[__COLOR:2__]' }
  ];

</script>

<div class="text-block-container w-full">
  {#if editable}
    {#each textLines as line, i}
      <div 
        class="relative group mb-4 p-2 border border-transparent hover:border-blue-300 rounded transition-all"
        onmouseenter={() => activeHoverIndex = i}
        onmouseleave={() => activeHoverIndex = -1}
      >
        <!-- Toolbar -->
        {#if activeHoverIndex === i}
          <div class="absolute -top-10 left-0 z-10 flex items-center gap-1 bg-white shadow-xl border rounded-md p-1 animate-in fade-in slide-in-from-bottom-2 duration-200">
            <!-- Font Style -->
            <div class="flex border-r pr-1 mr-1">
              <button onclick={() => toggleClass(i, 'font-bold')} class="p-1 hover:bg-gray-100 rounded font-bold w-7 h-7 flex items-center justify-center" title="Bold">B</button>
              <button onclick={() => toggleClass(i, 'italic')} class="p-1 hover:bg-gray-100 rounded italic w-7 h-7 flex items-center justify-center" title="Italic">I</button>
              <button onclick={() => toggleClass(i, 'underline')} class="p-1 hover:bg-gray-100 rounded underline w-7 h-7 flex items-center justify-center" title="Underline">U</button>
            </div>

            <!-- Size Selection -->
            <select 
              class="text-xs border-r pr-1 mr-1 bg-transparent"
              onchange={(e) => toggleClass(i, e.currentTarget.value, /^text-(xs|sm|base|lg|xl|2xl|3xl)/)}
            >
              <option value="">Size</option>
              {#each sizes as size}
                <option value={size.class}>{size.label}</option>
              {/each}
            </select>

            <!-- Color Selection -->
            <div class="flex border-r pr-1 mr-1">
              {#each colors as color}
                <button 
                  onclick={() => toggleClass(i, color.class, /^text-/)} 
                  class="w-4 h-4 rounded-full border border-gray-200" 
                  style="background-color: {color.color || 'transparent'};"
                  title={color.label}
                ></button>
              {/each}
            </div>

            <!-- Layout -->
            <div class="flex border-r pr-1 mr-1">
              <button onclick={() => toggleClass(i, 'pl-4', /^pl-/)} class="p-1 hover:bg-gray-100 rounded text-xs" title="Tab">Tab</button>
              <button onclick={() => toggleClass(i, 'mt-4', /^mt-/)} class="p-1 hover:bg-gray-100 rounded text-xs" title="Margin Top">MT</button>
              <button onclick={() => toggleClass(i, 'mb-4', /^mb-/)} class="p-1 hover:bg-gray-100 rounded text-xs" title="Margin Bottom">MB</button>
            </div>

            <!-- Delete -->
            <button 
              onclick={() => textLines.splice(i, 1)} 
              class="p-1 hover:bg-red-100 text-red-500 rounded w-7 h-7 flex items-center justify-center"
              title="Delete"
            >
              &times;
            </button>
          </div>
        {/if}

        <textarea
          bind:value={line.text}
          class="w-full p-2 border-none bg-transparent focus:ring-0 resize-y {line.css}"
          placeholder="Enter text..."
        ></textarea>
        
        <div class="text-[10px] text-gray-400 opacity-0 group-hover:opacity-100 transition-opacity flex justify-between">
          <span>Classes: {line.css || 'none'}</span>
        </div>
      </div>
    {/each}
    
    <button 
      onclick={() => textLines.push({ text: '', css: '' })}
      class="w-full py-2 border-2 border-dashed border-gray-200 hover:border-blue-400 hover:text-blue-500 rounded-lg text-sm text-gray-400 transition-colors"
    >
      + Add new text block
    </button>
  {:else}
    {#each textLines as line}
      <div class={line.css}>
        {line.text}
      </div>
    {/each}
  {/if}
</div>

<style>
  /* Optional: ensure the container has enough space for the floating toolbar */
  .text-block-container {
    padding-top: 10px;
  }
</style>
