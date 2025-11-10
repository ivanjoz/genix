<script lang="ts" module>

  export interface Element {
    id: number | string
    css?: string
    content?: string
    isButton?: boolean
    onClick?: (id: number | string) => void
    children?: Element[]
  }

</script>

<script lang="ts">

  const { 
    elements 
  }: { elements: Element[] } = $props()

  const handleClick = (element: Element) => {
    if (element.onClick) {
      element.onClick(element.id)
    }
  }

</script>

{#snippet renderElement(element: Element)}
  {#if element.isButton}
    <button
      class={element.css}
      onclick={() => handleClick(element)}
    >
      {#if element.content}
        {element.content}
      {/if}
      {#if element.children}
        {#each element.children as child (child.id)}
          {@render renderElement(child)}
        {/each}
      {/if}
    </button>
  {:else}
    <div class={element.css}>
      {#if element.content}
        {element.content}
      {/if}
      {#if element.children}
        {#each element.children as child (child.id)}
          {@render renderElement(child)}
        {/each}
      {/if}
    </div>
  {/if}
{/snippet}

{#each elements as element (element.id)}
  {@render renderElement(element)}
{/each}


