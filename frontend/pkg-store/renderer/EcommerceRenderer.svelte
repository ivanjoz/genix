<script lang="ts" module>
  import type { IProducto } from '$services/services/productos.svelte';

  export interface ComponentVariable {
  	key: string, // Example: "__v1__"
   	defaultValue: string, // Example: 80
   	type: string, // Example: "w"
    units?: ("px"|"pc"|"rem")[] // Example: "pc". "px" is default
  }
  
  export interface ITextLine {
  	text: string, css: string
  }
  
	export interface ComponentAST {
    id?: number | string
    css?: string // tailwind css rules
    description?: string
    tagName?: string // Or the tag of the component
    onClick?: (id: number | string) => void
    children?: ComponentAST[]
    text?: string
    textLines?: ITextLine[]
    backgroudImage?: string
    variables: ComponentVariable[]
    // Ecommerce Properties
    productosIDs?: number[]
    productos?: IProducto[] // Obtiene según los productos IDs
    categoriaID?: number
    marcaID?: number
  }

</script>

<script lang="ts">
  import ProductCard from '$store/components/ProductCard.svelte';
  import ProductCardHorizonal from '$store/components/ProductCardHorizonal.svelte';

  const {
    elements
  }: { elements: ComponentAST | ComponentAST[] } = $props()

  const handleClick = (element: ComponentAST) => {
    if (element.onClick) {
      element.onClick(element.id || 0)
    }
  }

</script>

{#snippet renderElement(element: ComponentAST)}
  {#if element.tagName === 'ProductCard'}
	  {#each element.productos as producto}
			<ProductCard productoID={producto.ID} css={element.css}/>
	  {/each}
	{:else if element.tagName === 'ProductCardHorizonal'}
	  {#each element.productos as producto}
			<ProductCardHorizonal producto={producto} css={element.css}/>
	  {/each}
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
