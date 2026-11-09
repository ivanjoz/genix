import { Env } from '#core/env.ts';
import { GetHandler } from '#libs/ui-runtime.svelte.ts';

export interface IGalleryImage {
  ID: number;
  ImageID: number;
  Image: string;
  Description: string;
  ss: number;
  upd: number;
}

export class GalleryImagesService extends GetHandler<IGalleryImage> {
  route = 'gallery-images';
  useCache = { min: 5, ver: 2 };
  keyID = 'ImageID';
  inferRemoveFromStatus = true;
  prependOnSave = true;

  handler(result: IGalleryImage[]): void {
    this.records = [];
    this.recordsMap = new Map();
    // GetHandler indexes records by ID; ImageID remains the persisted backend key.
    this.addSavedRecords(...(result || []).map((image) => ({ ...image, ID: image.ImageID })));
  }

  imageURL(image: IGalleryImage, size: 4 | 8 = 4): string {
    return Env.makeCDNRoute('img-galeria', `${image.Image}-x${size}.avif`);
  }

  async remove(imageID: number): Promise<void> {
    // The backend soft-deletes the row so cached clients receive the eviction delta, which comes
    // back with the write.
    await this.writeAndSync('gallery-image', { ImageToDelete: imageID });
  }

  constructor(init: boolean = false) {
    super();
    if (init) this.fetch();
  }
}
