<script lang="ts">
  interface Props {
    css?: string;
    children?: import('svelte').Snippet;
  }

  let {
    css, children,
  }: Props = $props();
</script>

<div class="layer-static {css}"> 
  {@render children?.()}
</div>

<style>
  .layer-static {
    height: calc(100vh - var(--header-height));
    position: fixed;
    top: var(--header-height);
    right: 0;
  }
</style>