import { defineConfig } from "@solidjs/start/config";

export default defineConfig({
  start: { ssr: false, 
    server: { 
      hmr: false,
      /*
      prerender: {
        crawlLinks: true
      }
      */
    } 
  },
  server: { hmr: false },
  base: '/',
})

/*
export default defineConfig({
  server: {
     port: 3588,  hmr: false,
  },
  start: { ssr: false, server: {  } },
  base: '/',
  // optimizeDeps: { force: false },
  plugins: [
    { name: 'html-transform',
      transformIndexHtml(html) {
        console.log("html::", html)
        return html.replace(
          `<body ></body>`,
          `
          <head>
            <title>GENIX - MyPes</title>
            <base href="/"/>
            <meta charset="utf-8"/>
            <meta name="viewport" content="width=device-width, initial-scale=1" />
            <meta name="theme-color" content="#4285f4" />
            <link rel="manifest" href="manifest.webmanifest" />
          </head>
          <body ></body>`,
        )
      },
    },
  ],
});
*/