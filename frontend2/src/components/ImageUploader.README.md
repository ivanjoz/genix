# ImageUploader Component

A Svelte 5 component for uploading and managing images with built-in image resizing, preview, and progress tracking.

## Migration Notes

This component has been migrated from the Solid.js version located at:
`/home/ivanjoz/projects/genix/frontend/src/components/Uploaders.tsx`

### Key Changes:
- Converted from Solid.js to Svelte 5 with runes
- Uses `$state`, `$effect`, and `$props` runes instead of createSignal/createEffect
- Uses native event handlers (onclick, onblur, onchange) instead of JSX event handlers
- Implemented XMLHttpRequest for upload progress tracking (instead of axios)
- Updated imports to match frontend2 project structure

## Features

- **Image Upload**: Supports PNG, JPEG, and WebP formats
- **Automatic Resizing**: Resizes images to a maximum area while maintaining aspect ratio
- **WebP Conversion**: Converts images to WebP format for optimal file size
- **Upload Progress**: Real-time progress tracking during upload
- **Image Preview**: Shows preview with support for multiple image formats (AVIF, WebP)
- **Description Field**: Optional description/name field for uploaded images
- **Delete Functionality**: Ability to delete/clear images
- **S3 Integration**: Supports S3 URLs for image storage

## Usage

### Basic Example

```svelte
<script lang="ts">
import ImageUploader from '$components/ImageUploader.svelte';

function handleUploaded(imagePath: string, description?: string) {
  console.log('Image uploaded:', imagePath, description);
}
</script>

<ImageUploader 
  onUploaded={handleUploaded}
/>
```

### With Existing Image

```svelte
<ImageUploader 
  src="image-name-x2"
  types={['webp', 'avif']}
  onUploaded={handleUploaded}
/>
```

### Custom Styling

```svelte
<ImageUploader 
  cardStyle="width: 15rem; height: 15rem;"
  cardCss="custom-class"
  onUploaded={handleUploaded}
/>
```

### With Custom API Endpoint

```svelte
<ImageUploader 
  saveAPI="custom/upload/endpoint"
  onUploaded={handleUploaded}
/>
```

### With Delete Handler

```svelte
<ImageUploader 
  src="existing-image"
  types={['webp']}
  onDelete={(src) => {
    console.log('Delete image:', src);
    // Handle deletion logic
  }}
/>
```

### Hide Upload Button (Auto-upload)

```svelte
<ImageUploader 
  hideUploadButton={true}
  hideFormUseMessage="Image will be uploaded automatically"
  onUploaded={handleUploaded}
/>
```

## Props

### IImageUploaderProps

| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `src` | `string` | `""` | Source URL of existing image |
| `types` | `string[]` | `[]` | Image format types (e.g., ['webp', 'avif']) |
| `saveAPI` | `string` | `"images"` | API endpoint for image upload |
| `refreshIndexDBCache` | `string` | `""` | IndexDB cache key to refresh after upload |
| `onUploaded` | `(imagePath: string, description?: string) => void` | `undefined` | Callback when image is successfully uploaded |
| `setDataToSend` | `(data: any) => void` | `undefined` | Callback to modify data before sending |
| `clearOnUpload` | `boolean` | `false` | Clear image after successful upload |
| `description` | `string` | `""` | Initial description for the image |
| `cardStyle` | `string` | `""` | Inline CSS styles for the card container |
| `onDelete` | `(src: string) => void` | `undefined` | Callback when delete button is clicked |
| `cardCss` | `string` | `""` | Additional CSS classes for the card |
| `hideFormUseMessage` | `string` | `""` | Custom message to show instead of description field |
| `hideUploadButton` | `boolean` | `false` | Hide the upload button |
| `id` | `number` | `undefined` | Custom ID for the uploader instance |

## Image Data Structure

### Upload Data Format

```typescript
interface ImageData {
  Content: string;      // Base64 encoded image data
  Folder?: string;      // Folder path (default: "img-uploads")
  Description?: string; // Image description/name
}
```

### Upload Response Format

```typescript
interface IImageResult {
  id: number;           // Uploader instance ID
  imageName: string;    // Uploaded image filename/path
  description?: string; // Image description
}
```

## Advanced Features

### Batch Upload

The component exports `imagesToUpload` Map and `uploadCurrentImages()` function for batch uploading multiple images:

```svelte
<script lang="ts">
import ImageUploader, { uploadCurrentImages } from '$components/ImageUploader.svelte';

async function handleBatchUpload() {
  const results = await uploadCurrentImages();
  console.log('All images uploaded:', results);
}
</script>

<ImageUploader id={1} hideUploadButton={true} />
<ImageUploader id={2} hideUploadButton={true} />
<ImageUploader id={3} hideUploadButton={true} />

<button onclick={handleBatchUpload}>
  Upload All Images
</button>
```

### Image Resizing Configuration

The component automatically resizes images to a maximum area of 1.2 megapixels with 89% quality. To modify these settings, edit the `resizeImageCanvasWebp` call in the `onFileChange` function:

```typescript
const imageB64 = await resizeImageCanvasWebp({
  source: imageFile,
  size: 1.2,     // Max area in megapixels
  quality: 0.89  // WebP quality (0-1)
});
```

## Styling

The component uses CSS modules from `components.module.css`. Key CSS classes:

- `.card_image_1` - Main card container
- `.card_input` - Card in input state
- `.card_image_layer` - Overlay layer
- `.card_image_textarea` - Description textarea
- `.card_image_btn` - Action buttons
- `.card_image_layer_loading` - Loading overlay

### Dark Mode Support

The component includes dark mode styles that activate when the body has a `dark` class.

## Dependencies

- `$lib/http` - POST function for API calls
- `../core/helpers` - Notify utility for user notifications
- `../shared/env` - Env configuration (S3_URL, apiRoute)
- `components.module.css` - Component styles

## Browser Compatibility

- Requires Canvas API support for image resizing
- Requires FileReader API for file handling
- Requires XMLHttpRequest Level 2 for upload progress
- WebP encoding support required (available in modern browsers)

## Notes

- Images are automatically converted to WebP format
- Upload progress is tracked using XMLHttpRequest
- The component manages cleanup on destroy to prevent memory leaks
- Images are stored in the `imagesToUpload` Map until uploaded or component is destroyed

