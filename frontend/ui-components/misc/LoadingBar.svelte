<script lang="ts">
  let {
    css = '',
    label = '',
  }: {
    css?: string;
    label?: string;
  } = $props();
</script>

<div class={`loading-bar-wrapper ${css}`.trim()}>
  {#if label}
    <span class="loading-bar-label ff-semibold">{label}</span>
  {/if}
  <span class="loader"></span>
</div>

<style>
  .loading-bar-wrapper {
    width: 100%;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .loading-bar-label {
    color: #374151;
    font-size: 13px;
    line-height: 18px;
  }

  .loader {
    width: 100%;
    height: 12px;
    display: inline-block;
    background-color: #fff;
    background-image: linear-gradient(45deg, rgba(0, 0, 0, 0.25) 25%, transparent 25%, transparent 50%, rgba(0, 0, 0, 0.25) 50%, rgba(0, 0, 0, 0.25) 75%, transparent 75%, transparent);
    font-size: 30px;
    background-size: 1em 1em;
    box-sizing: border-box;
    animation: barStripe 1s linear infinite;
  }

  @keyframes barStripe {
    0% {
      background-position: 1em 0;
    }
    100% {
      background-position: 0 0;
    }
  }
</style>
