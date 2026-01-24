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
  
}
declare const appId: string;
