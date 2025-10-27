<script lang="ts" generics="T">
    import { Popover2 } from "./popover2";


	export interface ICellEditableProps<T> {
    id?: string
		saveOn: T;
		save?: string;
		css?: string;
		contentClass?: string;
		inputClass?: string;
		onChange?: (newValue: string | number) => void;
		render?: (value: number | string, isEditing: boolean) => any;
		getValue?: (e: T) => number | string;
		required?: boolean;
		type?: string;
	}

  let {
		id
	}: ICellEditableProps<T> = $props();

  let showTop = $state(false);
  let refElement: HTMLDivElement | null = $state(null);

</script>

<div>

  <div role="button" id="{id}"
    tabindex="0"
    bind:this={refElement}
    onclick={() => showTop = !showTop}
    onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && (showTop = !showTop)}
    class="px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
  >
    Hello
  </div>
  <Popover2
    referenceElement={refElement}
    open={showTop}
    placement="bottom"
  >
    <div class="popover2-container _1">
      <div class="popover2-content">
        This popover prefers top placement but will flip if there's no space above.
      </div>
    </div>
  </Popover2>

</div>

<style>
  ._1 {
    background-color: aqua;
  }

</style>