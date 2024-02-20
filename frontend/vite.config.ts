import { defineConfig } from "@solidjs/start/config";

export default defineConfig({
  start: {
    ssr: false,
  },
  server: {
    hmr: false
  },
})

/*
export default defineConfig({
  start: { 
    ssr: false, 
    
    server: { 
      prerender: {
        // autoSubfolderIndex: true,
        routes: ["/admin/demo3"],
        // crawlLinks: false,
      }
    } 
  },
  server: { hmr: false },
  base: '/',
})
*/

/*

export default defineConfig({
  start: { 
    ssr: true, 
    server: { 
      prerender: {
        // autoSubfolderIndex: true,
        routes: ["/"],
        // crawlLinks: false,
      }
    } 
  },
  server: { hmr: false },
  base: '/',
})
*/