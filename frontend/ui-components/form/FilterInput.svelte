<script lang="ts">
  // Reusable filter text input with leading icon and per-instance debounce.
  import { Env } from '$core/env'
  import { Agent } from '$components/agent/registry'
  import { tr } from '$core/store.svelte'

  let {
    css = '',
    placeholder = '',
    throttle: throttleMs = 150,
    icon = 'icon-[fa--filter]',
    label = '',
    value = $bindable(''),
  }: {
    css?: string
    placeholder?: string
    throttle?: number
    icon?: string
    label?: string
    value?: string
  } = $props()

  // Local timer so multiple instances don't collide on a shared throttle
  let debounceTimer: ReturnType<typeof setTimeout> | null = null
  // Mirrors what the user sees in the box; lets the agent write into it.
  let inputValue = $state(value)

  // Normalise the way the consumer expects: lowercase + trim.
  const commitValue = (raw: string) => {
    value = raw.toLowerCase().trim()
  }

  const handleKeyUp = (ev: KeyboardEvent) => {
    ev.stopPropagation()
    const raw = (ev.target as HTMLInputElement).value || ''
    inputValue = raw
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => commitValue(raw), throttleMs)
  }

  const componentID = Env.getComponentID()

  $effect(() => {
    return Agent.register({
      id: componentID,
      type: 'FilterInput',
      label: label || placeholder || '',
      // Agent path bypasses the debounce — there's no keystroke stream to coalesce.
      setValue: (next: string | number) => {
        const raw = String(next ?? '')
        inputValue = raw
        if (debounceTimer) clearTimeout(debounceTimer)
        commitValue(raw)
      },
    })
  })
</script>

<div
  data-id="FilterInput:{componentID}"
  data-value={inputValue}
  data-label={label || placeholder || ''}
  data-type="text"
  class={`filter-input-wrap relative flex items-center bg-white border border-gray-300 rounded-md ${css}`.trim()}
>
  <div class="absolute inset-y-0 pl-8 flex items-center justify-center text-gray-400 pointer-events-none leading-none">
    <i class={`${icon} block leading-none`}></i>
  </div>
  <input
    class="w-full pl-34 pr-12 py-7 bg-transparent text-sm leading-none focus:outline-none placeholder:text-sm"
    autocomplete="off"
    type="text"
    aria-label={tr(label || placeholder || undefined)}
    placeholder={tr(placeholder)}
    bind:value={inputValue}
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
