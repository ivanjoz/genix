import { genixUiRuntime } from '$libs/ui-runtime.svelte';
import {
  GetHandler as ReusableGetHandler,
  type GetHandlerRecord,
} from '@genix/ui/http';

export type {
  AxiosProgressEvent,
  INewIDToID,
  IHttpStatus,
  httpProps,
} from '@genix/ui/http';

export const {
  buildHeaders,
  GET,
  GETWithGroupCache,
  POST,
  PUT,
  POST_XMLHR,
} = genixUiRuntime.http

export const {
  fileToImage,
  bitmapToImage,
} = genixUiRuntime.imageConverter

// Genix services keep a zero-argument base class while the package receives host policy explicitly.
export class GetHandler<T extends GetHandlerRecord = any> extends ReusableGetHandler<T> {
  constructor() {
    super(genixUiRuntime.getHandlerRuntime)
  }
}
