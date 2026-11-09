<script lang="ts">
  import Card from '#components/cards/Card.svelte';
  import FileCard from '#components/files/FileCard.svelte';
  import Badge from '#components/misc/Badge.svelte';
  import T from '#components/misc/T.svelte';
  import { Env } from '#core/env.ts';
  import { tr } from '#core/store.svelte.ts';
  import { storyCode, storyType, type IStory } from '../project.ts';
  import { formatFileSize } from '#libs/helpers.ts';
  import type { IStoryDocuments } from './documents.ts';

  interface Props {
    storyDocumentGroups: IStoryDocuments[]
    onOpenStory: (story: IStory) => void
  }

  const { storyDocumentGroups, onOpenStory }: Props = $props()
</script>

<section class="flex flex-col gap-12">
  <h2 class="ff-bold text-fg-soft"><T text="Files in the stories|Archivos en las historias" /></h2>
  {#each storyDocumentGroups as group (group.story.ID)}
    <div class="flex flex-col gap-8 rounded-lg border border-line bg-surface p-10">
      <!-- The story's header opens it: its files are added and removed there. -->
      <Card label={`Open ${storyCode(group.story)}|Abrir ${storyCode(group.story)}`} onClick={() => onOpenStory(group.story)}
        css="flex items-center gap-8 rounded-[6px] px-6 py-4 cursor-pointer hover:bg-surface-muted">
        <i class="{storyType(group.story.Type).icon} {storyType(group.story.Type).iconCss} shrink-0"></i>
        <span class="text-fg-muted shrink-0">{storyCode(group.story)}</span>
        <span class="ff-bold grow truncate">{group.story.Title}</span>
        <Badge text={group.sprintLabel} css="bg-surface-muted text-fg-soft shrink-0" />
      </Card>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
        {#each group.documents as document (document.Key)}
          <FileCard name={document.Name} href={Env.makeCDNRoute(document.Key)}
            detail={document.ActivityID ? `${formatFileSize(document.Size)} · ${tr('In a comment|En un comentario')}` : formatFileSize(document.Size)} />
        {/each}
      </div>
    </div>
  {:else}
    <div class="text-[14px] text-fg-muted py-8"><T text="No story has attached files.|Ninguna historia tiene archivos adjuntos." /></div>
  {/each}
</section>
