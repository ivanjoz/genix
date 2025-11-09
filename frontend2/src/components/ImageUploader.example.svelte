<script lang="ts">
/**
 * ImageUploader Component Example
 * 
 * This file demonstrates various use cases of the ImageUploader component.
 */

import ImageUploader, { uploadCurrentImages, type IImageResult } from './ImageUploader.svelte';

// Example 1: Basic Usage
let uploadedImage1 = $state('');

function handleUpload1(imagePath: string, description?: string) {
  console.log('Image 1 uploaded:', imagePath, description);
  uploadedImage1 = imagePath;
}

// Example 2: With existing image
let existingImageSrc = $state('example-image-x2');
let existingImageTypes = ['webp', 'avif'];

function handleUpload2(imagePath: string, description?: string) {
  console.log('Image 2 uploaded:', imagePath, description);
  existingImageSrc = imagePath + '-x2';
}

// Example 3: Custom styling
function handleUpload3(imagePath: string, description?: string) {
  console.log('Image 3 uploaded:', imagePath, description);
}

// Example 4: With delete handler
let deletableImage = $state('sample-image-x2');

function handleDelete(src: string) {
  console.log('Delete requested for:', src);
  // Call your API to delete the image
  deletableImage = '';
}

function handleUpload4(imagePath: string, description?: string) {
  console.log('Image 4 uploaded:', imagePath, description);
  deletableImage = imagePath + '-x2';
}

// Example 5: Batch upload
let batchImages = $state<string[]>([]);
let isUploading = $state(false);

async function handleBatchUpload() {
  isUploading = true;
  try {
    const results: IImageResult[] = await uploadCurrentImages();
    console.log('Batch upload results:', results);
    batchImages = results.map(r => r.imageName);
  } catch (error) {
    console.error('Batch upload failed:', error);
  } finally {
    isUploading = false;
  }
}

// Example 6: With custom API endpoint
function handleUpload6(imagePath: string, description?: string) {
  console.log('Image 6 uploaded to custom endpoint:', imagePath, description);
}
</script>

<div class="examples-container">
  <h1>ImageUploader Component Examples</h1>

  <!-- Example 1: Basic Usage -->
  <section class="example">
    <h2>1. Basic Usage</h2>
    <p>Simple image uploader with default settings</p>
    <ImageUploader onUploaded={handleUpload1} />
    {#if uploadedImage1}
      <p class="result">Uploaded: {uploadedImage1}</p>
    {/if}
  </section>

  <!-- Example 2: With Existing Image -->
  <section class="example">
    <h2>2. With Existing Image</h2>
    <p>Displays an existing image with the ability to replace it</p>
    <ImageUploader 
      src={existingImageSrc}
      types={existingImageTypes}
      onUploaded={handleUpload2}
    />
  </section>

  <!-- Example 3: Custom Styling -->
  <section class="example">
    <h2>3. Custom Styling</h2>
    <p>Custom size and CSS classes</p>
    <ImageUploader 
      cardStyle="width: 15rem; height: 15rem;"
      cardCss="border-red"
      onUploaded={handleUpload3}
    />
  </section>

  <!-- Example 4: With Delete Handler -->
  <section class="example">
    <h2>4. With Delete Handler</h2>
    <p>Allows deleting existing images</p>
    {#if deletableImage}
      <ImageUploader 
        src={deletableImage}
        types={['webp']}
        onDelete={handleDelete}
        onUploaded={handleUpload4}
      />
    {:else}
      <ImageUploader onUploaded={handleUpload4} />
    {/if}
  </section>

  <!-- Example 5: Batch Upload -->
  <section class="example">
    <h2>5. Batch Upload</h2>
    <p>Upload multiple images at once</p>
    <div class="batch-grid">
      <ImageUploader id={101} hideUploadButton={true} />
      <ImageUploader id={102} hideUploadButton={true} />
      <ImageUploader id={103} hideUploadButton={true} />
    </div>
    <button 
      class="batch-upload-btn"
      onclick={handleBatchUpload}
      disabled={isUploading}
    >
      {isUploading ? 'Uploading...' : 'Upload All Images'}
    </button>
    {#if batchImages.length > 0}
      <div class="result">
        <p>Uploaded images:</p>
        <ul>
          {#each batchImages as img}
            <li>{img}</li>
          {/each}
        </ul>
      </div>
    {/if}
  </section>

  <!-- Example 6: Custom API Endpoint -->
  <section class="example">
    <h2>6. Custom API Endpoint</h2>
    <p>Upload to a custom endpoint</p>
    <ImageUploader 
      saveAPI="gallery/upload"
      onUploaded={handleUpload6}
    />
  </section>

  <!-- Example 7: Auto-upload with Message -->
  <section class="example">
    <h2>7. Auto-upload with Custom Message</h2>
    <p>Hides upload button and shows custom message</p>
    <ImageUploader 
      hideUploadButton={true}
      hideFormUseMessage="Image will be uploaded with the form"
      onUploaded={(path) => console.log('Auto-uploaded:', path)}
    />
  </section>

  <!-- Example 8: Clear on Upload -->
  <section class="example">
    <h2>8. Clear on Upload</h2>
    <p>Clears the image after successful upload</p>
    <ImageUploader 
      clearOnUpload={true}
      onUploaded={(path) => console.log('Uploaded and cleared:', path)}
    />
  </section>
</div>

<style>
  .examples-container {
    padding: 2rem;
    max-width: 1200px;
    margin: 0 auto;
  }

  h1 {
    margin-bottom: 2rem;
    color: #333;
  }

  .example {
    margin-bottom: 3rem;
    padding: 1.5rem;
    border: 1px solid #ddd;
    border-radius: 8px;
    background-color: #f9f9f9;
  }

  .example h2 {
    margin-top: 0;
    color: #555;
  }

  .example p {
    color: #666;
    margin-bottom: 1rem;
  }

  .result {
    margin-top: 1rem;
    padding: 0.5rem;
    background-color: #e8f5e9;
    border-radius: 4px;
    color: #2e7d32;
  }

  .batch-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(10rem, 1fr));
    gap: 1rem;
    margin-bottom: 1rem;
  }

  .batch-upload-btn {
    padding: 0.75rem 1.5rem;
    background-color: #1976d2;
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 1rem;
  }

  .batch-upload-btn:hover:not(:disabled) {
    background-color: #1565c0;
  }

  .batch-upload-btn:disabled {
    background-color: #bdbdbd;
    cursor: not-allowed;
  }

  .result ul {
    margin-top: 0.5rem;
    padding-left: 1.5rem;
  }

  .result li {
    margin-bottom: 0.25rem;
  }

  :global(.border-red) {
    border-color: #f44336 !important;
  }
</style>

