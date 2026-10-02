import { POST } from '#libs/ui-runtime.svelte.ts';
import type { IUser } from '#core/types/common.ts';

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
