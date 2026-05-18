import { POST } from '$libs/http.svelte';
import type { IUser } from '$core/types/common';

export const postUsuario = (data: IUser) => {
  return POST({
    data,
    route: "usuarios",
    refreshRoutes: ["usuarios"]
  })
}

export const postUsuarioPropio = (data: IUser) => {
  return POST({
    data,
    route: "usuario-propio",
    refreshRoutes: ["usuarios"]
  })
}
