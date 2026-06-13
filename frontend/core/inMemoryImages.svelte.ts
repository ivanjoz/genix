import { SvelteMap } from 'svelte/reactivity'

// Lifecycle of an optimistic image while its bytes are converted/uploaded in the background.
export type InMemoryImageStatus = 'converting' | 'pending' | 'uploading' | 'error'

export interface InMemoryImage {
  name: string            // base CDN name "<companyID>_<imageID>" — the map key
  id: number              // imageID (autoincrement*10 + configDigit)
  folder: string          // e.g. "img-productos"
  base64: string          // shown by every renderer while the image is in flight
  description?: string
  status: InMemoryImageStatus
  progress: number        // upload progress 0..100 (-1 = none yet)
  error?: string
  // Flush step: convert (if needed) + upload the bytes. Parents call this AFTER saving their own
  // record, passing `override` (e.g. { ProductID }) when the real id wasn't known at confirm time.
  // Resolves when the upload succeeds.
  upload?: (override?: Record<string, any>) => Promise<void>
}

// Global, keyed by the final base CDN name so ANY component (product list, detail panel,
// ecommerce card, …) can resolve the same in-flight image without threading state around.
// While an entry exists the image is rendered from its base64; once the upload succeeds the
// entry is deleted and renderers fall through to the real AVIF on the CDN.
export const inMemoryImages = $state(new SvelteMap<string, InMemoryImage>())

// Normalize ANY src form to the map key (base CDN name "<companyID>_<imageID>"):
// drop an optional folder prefix ("img-productos/123_57") and an optional "-x{n}" resolution
// suffix ("123_57-x2"). Lets every renderer look an image up by whatever src it happens to hold.
const toBaseKey = (src: string): string =>
  (src || "").split("/").pop()!.replace(/-x\d$/, "")

// The in-flight entry for any src form (folder/suffix tolerant), or undefined once it's done.
export const getInMemoryImage = (src: string): InMemoryImage | undefined =>
  inMemoryImages.get(toBaseKey(src))

// Returns the optimistic base64 for any src form while it is still in flight, else "".
export const getInMemoryImageBase64 = (src: string): string =>
  getInMemoryImage(src)?.base64 || ""

// True while the image for `src` is still converting/uploading (any page, any component).
// Drives the corner loader independently of whichever parent rendered the image.
export const isImageInFlight = (src: string): boolean => {
  const entry = getInMemoryImage(src)
  return !!entry && entry.status !== 'error'
}
