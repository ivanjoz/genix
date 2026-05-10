<script lang="ts">
  import { Env } from '$core/env';
  import { Agent } from '$core/agent/registry';

  type ButtonColor = 'blue' | 'green' | 'red' | 'orange' | 'yellow' | 'purple';

  interface Props {
    icon?: string;
    onClick?: (ev: MouseEvent) => void;
    name?: string;
    label?: string;
    color?: ButtonColor;
    useCircle?: boolean;
    hideNameOnMobile?: boolean;
    css?: string;
    disabled?: boolean;
    // Semantic role for the agent (e.g. "save" | "delete" | "close" | custom).
    role?: string;
  }

  let {
    icon,
    onClick,
    name,
    label,
    color,
    useCircle = false,
    hideNameOnMobile = false,
    css = '',
    disabled = false,
    role,
  }: Props = $props();

  // Single source for the click action so the Agent and the DOM trigger the same handler.
  const triggerClick = (ev?: MouseEvent) => {
    if (ev) ev.stopPropagation();
    onClick?.(ev as MouseEvent);
  };

  const componentID = Env.getComponentID();

  $effect(() => {
    return Agent.register({
      id: componentID,
      type: 'Button',
      label: label || name || '',
      click: () => { triggerClick(); },
    });
  });
</script>

<button
  data-id="Button:{componentID}"
  aria-label={label}
  data-value={role}
  class={`bx-${color}${useCircle ? ' round' : ''} ${css}`.trim()}
  {disabled}
  onclick={triggerClick}
>
  {#if icon}<i class={icon}></i>{/if}
  {#if name}<span class={hideNameOnMobile ? 'hidden md:block' : ''}>{name}</span>{/if}
</button>
