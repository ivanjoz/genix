<script lang="ts">
  // Reusable filter text input with leading icon and per-instance debounce.
  let {
    css = '',
    placeholder = '',
    throttle: throttleMs = 150,
    icon = 'icon-filter',
    value = $bindable(''),
  }: {
    css?: string
    placeholder?: string
    throttle?: number
    icon?: string
    value?: string
  } = $props()

  // Local timer so multiple instances don't collide on a shared throttle
  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  const handleKeyUp = (ev: KeyboardEvent) => {
    ev.stopPropagation()
    const raw = (ev.target as HTMLInputElement).value || ''
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      value = raw.toLowerCase().trim()
    }, throttleMs)
  }
</script>

<div class={`filter-input-wrap relative flex items-center bg-white border border-gray-300 rounded-md ${css}`.trim()}>
  <div class="absolute inset-y-0 pl-8 flex items-center justify-center text-gray-400 pointer-events-none leading-none">
    <i class={`${icon} block leading-none`}></i>
  </div>
  <input
    class="w-full pl-34 pr-12 py-8 bg-transparent text-sm leading-none focus:outline-none placeholder:text-sm"
    autocomplete="off"
    type="text"
    {placeholder}
    onkeyup={handleKeyUp}
  />
</div>

<style>
  /* Scoped to this component by Svelte; copies the focus styling from .i-search */
  .filter-input-wrap:focus-within {
    border-color: #738dff;
    box-shadow: #5a5e6330 0 1px 6px, #738dff 0 0 1px 1px;
  }
</style>
