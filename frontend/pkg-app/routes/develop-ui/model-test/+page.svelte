<script lang="ts">
import ONNXInference from '$routes/develop-ui/model-test/ONNXInference.svelte';
</script>

<svelte:head>
  <title>ONNX Inference Demo</title>
</svelte:head>

<main>
  <ONNXInference />
</main>

<style>
  :global(body) {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    min-height: 100vh;
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  }

  main {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    padding: 20px;
  }
</style>