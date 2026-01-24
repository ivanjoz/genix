# Store Integration Notes

## Overview
This document contains notes about the Genix e-commerce store integration into the main frontend application.

## Thumbhash Implementation (TODO)

### Status
The thumbhash generator code has been added to the frontend but is not yet implemented or integrated into the build process.

### Purpose
Thumbhash is used to generate compact blur placeholders for product images, improving the user experience by showing a preview while images load.

### File Locations

#### Source Code
- **Generator Script**: `frontend/scripts/thumbhash-prebuild.js`
  - Copied from `store/prebuild.js`
  - Generates thumbhashes for all images in `static/images/`
  - Renames images based on thumbhash suffix
  - Outputs thumbhash data to `static/images/thumbhash.txt`

#### Dependencies
- **thumbhash**: `^0.1.1` (added to package.json)
- **sharp**: `^0.34.3` (added to package.json as dev dependency)

#### Runtime Files
- **Blurhash Script**: `frontend/src/lib/blurhash.js`
  - Copied from `store/src/lib/blurhash.js`
  - Used to decode thumbhash data URLs in the browser

### Implementation Steps

When ready to implement thumbhash:

1. **Update package.json scripts**
   ```json
   {
     "scripts": {
       "prebuild:store": "node scripts/thumbhash-prebuild.js",
       "build:store": "npm run prebuild:store && vite build --mode store"
     }
   }
   ```

2. **Update build process**
   - Run `prebuild:store` before building the store
   - This will generate thumbhashes for all product images
   - Images will be renamed based on their thumbhash

3. **Integrate with image upload workflow**
   - When uploading new product images, generate thumbhash
   - Store thumbhash data in product metadata
   - Use thumbhash for image previews in the store

4. **Update product service**
   - Modify `frontend/src/services/productos.svelte.ts` to include thumbhash data
   - Update `ProductCard` components to use thumbhash placeholders

5. **Test the implementation**
   - Verify thumbhash generation works correctly
   - Check that images are renamed properly
   - Ensure blurhash script loads and displays previews
   - Test with various image formats (jpg, png, webp, etc.)

### Current Limitations

- Thumbhash generation is not automated in the build process
- No integration with product image upload workflow
- Thumbhash data is not stored or used in product displays
- Images are not renamed based on thumbhash

### Future Enhancements

- [ ] Automate thumbhash generation on image upload
- [ ] Store thumbhash data in product database
- [ ] Implement lazy loading with thumbhash previews
- [ ] Add thumbhash to product API responses
- [ ] Create admin UI to regenerate thumbhashes

## Additional Notes

### Store Routes
- Store is accessible at `/store/*`
- Admin routes are at `/admin/*`
- Store routes use SSR for better SEO
- Admin routes use SPA mode

### CSS Hashing
- Store uses frontend's counter-based CSS hashing system
- Located in `frontend/plugins.js`
- Ensures consistent class names across builds

### Service Worker
- Store uses frontend's service worker for offline caching
- Located in `frontend/src/lib/sw-cache.ts`
- Caches store assets and API responses

### UI Components
- Store uses frontend's base UI components (Input, Select, etc.)
- Located in `frontend/src/components/`
- Deprecated store-specific components have been removed

## Migration Date
2024-01-XX

## Last Updated
2024-01-XX