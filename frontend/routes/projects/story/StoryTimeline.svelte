<script lang="ts">
  import Button from '#components/buttons/Button.svelte';
  import Input from '#components/form/Input.svelte';
  import T from '#components/misc/T.svelte';
  import { tr } from '#core/store.svelte.ts';
  import { formatTime } from '#libs/helpers.ts';
  import { hideLoading, notifyFailure, showLoading } from '@genix/ui/notify';
  import AttachmentFiles from '#domain/attachments/AttachmentFiles.svelte';
  import type { IAttachment } from '#domain/attachments/attachments.ts';
  import { ACTIVITY_KIND_COMMENT, activityDescription, isStoppedWithReason, sortActivityNewestFirst } from './story.ts';
  import { uploadAttachment, type OpenStory } from './story.svelte.ts';

  interface Props {
    openStory: OpenStory
    projectID: number
    storyID: number
    canComment: boolean
    userName: (userID: number) => string
    sprintName: (sprintID: number) => string
    onCommented: () => void
  }

  const { openStory, projectID, storyID, canComment, userName, sprintName, onCommented }: Props = $props()

  let commentForm = $state<{ Text: string, Attachments: IAttachment[] }>({ Text: '', Attachments: [] })
  let uploadingCount = $state(0)
  let commentAttachments = $state<ReturnType<typeof AttachmentFiles>>()
  const timeline = $derived(sortActivityNewestFirst(openStory.activity))

  const sendComment = async () => {
    if (uploadingCount > 0) {
      notifyFailure(tr('Wait for the files to finish uploading.|Espera a que terminen de subir los archivos.'))
      return
    }
    if (!commentForm.Text.trim() && commentForm.Attachments.length === 0) {
      notifyFailure(tr('The comment is empty.|El comentario está vacío.'))
      return
    }
    showLoading(tr('Sending comment...|Enviando comentario...'))
    try {
      await openStory.addComment(projectID, storyID, commentForm.Text, commentForm.Attachments)
      // A new object clears the fields.
      commentForm = { Text: '', Attachments: [] }
      onCommented()
    } catch {
      // genix-ui already notified the backend's error message. Keep the comment to retry.
    }
    hideLoading()
  }
</script>

<div class="flex flex-col gap-12">
  {#if canComment}
    <div class="flex flex-col gap-6 rounded border border-line bg-surface-soft p-8">
      <div class="flex items-center gap-8">
        <span class="grow min-w-0 text-fg-muted"><T text="Write a comment or drop a file|Escribe un comentario o arrastra un archivo" /></span>
        <Button icon="icon-[fa--paperclip]" label="Attach files of up to 4 MB each.|Adjunta archivos de hasta 4 MB cada uno."
          css="bn-white" onClick={() => commentAttachments?.pickFiles()} />
        <Button icon="icon-[fa--paper-plane]" color="blue" label="Sends the comment.|Envía el comentario." onClick={sendComment} />
      </div>
      <!-- Files dropped on the text box are attached to the comment. -->
      <Input saveOn={commentForm} save="Text" useTextArea={true} rows={3} useFileDragUpload={true}
        onFileDrop={(files) => commentAttachments?.uploadFiles(files)} />
      <AttachmentFiles bind:this={commentAttachments} bind:attachments={commentForm.Attachments} bind:uploadingCount
        uploadFile={(file) => uploadAttachment(projectID, file)}
        canEdit={true} showDropZone={false} />
    </div>
  {/if}

  {#if !openStory.isLoaded}
    <div class="text-fg-muted"><T text="Loading...|Cargando..." /></div>
  {:else if timeline.length === 0}
    <div class="text-fg-muted"><T text="No activity yet.|Todavía no hay actividad." /></div>
  {/if}

  {#each timeline as activity (activity.ID)}
    {#if activity.Kind === ACTIVITY_KIND_COMMENT}
      <div class="rounded border border-line bg-surface p-10">
        <div class="flex items-center gap-8 mb-4 text-[14px]">
          <span class="ff-bold">{userName(activity.AuthorID)}</span>
          <span class="text-fg-muted">{formatTime(activity.Created, 'd/m/Y h:n')}</span>
        </div>
        {#if activity.Text}
          <div class="whitespace-pre-wrap break-words">{activity.Text}</div>
        {/if}
        {#if activity.Attachments?.length}
          <div class="mt-6">
            <AttachmentFiles attachments={activity.Attachments} canEdit={false} />
          </div>
        {/if}
      </div>
    {:else}
      <div class="flex gap-8 text-[14px] text-fg-muted px-4">
        <i class="icon-[fa--history] mt-3 shrink-0 text-fg-subtle"></i>
        <div>
          <span><T text={activityDescription(activity, userName, sprintName)} /></span>
          <span class="text-fg-subtle"> · {userName(activity.AuthorID)} · {formatTime(activity.Created, 'd/m/Y h:n')}</span>
          {#if isStoppedWithReason(activity)}
            <div class="mt-2 rounded bg-red-bg text-red-fg px-8 py-4">{activity.Text}</div>
          {/if}
        </div>
      </div>
    {/if}
  {/each}
</div>
