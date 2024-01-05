import { createSignal } from "solid-js";

export function Counter() {
  const [count, setCount] = createSignal(0);

  /*
  if(typeof window !== 'undefined'){
    window._setCount = setCount
    window._count = count
  }
  */

  return (
    <button class="increment" onClick={() => setCount(count() + 1)}>
      Clicks: {count()}
    </button>
  );
}
