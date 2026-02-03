<script lang="ts" module>
  import type { SectionSchema } from '../renderer/section-types';

  export const schema: SectionSchema = {
    name: 'Hero Standard',
    description: 'A versatile hero section with support for text alignment, background images, and primary actions.',
    category: 'hero',
    content: [
      'title', 
      'subTitle', 
      'description', 
      'textLeft', 
      'textCenter', 
      'textRight', 
      'bgImage', 
      'primaryActionLabel', 
      'primaryActionHref'
    ],
    css: ['container', 'title', 'subTitle', 'description', 'contentBox', 'button']
  };
</script>

<script lang="ts">
  import type { StandardContent } from '../renderer/section-types';

  interface Props {
    content: StandardContent;
    css: Record<string, string>;
  }

  let { content, css }: Props = $props();

  // Default classes if not provided
  const containerClass = $derived(css.container || 'relative py-24 px-6 overflow-hidden bg-slate-900 text-white min-h-[60vh] flex items-center');
  const contentBoxClass = $derived(css.contentBox || 'relative z-10 max-w-4xl mx-auto w-full');
  const titleClass = $derived(css.title || 'text-5xl md:text-7xl font-black mb-6 leading-tight');
  const subTitleClass = $derived(css.subTitle || 'text-xl md:text-2xl font-medium mb-4 text-blue-400 uppercase tracking-wider');
  const descClass = $derived(css.description || 'text-lg md:text-xl text-slate-300 mb-8 max-w-2xl');
  const buttonClass = $derived(css.button || 'inline-block bg-blue-600 hover:bg-blue-700 text-white font-bold py-4 px-8 rounded-lg transition-all transform hover:scale-105');
</script>

<section class={containerClass}>
  {#if content.bgImage}
    <div class="absolute inset-0 z-0">
      <img 
        src={content.bgImage} 
        alt="" 
        class="w-full h-full object-cover opacity-40"
      />
      <div class="absolute inset-0 bg-gradient-to-t from-slate-950/80 to-transparent"></div>
    </div>
  {/if}

  <div class={contentBoxClass}>
    {#if content.subTitle}
      <p class={subTitleClass}>{content.subTitle}</p>
    {/if}
    
    {#if content.title}
      <h1 class={titleClass}>{content.title}</h1>
    {/if}

    {#if content.description}
      <p class={descClass}>{content.description}</p>
    {/if}

    <!-- Example of supporting specific text placements -->
    {#if content.textLeft || content.textCenter || content.textRight}
      <div class="grid grid-cols-1 md:grid-cols-3 gap-8 my-8">
        {#if content.textLeft}
          <div class="text-left p-4 border-l-2 border-blue-500 bg-white/5">{content.textLeft}</div>
        {/if}
        {#if content.textCenter}
          <div class="text-center p-4 border-y-2 border-blue-500 bg-white/5">{content.textCenter}</div>
        {/if}
        {#if content.textRight}
          <div class="text-right p-4 border-r-2 border-blue-500 bg-white/5">{content.textRight}</div>
        {/if}
      </div>
    {/if}

    {#if content.primaryActionLabel}
      <div class="mt-4">
        <a href={content.primaryActionHref || '#'} class={buttonClass}>
          {content.primaryActionLabel}
        </a>
      </div>
    {/if}
  </div>
</section>

<style>
  /* Local overrides if necessary, though Tailwind is preferred */
  section {
    display: flex;
    flex-direction: column;
    justify-content: center;
  }
</style>
