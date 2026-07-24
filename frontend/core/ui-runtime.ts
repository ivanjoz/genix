import {
  configureCacheRuntime,
  configureServiceWorkerRuntime,
  createUiRuntime,
  getRecordWithCache,
  type UiInMemoryImage,
  type UiRuntime,
} from '@genix/ui';
import {
  getInMemoryImageBase64,
  inMemoryImages,
  isImageInFlight,
} from './inMemoryImages.svelte';
import { addProcess, updateProcess } from './notifications.svelte';
import { Env } from './env';
import { fetchEvent, tr, Core } from './store.svelte';
import { fileToImage, Notify, persistFieldValue, readFieldValue } from '$libs/helpers';
import { GET, POST_XMLHR, setFetchProgress } from '$libs/http.svelte';

export const createGenixUiRuntime = (): UiRuntime => {
  // Cache IO stays package-owned; Genix injects authentication, tenant identity, and routing.
  configureCacheRuntime({
    getCompanyID: Env.getCompanyID,
    getEnvironment: () => Env.enviroment || 'main',
    get: GET,
    navigate: Env.navigate,
  });
  configureServiceWorkerRuntime({
    getWorkerUrl: () => Env.serviceWorker,
    getEnvironment: () => Env.enviroment || 'main',
    getCompanyID: Env.getCompanyID,
    makeRoute: Env.makeRoute,
    verifyRouteMemoryState: () => Env.DELTA_CACHE_VERIFY_ROUTE_MEMORY,
    reportFetch: fetchEvent,
    reportProgress: setFetchProgress,
    notifyFailure: Notify.failure,
  });

  // Genix owns tenant-aware IO; the package receives only the capabilities its UI needs.
  return createUiRuntime({
    defaultLanguage: Core.languaje,
    translate: tr,
    makeCdnRoute: Env.makeCDNRoute,
    notify: Notify,
    images: {
      entries: inMemoryImages as unknown as Map<string, UiInMemoryImage>,
      getBase64: getInMemoryImageBase64,
      isInFlight: isImageInFlight,
    },
    uploads: {
      get: GET,
      post: POST_XMLHR,
      convertImage: fileToImage,
      addProcess,
      updateProcess,
    },
    persistFieldValue,
    readFieldValue,
    resolveRecord: ((apiRoute: string, recordId: number) =>
      getRecordWithCache(apiRoute, recordId)) as UiRuntime['resolveRecord'],
  });
};
