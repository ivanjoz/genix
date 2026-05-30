<script lang="ts">
  import { Core, tr } from '$core/store.svelte'

  interface TProps {
    text: string
    css?: string
  }

  let { text = '', css }: TProps = $props()

  // text can contain a "|" pipe separator: [0] = English, [1] = Spanish.
  // Reading Core.languaje keeps this reactive to language changes.
  const content = $derived(tr(text, Core.languaje))
</script>

<span class={css}>{content}</span>
