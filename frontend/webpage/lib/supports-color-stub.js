// Storefront-only stub for supports-color, aliased in only for the renderer build
// (VITE_RENDERER_BUILD set), which is what gets published — see webpage/vite.config.ts.
//
// Lo pide `debug` con un require() opcional dentro de un try/catch, solo para decidir si
// colorea. En el artefacto publicado ese require no resuelve — corre sin node_modules — y
// aunque el catch se lo traga, deja un paquete fuera del bundle: exactamente lo que el guard
// de scripts/build-renderer.mjs prohíbe, porque es indistinguible de uno que sí hace falta.
//
// La respuesta correcta aquí es además la de verdad: la salida del renderer va a journald o
// a CloudWatch, donde los códigos de color son ruido. `false` es lo que devuelve la librería
// real cuando el destino no admite color.
export default { stdout: false, stderr: false };
