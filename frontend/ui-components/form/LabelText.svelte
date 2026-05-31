<script lang="ts">
  import T from '$components/misc/T.svelte'

  interface LabelTextProps {
    label: string
    text: string | number
    css?: string
    contentCss?: string
  }

  let {
    label,
    text,
    css = '',
    contentCss = '',
  }: LabelTextProps = $props()
</script>

<div class={css}>
  <div class="text-gray-500 text-sm leading-[1.15]"><T text={label} /></div>
  <div class={contentCss}>{text}</div>
</div>
