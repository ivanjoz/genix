---
schema: 1
page_id: webpage-builder.gallery
route: /webpage-builder/gallery
title: Gallery (Galería)
status: implemented
visibility: tenant
description_en: >-
  The company's image gallery for its website and its restaurant carta. Upload PNG, JPG or WebP
  images with a name, search them by name, and delete them. The images uploaded here are the ones
  offered in the image picker of the page builder and the carta builder.
description_es: >-
  Galería de imágenes de la empresa para su sitio web y su carta del restaurante. Subir imágenes
  PNG, JPG o WebP con un nombre, buscarlas por nombre y eliminarlas. Las imágenes subidas aquí son
  las que ofrece el selector de imágenes del constructor de páginas y del constructor de la carta.
---

# Gallery (Galería)

<!-- DOC-ID: page-purpose -->
## Page purpose

Gallery (`Galería`) is the company's own image library (`galería de imágenes`, `banco de
imágenes`) for its website (`tienda web`, `página web`). Images uploaded here are offered in the
**Select image (Seleccionar imagen)** picker, tab **Gallery (Galería)**, when editing an image of a
web page section or of the restaurant carta.

This page does not place an image on a page: that is done in the page builder, opened from **Pages
(Páginas)**, or in the carta builder, **Menu (Carta)**.

<!-- DOC-ID: capability.upload-image -->
## Upload an image (Subir una imagen)

### User intention (Intención del usuario)

Add a photo, banner or logo the website or the carta can use.

### Where to find it (Dónde encontrarlo)

**Commercial (Comercial) → Store / Web (Tienda / Web) → Gallery (Galería)** at
`/webpage-builder/gallery`. The first card of the grid, **Upload Image (Subir Imagen)**: click it to
choose a file, or drop the file on it (**Suelta aquí la imagen**). Type an optional name in
**Nombre...** on the preview and confirm with the check button (**Subir imagen**); the × button
discards it.

### Required information and prerequisites (Requisitos previos)

The file must be PNG, JPG or WebP; any other format is refused with "La imagen debe ser PNG, JPG o
WebP." The name is optional.

### Business rules and rationale (Reglas y razón de negocio)

The browser converts the image to AVIF in two sizes before uploading; the gallery shows the smaller
one and the picker inserts the larger one into the page.

### Result and side effects (Resultado y efectos)

The image appears in the grid at once, with a small loader while it finishes uploading. Without a
name, its card reads **Imagen de galería**. The upload card clears for the next image.

### Limitations (Limitaciones)

- The name cannot be changed after uploading; delete and upload again.
- There are no folders or categories in the gallery.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo subo una imagen para mi página web? ¿Dónde cargo el banner de la tienda?
- ¿Qué formatos acepta? → PNG, JPG o WebP.
- Search terms: `subir imagen`, `cargar foto`, `galería`, `banner`, `logo`, `imagen de la web`.

<!-- DOC-ID: capability.search-images -->
## Search the gallery (Buscar imágenes)

The **Search gallery images (Buscar imágenes de galería)** box, **Buscar por nombre...**, filters the
grid by the name given at upload. With no match the page says **No hay imágenes con ese nombre.**;
an empty gallery says **Sube tu primera imagen a la galería.** Images without a name cannot be
found by search.

- ¿Cómo busco una imagen en la galería?

<!-- DOC-ID: capability.delete-image -->
## Delete an image (Eliminar una imagen)

### User intention (Intención del usuario)

Remove an image the company no longer wants to offer.

### Where to find it (Dónde encontrarlo)

The × button on the image (**Eliminar imagen**), then confirm **ELIMINAR IMAGEN** — "¿Eliminar la
imagen seleccionada?".

### Result and side effects (Resultado y efectos)

The image leaves this gallery and the **Galería** tab of the image picker.

### Limitations (Limitaciones)

Genix does not check whether a web page or the carta already uses the image before deleting it,
and the deletion does not edit those pages. There is no undo.

### Common questions and vocabulary (Preguntas y vocabulario)

- ¿Cómo borro una imagen de la galería? ¿Si la borro desaparece de mi página?

<!-- DOC-ID: related-pages -->
## Related pages and workflows (Páginas y procesos relacionados)

- **Pages (Páginas)** opens the page builder, where **Seleccionar imagen** offers these images
  (tab **Galería**) and the stock images (tab **Stock Imágenes**). That picker can also upload a new
  gallery image in place.
- **Menu (Carta)** uses the same picker for the carta's images.
- **Preview (Previsualizar)** shows the published store.

### FILES

```yaml
schema: 1
hash_algorithm: sha256
files:
  - path: db/modules.toml
    role: user-interface
    hash: sha256:0aa76367dcb4cca954b7696a2a424bae489a875e7b4027907fba7f0421cc0a03
    supports: [capability.upload-image]
  - path: frontend/routes/webpage-builder/gallery/+page.svelte
    role: page
    hash: sha256:62eed02e3067ae0a6a298776d5847d57f3c38f76be6cf8f696961ee18abf8ed0
    supports: [page-purpose, capability.upload-image, capability.search-images, capability.delete-image]
  - path: frontend/services/webpage/gallery.svelte.ts
    role: frontend-service
    hash: sha256:3ea481cea7d3e45f9684af6e18050308ea38057e9395a99c795251945ec203a8
    supports: [capability.upload-image, capability.delete-image]
  - path: frontend/packages/genix-ui/files/ImageUploader.svelte
    role: user-interface
    hash: sha256:096787362e28a625af56a0d25e06effea5617ebb8ccee3000696c8bdcd166feb
    supports: [capability.upload-image]
  - path: frontend/packages/genix-ui/files/Image.svelte
    role: user-interface
    hash: sha256:2d2953758439d8d56b807e2850b4e45549b86bbb31ea07f0eb0cb8dc11b9d9c8
    supports: [capability.delete-image]
  - path: frontend/domain-components/ImagePickerModal.svelte
    role: user-interface
    hash: sha256:07fb5fa152f18151d25369ffbb5b6ad488bb4f4d377229df95c1370dc5c607d4
    supports: [page-purpose, capability.upload-image, capability.delete-image, related-pages]
  - path: frontend/routes/webpage-builder/components/ImageBlockEditor.svelte
    role: user-interface
    hash: sha256:592f234237d36eaa0c5bc560826e2b4909ee9b95e1d31c2d155b42116efc6b75
    supports: [page-purpose, related-pages]
  - path: frontend/routes/dining/menu-builder/MenuBlockEditor.svelte
    role: user-interface
    hash: sha256:54eca8272ac963fd52a45bf90ab8b5e556328d887c7257cc8a11bc428e3d6421
    supports: [page-purpose, related-pages]
  - path: backend/business/gallery-image.go
    role: backend-handler
    hash: sha256:73dd2fe98c71f319671ee8bb7d51ce590e0e4020a02c14d558eae712800666c9
    supports: [capability.upload-image, capability.delete-image]
  - path: backend/business/types/cms.go
    role: data-model
    hash: sha256:c5173ad0fd3489098b90833ad64e8f9d5d88a5170005d4daee6d321c37cc251a
    supports: [capability.upload-image, capability.delete-image]
  - path: frontend/packages/genix-ui/form/FilterInput.svelte
    role: user-interface
    hash: sha256:0c46789df7d9ad8e2121515388ed96e071cad20d74b519f94d5b6f490f6b6309
    supports: [capability.search-images]
```
