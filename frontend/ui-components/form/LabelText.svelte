<script lang="ts">
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
  <div class="text-gray-500 text-sm leading-[1.15]">{label}</div>
  <div class={contentCss}>{text}</div>
</div>
