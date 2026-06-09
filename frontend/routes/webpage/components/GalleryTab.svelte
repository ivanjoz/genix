<script lang="ts">
  import T from '$components/misc/T.svelte';
  // Placeholder for the image gallery
</script>

<div class="gallery-tab">
  <div class="empty-state">
    <span class="empty-icon">🖼️</span>
    <h3><T text="Gallery|Galería" /></h3>
    <p><T text="The image gallery is coming soon.|La galería de imágenes estará disponible pronto." /></p>
    <p class="hint">
      <T text="Soon you'll be able to browse and insert images here.|Pronto podrás explorar e insertar imágenes aquí." />
    </p>
  </div>
</div>

<style>
  .gallery-tab {
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    height: 100%;
    min-height: 300px;
    padding: 40px 20px;
    text-align: center;
  }

  .empty-icon {
    font-size: 64px;
    margin-bottom: 24px;
  }

  h3 {
    font-size: 18px;
    color: #f1f5f9;
    margin-bottom: 8px;
  }

  p {
    color: #94a3b8;
    font-size: 14px;
    margin: 0;
  }

  .hint {
    color: #475569;
    font-size: 12px;
    margin-top: 12px;
  }
</style>
