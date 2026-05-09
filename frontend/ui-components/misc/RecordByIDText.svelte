<script lang="ts">
  import {
    getRecordWithCache,
    type IMinimalRecord,
    type IRecordRef,
  } from '$libs/cache/cache-by-ids.svelte';
  import { untrack } from 'svelte';

  interface ITextRecord extends IMinimalRecord {
    Usuario?: string;
    Nombre?: string;
    Nombres?: string;
    Apellidos?: string;
    Name?: string;
  }

  let {
    apiRoute,
    recordID,
    placeholder = '-',
  }: {
    apiRoute: string;
    recordID: number;
    placeholder?: string;
  } = $props();

  let recordReference = $state<IRecordRef<ITextRecord> | null>(null);
  let lastResolvedRecordID = $state(0);

  const normalizedRecordID = $derived.by(() => Number(recordID || 0));
  const resolvedRecord = $derived(recordReference?.record || null);

  const resolvedText = $derived.by(() => {
    if (!resolvedRecord) { return placeholder; }

    // Prefer common display keys used across modules before fallback to placeholder.
    if (resolvedRecord.Usuario) { return resolvedRecord.Usuario; }
    if (resolvedRecord.Nombre) { return resolvedRecord.Nombre; }
    if (resolvedRecord.Name) { return resolvedRecord.Name; }
    const fullName = [resolvedRecord.Nombres, resolvedRecord.Apellidos].filter(Boolean).join(' ').trim();
    if (fullName.length > 0) { return fullName; }
    return placeholder;
  });

  $effect(() => {
    const nextRecordID = normalizedRecordID;
    if (!(nextRecordID > 0)) {
      untrack(() => {
        recordReference = null;
        lastResolvedRecordID = 0;
      });
      return;
    }

    const hasSameTargetRecord = untrack(() => lastResolvedRecordID === nextRecordID);
    if (hasSameTargetRecord) {
      return;
    }

    untrack(() => {
      // Keep one cache ref per current ID to avoid unnecessary re-instantiations.
      lastResolvedRecordID = nextRecordID;
      recordReference = getRecordWithCache<ITextRecord>(apiRoute, nextRecordID);
    });
  });
</script>

<span>{resolvedText}</span>
