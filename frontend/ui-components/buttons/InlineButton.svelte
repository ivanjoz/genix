<script lang="ts">
  type InlineButtonMode = 'default' | 'checked';
  type InlineButtonColor = 'blue' | 'green';

  let {
    label,
    mode = 'default',
    color = 'green',
    css = '',
  }: {
    label: string;
    mode?: InlineButtonMode;
    color?: InlineButtonColor;
    css?: string;
  } = $props();

  const baseCss = 'inline-flex h-22 min-w-28 items-center justify-center rounded-[3px] border text-10 ff-bold';
  const colorCssByName: Record<InlineButtonColor, Record<InlineButtonMode, string>> = {
    blue: {
      default: 'border-red-300 bg-red-100 text-red-800',
      // Keep the extra padding only in checked mode so the corner icon never overlaps the label.
      checked: 'relative border-blue-300 bg-blue-100 text-blue-800 pb-2 pr-5',
    },
    green: {
      default: 'border-red-300 bg-red-100 text-red-800',
      checked: 'relative border-green-300 bg-green-100 text-green-800 pb-2 pr-5',
    },
  };
</script>

<span class={`${baseCss} ${colorCssByName[color][mode]} ${css}`.trim()}>
  {label}
  {#if mode === 'checked'}
    <i class="icon-ok absolute -bottom-6 -right-6 text-12 leading-none"></i>
  {/if}
</span>
