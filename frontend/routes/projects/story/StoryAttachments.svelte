<script lang="ts">
  import FileCard from '#components/files/FileCard.svelte';
  import FileDropZone from '#components/files/FileDropZone.svelte';
  import T from '#components/misc/T.svelte';
  import { tr } from '#core/store.svelte.ts';
  import { Env } from '#core/env.ts';
  import { notifyFailure } from '@genix/ui/notify';
  import { formatFileSize, validateAttachmentFile, type IAttachment } from './story.ts';
  import { uploadAttachment } from './story.svelte.ts';

  interface Props {
    attachments: IAttachment[]
    projectID: number
    canEdit: boolean
    // Files uploading: the panel waits for them before saving.
    uploadingCount?: number
    // The comment composer draws its own icon button in its header and calls pickFiles(),
    // so it gets the bare file cards instead of the drop zone.
    showDropZone?: boolean
  }

  let { attachments = $bindable(), projectID, canEdit, uploadingCount = $bindable(0), showDropZone = true }: Props = $props()

  let fileInput = $state<HTMLInputElement>()

  export const pickFiles = () => fileInput?.click()

  // Each file goes straight to storage; a failed one is reported and the others go on.
  // Exported so the comment composer uploads the files dropped on its text box.
  export const uploadFiles = async (files: File[] | FileList | null) => {
    for (const file of files || []) {
      const fileError = validateAttachmentFile(file, attachments.length + uploadingCount)
      if (fileError) {
        notifyFailure(`${file.name}: ${tr(fileError)}`)
        continue
      }
      uploadingCount++
      try {
        const attachment = await uploadAttachment(projectID, file)
        attachments = [...attachments, attachment]
      } catch (uploadError) {
        notifyFailure(tr(String((uploadError as Error)?.message || uploadError)))
      }
      uploadingCount--
    }
  }
</script>

{#snippet attachmentCards()}
  {#if attachments.length > 0}
    <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
      {#each attachments as attachment (attachment.Key)}
        <FileCard name={attachment.Name} detail={formatFileSize(attachment.Size)} href={Env.makeCDNRoute(attachment.Key)}
          onRemove={canEdit ? () => { attachments = attachments.filter((kept) => kept.Key !== attachment.Key) } : undefined} />
      {/each}
    </div>
  {/if}
{/snippet}

{#if canEdit && showDropZone}
  <FileDropZone multiple onFiles={uploadFiles} {uploadingCount}
    label="Select or drop a file (up to 4 MB each)|Seleccione o arrastre un archivo (hasta 4 MB cada uno)">
    {@render attachmentCards()}
  </FileDropZone>
{:else}
  <!-- The comment composer opens the picker through pickFiles(). -->
  {#if canEdit}
    <!-- Cleared at once so picking the same file again still fires a change. -->
    <input bind:this={fileInput} type="file" multiple class="hidden"
      onchange={(ev) => { const files = Array.from(ev.currentTarget.files || []); ev.currentTarget.value = ''; void uploadFiles(files) }} />
  {/if}
  <div class="flex flex-col gap-6">
    {@render attachmentCards()}
    {#if uploadingCount > 0}
      <span class="text-[14px] text-fg-muted"><T text="Uploading...|Subiendo..." /> ({uploadingCount})</span>
    {/if}
    {#if !canEdit && attachments.length === 0}
      <span class="text-[14px] text-fg-muted"><T text="No attached files.|Sin archivos adjuntos." /></span>
    {/if}
  </div>
{/if}
