/// <reference types="vite/client" />

type MyGlobalFunctionType = (name: string) => void
type ConsoleLog = (...e: any) => void
type AppHistory = (route: string) => void
type GlobalParams = { [key: string]: any, counter?: number }
type IndexedDBTables = { [key: string]: string }

namespace JSX {
  interface IntrinsicElements {
    'json-viewer': React.DetailedHTMLProps<
      React.HTMLAttributes<HTMLElement>,
      HTMLElement
    >;
  }
}

// probando branch
interface Window {
  appId: string
  PRD_HOSTS: strings[]
  QAS_HOSTS: strings[]
  _env: {};
  // The live storefront route starts fetching its published CDN snapshot from an
  // inline <head> script (before the JS bundle loads) and parks the response here;
  // onMount awaits it instead of issuing a second fetch.
  _pageContentPromise?: Promise<unknown>;
}

// Build-time constants injected via Vite
declare module '$domain/libs/blurhash?raw' {
	const content: string;
	export default content;
}
declare const appId: string;
