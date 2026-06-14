<script lang="ts">
  import { sectionTemplates } from '$ecommerce/templates';

  interface Props {
    onSelect: (template: { id: string }) => void;
  }

  type SectionTemplate = (typeof sectionTemplates)[number];

  let { onSelect }: Props = $props();

  let searchQuery = $state('');
  let selectedCategory = $state('all');

  const categories = [
    'all',
    ...new Set(sectionTemplates.flatMap(template => template.category ? [template.category] : []))
  ];

  const filteredTemplates = $derived(() => {
    return sectionTemplates.filter(template => {
      const matchesSearch = (template.name ?? '').toLowerCase().includes(searchQuery.toLowerCase());
      const matchesCategory = selectedCategory === 'all' || template.category === selectedCategory;
      return matchesSearch && matchesCategory;
    });
  });

  function handleDragStart(event: DragEvent, template: SectionTemplate) {
    if (!event.dataTransfer) return;
    event.dataTransfer.setData('application/x-genix-template', JSON.stringify(template));
    event.dataTransfer.effectAllowed = 'copy';
    (event.currentTarget as HTMLElement).style.opacity = '0.5';
  }

  function handleDragEnd(event: DragEvent) {
    (event.currentTarget as HTMLElement).style.opacity = '1';
  }
</script>

<div class="templates-tab">
  <div class="search-bar">
    <input
      type="text"
      bind:value={searchQuery}
      placeholder="Search templates..."
      class="search-input"
      aria-label="Search section templates"
    />
  </div>

  <div class="categories-strip">
    {#each categories as category}
      <button
        class="category-btn"
        class:active={selectedCategory === category}
        onclick={() => selectedCategory = category}
        aria-label={`Filter by ${category} category`}
      >
        {category}
      </button>
    {/each}
  </div>

  <div class="templates-grid">
    {#each filteredTemplates() as template}
      <button
        class="template-card"
        onclick={() => onSelect(template)}
        draggable="true"
        ondragstart={(e) => handleDragStart(e, template)}
        ondragend={handleDragEnd}
        aria-label={`Add ${template.name} section to store`}
      >
        <div class="template-preview">
          {@html template.thumbnail ?? ''}
        </div>
        <div class="template-name">
          {(template.name ?? 'Template').replace(' (HTML)', '')}
        </div>
      </button>
    {/each}
  </div>
</div>

<style>
  .templates-tab {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  .search-input {
    width: 100%;
    padding: 10px 12px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 6px;
    color: #f1f5f9;
    font-size: 14px;
    box-sizing: border-box;
  }

  .categories-strip {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    padding-bottom: 4px;
  }

  .category-btn {
    padding: 4px 12px;
    border-radius: 20px;
    background: #1e293b;
    border: 1px solid #334155;
    color: #94a3b8;
    font-size: 12px;
    white-space: nowrap;
    cursor: pointer;
    text-transform: capitalize;
  }

  .category-btn.active {
    background: #3b82f6;
    border-color: #3b82f6;
    color: white;
  }

  .templates-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    align-items: start;
  }

  .template-card {
    display: block;
    min-width: 0;
    padding: 0;
    overflow: hidden;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .template-card:hover {
    border-color: #3b82f6;
    transform: translateY(-2px);
    background: #243146;
  }

  .template-preview {
    display: flex;
    flex-direction: column;
    justify-content: center;
    height: 100px;
    overflow: hidden;
    border-bottom: 1px solid #334155;
    pointer-events: none;
  }

  .template-name {
    padding: 9px 8px;
    overflow: hidden;
    color: #f1f5f9;
    font-size: 12px;
    font-weight: 600;
    text-align: center;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
