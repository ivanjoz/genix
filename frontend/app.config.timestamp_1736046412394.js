// app.config.ts
import { defineConfig } from "@solidjs/start/config";
import path from "path";
var IS_PRD = process.env.npm_lifecycle_event !== "dev";
console.log("IS_PROD:", IS_PRD);
var app_config_default = defineConfig({
  ssr: false,
  // IS_PRD,
  server: {
    prerender: {
      route
    }
  },
  devOverlay: false,
  solid: {
    hot: false,
    ssr: false
  },
  vite() {
    return {
      plugins: [].filter((x) => x),
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
