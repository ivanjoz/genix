import { POST } from '$libs/http.svelte';
import type { IUser } from '$core/types/common';

export const postUser = (data: IUser) => {
  return POST({
    data,
    route: "users",
    refreshRoutes: ["users"]
  })
}

export const postOwnUser = (data: IUser) => {
  return POST({
    data,
    route: "user-self",
    refreshRoutes: ["users"]
  })
}
