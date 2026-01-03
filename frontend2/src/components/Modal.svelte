<script lang="ts">
import { untrack } from 'svelte';
import Portal from './popover2/Portal.svelte';
    import { closeModal, openModals } from '$core/store.svelte';

interface Props {
  children?: import('svelte').Snippet;
  id: number;
  title: string | import('svelte').Snippet;
  css?: string;
  isEdit?: boolean;
  onSave?: () => void;
  onDelete?: () => void;
  onClose?: () => void;
  bodyCss?: string
  headCss?: string
  size: 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9
  saveIcon?: string
  saveButtonLabel?: string
}

let { 
  children, 
  id, 
  title, 
  css = "", 
  headCss,
  bodyCss,
  isEdit = false, 
  onSave, 
  onDelete, 
  onClose,
  size,
  saveIcon,
  saveButtonLabel
}: Props = $props();

// Local state for this modal instance
let isOpen = $state(false);
let modalDiv: HTMLDivElement | undefined = $state();

// Watch for changes in openModals array
$effect(() => {
  const isThisModalOpen = openModals.includes(id);
  if (isOpen === isThisModalOpen){ return };

  if (isThisModalOpen) {
    isOpen = true;
    // Use untrack to avoid creating dependencies on modalDiv
    untrack(() => {
      setTimeout(() => {
        if (modalDiv) {
          modalDiv.classList.add("show");
        }
      }, 0);
    });
  } else {
    untrack(() => {
      modalDiv?.classList?.remove("show");
      setTimeout(() => {
        isOpen = false;
      }, 300);
    });
  }
});

function handleClose(ev: MouseEvent) {
  ev.stopPropagation();
  if (onClose) onClose();
  closeModal(id);
}

function handleDelete(ev: MouseEvent) {
  if (onDelete) {
    onDelete();
    ev.stopPropagation();
  }
}

function handleSave(ev: MouseEvent) {
  if (onSave) {
    onSave();
    ev.stopPropagation();
  }
}

// Helper to check if title is a snippet
function isSnippet(value: any): value is import('svelte').Snippet {
  return typeof value === 'function';
}

const modalSizesMap = new Map([
  [1, "w-600 max-w-[64vw]"],
  [2, "w-650 max-w-[66vw]"],
  [3, "w-700 max-w-[68vw]"],
  [4, "w-750 max-w-[72vw]"],
  [5, "w-800 max-w-[75vw]"],
  [6, "w-850 max-w-[78vw]"],
  [7, "w-900 max-w-[82vw]"],
  [8, "w-950 max-w-[84vw]"],
  [9, "w-1000 max-w-[88vw]"],
])

const saveLabel = $derived.by(() => {
  if(saveButtonLabel){ return  saveButtonLabel }
  return isEdit ? "Actualizar" : "Guardar"
})

</script>

{#if isOpen}
  <Portal>
    <div class="_1 fixed top-0 left-0 flex items-center justify-center" bind:this={modalDiv}>
      <div class="_2 pt-50 min-h-460 flex flex-col relative {css} {modalSizesMap.get(size)}">
        <div class="_3 h-50 py-0 px-12 flex absolute w-full top-0 left-0 items-center justify-between mb-auto {headCss}">
          <div class="flex items-center ff-bold h2">
            {#if isSnippet(title)}
              {@render title()}
            {:else}
              {title}
            {/if}
          </div>
          <div class="flex items-center">
            {#if onDelete}
              <button class="bx-red mr-8 lh-10" onclick={handleDelete} aria-label="Eliminar">
                <i class="icon-trash"></i>
              </button>
            {/if}
            {#if onSave}
              <button class="bx-blue mr-8 lh-10" onclick={handleSave} aria-label={saveLabel}>
                <i class={saveIcon || "icon-floppy"}></i>
                <span>{saveLabel}</span>
              </button>
            {/if}
            <button class="bx-yellow h3 lh-10" onclick={handleClose} aria-label="Cerrar">
              <i class="icon-cancel"></i>
            </button>
          </div>
        </div>
        <div class="w-full grow py-6 px-10 {bodyCss}">
          {@render children?.()}
        </div>
      </div>
    </div>
  </Portal>
{/if}

<style>
  ._1 { /* background */
    width: 100vw;
    height: 100vh;
    position: fixed;
    background-color: rgba(0, 0, 0, 0.5);
    z-index: 195;
    opacity: 0;
    transition: opacity 0.3s ease;
  }

  ._1:global(.show) {
    opacity: 1;
  }

  ._2 { /* body */
    background-color: var(--white-6);
    transform: translateY(-80px);
    background-color: white;
    opacity: 0;
    transition: transform 0.3s ease, opacity 0.3s ease;
    border-radius: 0.5rem;
    box-shadow: 0 11px 15px -7px rgba(0, 0, 0, 0.2), 
                0px 24px 38px 3px rgba(0, 0, 0, 0.14),
                0px 9px 46px 8px rgba(0, 0, 0, 0.12);
  }

  ._1:global(.show) > ._2 {
    transform: translateY(0px);
    opacity: 1;
  }

  ._3 { /* Title */
    background-color: #f2f2f2;
    border-bottom: 1px solid #0000001a;
    border-radius: 7px 7px 0 0;
  }

  /* Mobile responsive */
  @media (max-width: 768px) {
    ._2 {
      max-width: 96vw;
      width: 96vw;
      padding-left: 0.4rem;
      padding-right: 0.4rem;
    }
    
    :global(.modal-title .name) {
      padding-left: 0;
      padding-top: 2px;
      font-size: 1.1rem;
      overflow: hidden;
    }
  }
</style>

