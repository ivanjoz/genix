import { getWindow } from "../env"

export const checkDevice = () => {
  const Window = getWindow()
  if(Window.innerWidth <= 680) return 3
  else if(Window.innerWidth <= 940) return 2
  else { return 1 }
}

export let Globals = $state({
  deviceType: checkDevice()
});

export let Ecommerce = $state({
  cartOption: 1
});
