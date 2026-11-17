<script lang="ts" module>
  // The pack row is a slim band, shorter than the dish rows under it (TableGrid getRowHeight).
  export const PACK_HEADER_ROW_HEIGHT = 30
</script>

<script lang="ts">
  // The full-width row a pack opens in "Nuevo pedido" and "Pagar": the dishes under it form the pack.
  let { packName, packAmount }: { packName: string, packAmount: string } = $props()
</script>

<div class="w-full h-full flex items-center gap-6 px-6 bg-purple-bg text-purple-fg ff-bold text-[14px]" aria-label="A pack the dishes below form">
  <i class="icon-[fa--gift]"></i>
  <span>Pack {packName} = <span class="ff-mono">{packAmount}</span></span>
</div>
