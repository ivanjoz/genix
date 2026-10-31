<script lang="ts">
  import FileCard from '#components/files/FileCard.svelte';
  import FileDropZone from '#components/files/FileDropZone.svelte';
  import T from '#components/misc/T.svelte';
  import { Env } from '#core/env.ts';
  import { tr } from '#core/store.svelte.ts';
  import { formatTime } from '#libs/helpers.ts';
  import { confirmWarn, notifyFailure } from '@genix/ui/notify';
  import { formatFileSize, validateAttachmentFile } from '../story/story.ts';
  import type { IDocument } from './documents.ts';
  import type { ProjectDocumentsService } from './documents.svelte.ts';

  interface Props {
    documentsService: ProjectDocumentsService
    documents: IDocument[]
    canEdit: boolean
  }

  const { documentsService, documents, canEdit }: Props = $props()

  let uploadingCount = $state(0)

  // Each file is uploaded and saved on its own; a failed one is reported and the others go on.
  const uploadFiles = async (files: File[]) => {
    for (const file of files) {
      const fileError = validateAttachmentFile(file, 0)
      if (fileError) {
        notifyFailure(`${file.name}: ${tr(fileError)}`)
        continue
      }
      uploadingCount++
      try {
        await documentsService.addDocument(file)
      } catch (uploadError) {
        // genix-ui already notified a failed POST; a failed upload to storage throws its own message.
        if (uploadError instanceof Error) { notifyFailure(tr(uploadError.message)) }
      }
      uploadingCount--
    }
  }

  const confirmDeleteDocument = async (document: IDocument) => {
    const isConfirmed = await confirmWarn({
      title: tr('Delete document|Eliminar documento'),
      message: tr(`Delete "${document.Name}"? The file is deleted too.|¿Eliminar "${document.Name}"? El archivo también se elimina.`),
    })
    if (!isConfirmed) { return }
    try {
      await documentsService.deleteDocument(document)
    } catch {
      // genix-ui already notified the backend's error message.
    }
  }
</script>

<section class="flex flex-col gap-8">
  <h2 class="ff-bold text-fg-soft"><T text="Project documents|Documentos del proyecto" /></h2>

  {#if canEdit}
    <FileDropZone multiple onFiles={uploadFiles} {uploadingCount}
      label="Select or drop files (up to 4 MB each)|Seleccione o arrastre archivos (hasta 4 MB cada uno)" />
  {/if}

  {#each documents as document (document.Key)}
    <FileCard name={document.Name} detail={`${formatFileSize(document.Size)} · ${formatTime(document.UploadedAt || 0, 'd/m/Y h:n')}`}
      href={Env.makeCDNRoute(document.Key)} onRemove={canEdit ? () => { void confirmDeleteDocument(document) } : undefined} />
  {:else}
    <span class="text-[14px] text-fg-muted py-4"><T text="The project has no documents yet.|El proyecto aún no tiene documentos." /></span>
  {/each}
</section>
