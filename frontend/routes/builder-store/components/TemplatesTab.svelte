<script lang="ts">
  import { SectionList } from '../../../pkg-store/templates/registry';

  interface Props {
    onSelect: (template: { id: string }) => void;
  }

  let { onSelect }: Props = $props();

  let searchQuery = $state('');
  let selectedCategory = $state('all');

  const categories = ['all', ...new Set(SectionList.map(t => t.category))];

  const filteredTemplates = $derived(() => {
    return SectionList.filter(t => {
      const matchesSearch = t.name.toLowerCase().includes(searchQuery.toLowerCase());
      const matchesCategory = selectedCategory === 'all' || t.category === selectedCategory;
      return matchesSearch && matchesCategory;
    });
  });

  function handleDragStart(e: DragEvent, template: any) {
    if (!e.dataTransfer) return;
    e.dataTransfer.setData('application/x-genix-template', JSON.stringify(template));
    e.dataTransfer.effectAllowed = 'copy';
    
    const target = e.target as HTMLElement;
    target.style.opacity = '0.5';
  }

  function handleDragEnd(e: DragEvent) {
    const target = e.target as HTMLElement;
    target.style.opacity = '1';
  }
</script>

<div class="templates-tab">
  <div class="search-bar">
    <input 
      type="text" 
      bind:value={searchQuery} 
      placeholder="Search templates..."
      class="search-input"
    />
  </div>

  <div class="categories-strip">
    {#each categories as category}
      <button 
        class="category-btn" 
        class:active={selectedCategory === category}
        onclick={() => selectedCategory = category}
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
      >
        <div class="template-icon">🧩</div>
        <div class="template-info">
          <div class="template-name">{template.name}</div>
          <div class="template-tag">{template.category}</div>
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
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .template-card {
    display: flex;
    gap: 12px;
    padding: 12px;
    background: #1e293b;
    border: 1px solid #334155;
    border-radius: 8px;
    text-align: left;
    cursor: pointer;
    transition: all 0.15s ease;
  }

  .template-card:hover {
    border-color: #3b82f6;
    transform: translateY(-2px);
    background: #243146;
  }

  .template-icon {
    width: 40px;
    height: 40px;
    background: #0f172a;
    border-radius: 6px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
  }

  .template-info {
    flex: 1;
  }

  .template-name {
    font-size: 14px;
    font-weight: 600;
    color: #f1f5f9;
    margin-bottom: 2px;
  }

  .template-tag {
    display: inline-block;
    font-size: 10px;
    font-weight: 700;
    color: #3b82f6;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }
</style>