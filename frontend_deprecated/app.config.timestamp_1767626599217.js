// app.config.ts
import { defineConfig } from "@solidjs/start/config";
import tailwindcss from "@tailwindcss/vite";
import path from "path";
var IS_PRD = process.env.npm_lifecycle_event !== "dev";
console.log("IS_PROD:", IS_PRD);
var app_config_default = defineConfig({
  ssr: false,
  server: {
    prerender: {
      preset: "cloudflare-pages-static",
      autoSubfolderIndex: false,
      routes: ["/", "/page/1"],
      baseURL: "dasdasd"
    }
  },
  devOverlay: false,
  solid: {
    hot: false,
    ssr: false
  },
  vite() {
    return {
      plugins: [
        tailwindcss()
      ].filter((x) => x),
      resolve: {
        alias: {
          "@styles": path.resolve(process.env.PWD, "src/styles")
        }
      },
      server: {
        hmr: false
      }
    };
  }
});
export {
  app_config_default as default
};
