import { getProductos } from "../services/productos.svelte";

export const csr = true;
export const ssr = true;
export const prerender = true;
// This prevents automatic data serialization
export const trailingSlash = 'ignore';

const localHosts = ["localhost","127.0.0.1","sveltekit-prerender"]

export async function load({ url }) {
  (globalThis as any)._isLocal = localHosts.some(x => url.host.includes(x))

  console.log("Env is local? = ", (globalThis as any)._isLocal,"|", url.host)

  console.log("obteniendo productos...");
  const productos = await getProductos(null)
  console.log("productos obtenidos:", productos.productos?.length)

  return { productos }
}