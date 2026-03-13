<script lang="ts" generics="T,TLeftOption,TRightOption">
  import type { Snippet } from "svelte";
  import { untrack } from "svelte";
  import SearchSelect from "$components/SearchSelect.svelte";
  import { WeakSearchRef } from "$core/store.svelte";

  type SearchOptionID = string | number;
  type SearchDualCardSource = "left" | "right";

  export interface ISearchDualCardSelectedItem<TOption> {
    source: SearchDualCardSource;
    id: SearchOptionID;
    option: TOption;
  }

  interface SearchDualCardProps<T, TLeftOption, TRightOption> {
    saveOn?: T;
    saveLeft: keyof T;
    saveRight: keyof T;
    css?: string;
    cardCss?: string;
    sharedLabel?: string;
    onChange?: (payload: {
      leftSelectedIDs: SearchOptionID[];
      rightSelectedIDs: SearchOptionID[];
    }) => void;
    leftOptions: TLeftOption[];
    leftKeyId: keyof TLeftOption;
    leftKeyName: keyof TLeftOption;
    leftLabel?: string;
    leftOptionsCss?: string;
    leftInputCss?: string;
    rightOptions: TRightOption[];
    rightKeyId: keyof TRightOption;
    rightKeyName: keyof TRightOption;
    rightLabel?: string;
    rightOptionsCss?: string;
    rightInputCss?: string;
    selectedItem?: Snippet<[ISearchDualCardSelectedItem<TLeftOption | TRightOption>]>;
  }

  const {
    saveOn = $bindable(),
    saveLeft,
    saveRight,
    css = "",
    cardCss = "",
    sharedLabel,
    onChange,
    leftOptions = [],
    leftKeyId,
    leftKeyName,
    leftLabel,
    leftOptionsCss,
    leftInputCss,
    rightOptions = [],
    rightKeyId,
    rightKeyName,
    rightLabel,
    rightOptionsCss,
    rightInputCss,
    selectedItem
  }: SearchDualCardProps<T, TLeftOption, TRightOption> = $props();

  let leftSelectedIDs = $state<SearchOptionID[]>([]);
  let rightSelectedIDs = $state<SearchOptionID[]>([]);

  // Cache option lookups per source list so selected IDs can be resolved without repeated scans.
  function buildLookup<TOption>(
    optionRecords: TOption[],
    optionKeyId: keyof TOption,
    optionKeyName: keyof TOption
  ) {
    if (WeakSearchRef.has(optionRecords)) { return; }

    const optionById = new Map<SearchOptionID, TOption>();
    const optionByName = new Map<string, TOption>();

    for (const optionRecord of optionRecords) {
      const optionId = optionRecord[optionKeyId] as SearchOptionID;
      const optionName = String(optionRecord[optionKeyName] || "").toLowerCase();
      optionById.set(optionId, optionRecord);
      optionByName.set(optionName, optionRecord);
    }

    WeakSearchRef.set(optionRecords, { idToRecord: optionById, valueToRecord: optionByName });
  }

  function getSavedIDs(fieldName: keyof T): SearchOptionID[] {
    const rawValue = saveOn?.[fieldName] as SearchOptionID[] | undefined;
    return Array.isArray(rawValue) ? [...rawValue] : [];
  }

  function areSameIDs(leftValues: SearchOptionID[], rightValues: SearchOptionID[]) {
    if (leftValues.length !== rightValues.length) { return false; }
    return leftValues.every((value, index) => value === rightValues[index]);
  }

  // Keep local state synchronized with the bound form object without creating render loops.
  function syncSelectedIDsFromProps() {
    if (!saveOn) { return; }

    const nextLeftSelectedIDs = getSavedIDs(saveLeft);
    const nextRightSelectedIDs = getSavedIDs(saveRight);

    if (
      areSameIDs(leftSelectedIDs, nextLeftSelectedIDs) &&
      areSameIDs(rightSelectedIDs, nextRightSelectedIDs)
    ) {
      return;
    }

    console.debug("SearchDualCard::syncSelectedIDsFromProps", {
      saveLeft: String(saveLeft),
      saveRight: String(saveRight),
      nextLeftSelectedIDs,
      nextRightSelectedIDs
    });

    untrack(() => {
      leftSelectedIDs = nextLeftSelectedIDs;
      rightSelectedIDs = nextRightSelectedIDs;
    });
  }

  function commitSelectedIDs() {
    if (saveOn) {
      saveOn[saveLeft] = [...leftSelectedIDs] as NonNullable<T>[keyof T];
      saveOn[saveRight] = [...rightSelectedIDs] as NonNullable<T>[keyof T];
    }

    console.debug("SearchDualCard::commitSelectedIDs", {
      saveLeft: String(saveLeft),
      saveRight: String(saveRight),
      leftSelectedIDs: $state.snapshot(leftSelectedIDs),
      rightSelectedIDs: $state.snapshot(rightSelectedIDs)
    });

    onChange?.({
      leftSelectedIDs: [...leftSelectedIDs],
      rightSelectedIDs: [...rightSelectedIDs]
    });
  }

  function getOptionRecord<TOption>(
    optionRecords: TOption[],
    optionKeyId: keyof TOption,
    optionKeyName: keyof TOption,
    optionID: SearchOptionID
  ): TOption {
    const optionById = WeakSearchRef.get(optionRecords)?.idToRecord || new Map();
    return optionById.get(optionID) as TOption
      || { [optionKeyId]: optionID, [optionKeyName]: `ID-${optionID}` } as TOption;
  }

  function addLeftSelectedID(optionRecord?: TLeftOption) {
    if (!optionRecord) { return; }

    const optionID = optionRecord[leftKeyId] as SearchOptionID;
    if (leftSelectedIDs.includes(optionID)) { return; }

    console.debug("SearchDualCard::addLeftSelectedID", { optionID, optionRecord });
    leftSelectedIDs = [...leftSelectedIDs, optionID];
    commitSelectedIDs();
  }

  function addRightSelectedID(optionRecord?: TRightOption) {
    if (!optionRecord) { return; }

    const optionID = optionRecord[rightKeyId] as SearchOptionID;
    if (rightSelectedIDs.includes(optionID)) { return; }

    console.debug("SearchDualCard::addRightSelectedID", { optionID, optionRecord });
    rightSelectedIDs = [...rightSelectedIDs, optionID];
    commitSelectedIDs();
  }

  function removeSelectedID(source: SearchDualCardSource, optionID: SearchOptionID) {
    console.debug("SearchDualCard::removeSelectedID", { source, optionID });

    if (source === "left") {
      leftSelectedIDs = leftSelectedIDs.filter((currentID) => currentID !== optionID);
    } else {
      rightSelectedIDs = rightSelectedIDs.filter((currentID) => currentID !== optionID);
    }

    commitSelectedIDs();
  }

  // Resolve the visible label in one place so the default renderer stays simple and type-safe.
  function getSelectedItemName(selectedOption: ISearchDualCardSelectedItem<TLeftOption | TRightOption>) {
    if (selectedOption.source === "left") {
      return String((selectedOption.option as TLeftOption)[leftKeyName] || "");
    }
    return String((selectedOption.option as TRightOption)[rightKeyName] || "");
  }

  const selectedItems = $derived.by(() => {
    const mergedSelectedItems: ISearchDualCardSelectedItem<TLeftOption | TRightOption>[] = [];

    for (const optionID of leftSelectedIDs) {
      mergedSelectedItems.push({
        source: "left",
        id: optionID,
        option: getOptionRecord(leftOptions, leftKeyId, leftKeyName, optionID)
      });
    }

    for (const optionID of rightSelectedIDs) {
      mergedSelectedItems.push({
        source: "right",
        id: optionID,
        option: getOptionRecord(rightOptions, rightKeyId, rightKeyName, optionID)
      });
    }

    return mergedSelectedItems;
  });

  $effect(() => {
    buildLookup(leftOptions, leftKeyId, leftKeyName);
  });

  $effect(() => {
    buildLookup(rightOptions, rightKeyId, rightKeyName);
  });

  $effect(() => {
    syncSelectedIDsFromProps();
  });
