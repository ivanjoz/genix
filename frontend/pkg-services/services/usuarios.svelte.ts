import { POST } from '$core/http.svelte';
import type { IUsuario } from '$core/types/common';

export const postUsuario = (data: IUsuario) => {
  return POST({
    data,
    route: "usuarios",
    refreshRoutes: ["usuarios"]
  })
}

export const postUsuarioPropio = (data: IUsuario) => {
  return POST({
    data,
    route: "usuario-propio",
    refreshRoutes: ["usuarios"]
  })
}
