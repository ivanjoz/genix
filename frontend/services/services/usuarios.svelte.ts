import { POST } from '$ecommerce/node_modules/@sveltejs/kit/src/utils/http';
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
