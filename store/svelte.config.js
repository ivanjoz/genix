import adapter from "@sveltejs/adapter-static";

const BASE64_CHARS =
  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

const numberToBase64 = (num) => {
  let value = "";
  while (num > 0) {
    value = BASE64_CHARS[num % 62] + value;
    num = Math.floor(num / 62);
  }
  return value;
}

/** @type {import('@sveltejs/kit').Config} */
export default {
  compilerOptions: {
    css: 'external',
    cssHash: ({ filename }) => {
      filename = filename.split("/src/")[1]
      if(!globalThis._fileNameMap.has(filename)){
        const code = numberToBase64(globalThis._fileNameMap.size + 1)
        globalThis._fileNameMap.set(filename, code)
        console.log("\nhash filename::", filename)
      }
      return globalThis._fileNameMap.get(filename)
    },
  },
  kit: {
    adapter: adapter({
      // default options are shown. On some platforms
      // these options are set automatically — see below
      inlineStyleThreshold: Infinity,
      pages: "build",
      assets: "build",
      fallback: undefined,
      precompress: false,
      strict: true,
    }),
    // ssr: false,
    prerender: {
      handleMissingId: 'ignore',
      handleHttpError: ({ path, referrer, status, message }) => {
        return 'fail';
        /*
        try {
          console.log("=== ERROR HANDLER START ===");
          console.log("hola 1");
          
          // Add more debugging info
          console.log("Error details:", { path, referrer, status, message });
          console.log("hola 2");
          
          // Check if the regex is causing issues
          console.log("About to test regex...");
          const isImageFile = path.match(/\.(jpg|jpeg|png|gif|webp|svg|avif)$/i);
          console.log("Regex result:", isImageFile);
          console.log("hola 2.5");
          
          if (isImageFile || (status === 404 && referrer)) {
            console.log("hola 3 - inside if");
            console.log(`Ignoring 404 for ${path} (linked from ${referrer})`);
            console.log("hola 4 - about to return ignore");
            return 'ignore';
          } else {
            console.log("hola 5 - in else branch");
          }

          console.log("hola 6 - before final return");
          
          return 'ignore';
        } catch (error) {
          console.log("ERROR IN HANDLER:", error);
          return 'ignore';
        }
        */
      },
    },
  },
};
