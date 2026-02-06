<script lang="ts" module>

  export interface ElementAST {
    id?: number | string
    css?: string
    tagName?: string
    text?: string | number
    onClick?: (id: number | string) => void
    children?: ElementAST[]
  }

</script>

<script lang="ts">

  const {
    elements
  }: { elements: ElementAST | ElementAST[] } = $props()

  const handleClick = (element: ElementAST) => {
    if (element.onClick) {
      element.onClick(element.id || 0)
    }
  }

</script>

{#snippet renderElement(element: ElementAST)}
  {#if element.tagName === 'BUTTON'}
    <button
      class={element.css}
      onclick={() => handleClick(element)}
    >
      {#if element.text}
        {element.text}
      {/if}
      {#if element.children}
        {#each element.children as child}
          {@render renderElement(child)}
        {/each}
      {/if}
    </button>
  {:else}
    <div class={element.css}>
      {#if element.text}
        {element.text}
      {/if}
      {#if element.children}
        {#each element.children as child}
          {@render renderElement(child)}
        {/each}
      {/if}
    </div>
  {/if}
{/snippet}
{#if Array.isArray(elements)}
  {#each elements as element}
    {@render renderElement(element)}
  {/each}
{:else}
  {@render renderElement(elements)}
{/if}
