export const csr = true;
export const ssr = false;
export const prerender = false;
// This prevents automatic data serialization
export const trailingSlash = 'ignore';

const localHosts = ["localhost","127.0.0.1","sveltekit-prerender"]

export async function load({ url }) {
  (globalThis as any)._isLocal = localHosts.some(x => url.host.includes(x))

  console.log("Env is local? = ", (globalThis as any)._isLocal,"|", url.host)

  // The catalog is loaded client-side from the single shared source (see +layout.svelte onMount).
  return {}
}
