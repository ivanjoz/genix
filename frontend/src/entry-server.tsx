import { createHandler, StartServer } from "@solidjs/start/server";

export default createHandler(() => (
  <StartServer document={({ assets, children, scripts }) => (
      <html lang="en">
        <head>
          <meta charset="utf-8" />
          <base href="/" />
          <meta name="viewport" content="width=device-width, initial-scale=1" />
          <link rel="icon" href="/favicon.ico" />
          <link rel="stylesheet" href="libs/fontello-embedded.css"/>
          <link rel="manifest" href="manifest.webmanifest" />
          <meta name="theme-color" content="#4a4ca0" />
          <script async defer src="https://js.culqi.com/checkout-js" />
          <script>
            Culqi.publicKey = 'pk_test_hjygMpDcsi6Mp5wA';
          </script>
          { assets }
        </head>
        <body>
          <div id="app">{children}</div>
          { scripts }
        </body>
      </html>
    )}
  />
))