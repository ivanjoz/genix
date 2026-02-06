<script lang="ts" module>
import type { SectionSchema } from '$ecommerce/renderer/section-types';

  export const schema: SectionSchema = {
    name: 'Feature List Simple',
    description: 'A clean list of features or benefits.',
    category: 'features',
    content: [
      'title',
      'textLines'
    ],
    css: ['container', 'title', 'list', 'item']
  };
</script>

<script lang="ts">
import type { StandardContent } from '$ecommerce/renderer/section-types';

  interface Props {
    content: StandardContent;
    css: Record<string, string>;
  }

  let { content, css }: Props = $props();
</script>

<section class="py-16 px-6 bg-slate-50 {css.container || ''}">
  {#if content.title}
    <h2 class="text-3xl font-bold mb-12 text-center text-slate-900 {css.title || ''}">{content.title}</h2>
  {/if}

  <div class="max-w-3xl mx-auto space-y-4 {css.list || ''}">
    {#each content.textLines || [] as line}
      <div class="flex items-start gap-4 p-4 bg-white rounded-xl shadow-sm border border-slate-100 {css.item || ''}">
        <div class="mt-1 shrink-0 w-6 h-6 bg-blue-100 text-blue-600 rounded-full flex items-center justify-center">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
            <polyline points="20 6 9 17 4 12"></polyline>
          </svg>
        </div>
        <div>
          <span class="text-slate-700 font-medium">{line.text}</span>
        </div>
      </div>
    {/each}
  </div>
</section>
