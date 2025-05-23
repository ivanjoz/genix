import { defineConfig } from "@solidjs/start/config";
import tailwindcss from '@tailwindcss/vite'
import type { Options } from "vite-plugin-solid"
import path from 'path'

const IS_PRD = process.env.npm_lifecycle_event !== 'dev'
console.log("IS_PROD:", IS_PRD)

export default defineConfig({
  ssr: false,
  server: {
    prerender: {
      preset: "cloudflare-pages-static",
      autoSubfolderIndex: false,
      routes: ["/","/page/1"],
      baseURL: "dasdasd"
    },
  },
  devOverlay: false,
  solid: {
    hot: false,
    ssr: false,
  } as Options,
  vite() {
    return { 
      plugins: [
        tailwindcss()
      ].filter(x => x),
      resolve: {
        alias: {
          "@styles": path.resolve(process.env.PWD as string,'src/styles'),
        }
      },
      server: {
        hmr: false,
      },
    }
  }
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