</script>

<div class={css}>
  <div class="grid grid-cols-24 gap-10">
    <SearchSelect
      options={leftOptions}
      keyId={leftKeyId}
      keyName={leftKeyName}
      clearOnSelect={true}
      avoidIDs={leftSelectedIDs}
      placeholder={leftLabel}
      css={`col-span-24 md:col-span-12 s1 ${leftInputCss || ""}`}
      optionsCss={leftOptionsCss}
      onChange={addLeftSelectedID}
    />
    <SearchSelect
      options={rightOptions}
      keyId={rightKeyId}
      keyName={rightKeyName}
      clearOnSelect={true}
      avoidIDs={rightSelectedIDs}
      placeholder={rightLabel}
      css={`col-span-24 md:col-span-12 s1 ${rightInputCss || ""}`}
      optionsCss={rightOptionsCss}
      onChange={addRightSelectedID}
    />
  </div>

  <div class={`p-4 min-h-40 _container ${cardCss}`}>
    {#if sharedLabel}
      <div class="_shared-label">{sharedLabel}</div>
    {/if}

    <div class="flex flex-wrap">
      {#each selectedItems as currentSelectedItem (`${currentSelectedItem.source}-${currentSelectedItem.id}`)}
        {@const removeSelected = () => removeSelectedID(currentSelectedItem.source, currentSelectedItem.id)}
        {#if selectedItem}
          <div class={`m-2 px-8 py-6 min-w-56 lh-10 flex _chip ${currentSelectedItem.source === "right" ? "_chip-right" : "_chip-left"}`}>
            <span class="_chip-text">
              {@render selectedItem(currentSelectedItem)}
            </span>
            <button
              class="_chip-remove absolute w-28 h-28 rounded right-2 top-2"
              aria-label="eliminar"
              onclick={(event) => {
                event.stopPropagation();
                removeSelected();
              }}
            >
              <i class="icon-trash"></i>
            </button>
          </div>
        {:else}
          <div class={`m-2 px-8 py-6 min-w-56 lh-10 flex _chip ${currentSelectedItem.source === "right" ? "_chip-right" : "_chip-left"}`}>
            <span class="_chip-text">{getSelectedItemName(currentSelectedItem)}</span>
            <button
              class="_chip-remove absolute w-28 h-28 rounded right-2 top-2"
              aria-label="eliminar"
              onclick={(event) => {
                event.stopPropagation();
                removeSelected();
              }}
            >
              <i class="icon-trash"></i>
            </button>
          </div>
        {/if}
      {/each}
    </div>
  </div>
</div>

<style>
  ._container {
    background-color: var(--light-blue-1);
    border-radius: 5px;
    box-shadow: #5f7187a8 0 1px 3px -1px;
  }

  ._shared-label {
    color: #6d5dad;
    line-height: 18px;
    margin-bottom: 8px;
  }

  ._chip {
    align-items: center;
    background-color: #fff;
    border: 1px solid #dfe1ea;
    border-radius: 4px;
    color: inherit;
    cursor: pointer;
    justify-content: center;
    min-height: 32px;
    position: relative;
    user-select: none;
  }

  ._chip-left {
    border-color: #d6d8f6;
  }

  ._chip-right {
    border-color: #d9e0f7;
  }

  ._chip:hover {
    border-color: rgb(236, 125, 125);
    color: rgb(209, 66, 66);
  }

  ._chip-text {
    display: block;
    width: 100%;
  }

  ._chip-remove {
    border-radius: 50%;
    font-size: 14px;
    opacity: 0;
  }

  ._chip:hover ._chip-remove {
    background-color: rgb(255, 221, 221);
    color: rgb(224, 61, 61);
    opacity: 1;
  }

  ._chip:hover ._chip-remove:hover {
    background-color: rgb(240, 102, 102);
    color: white;
  }
</style>
