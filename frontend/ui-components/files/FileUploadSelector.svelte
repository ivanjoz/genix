<script lang="ts">
import pdfFileIconRaw from '$components/svg/pdf-icon.svg?raw';
import excelFileIconRaw from '$components/svg/excel-icon.svg?raw';
import { Notify, parseSVG } from '$libs/helpers';

interface IFileUploadSelectorProps {
  onChange?: (file?: File, isRemoved?: boolean) => void;
  selectedFile?: File;
  accept?: string;
  extensions?: string[];
  disabled?: boolean;
  buttonLabel?: string;
}

let {
  onChange,
  selectedFile = $bindable<File | undefined>(undefined),
  accept = '',
  extensions = [],
  disabled = false,
  buttonLabel = 'Subir Archivo'
}: IFileUploadSelectorProps = $props();

let hiddenFileInputElement: HTMLInputElement | undefined;
const pdfIcon = parseSVG(pdfFileIconRaw);
const excelIcon = parseSVG(excelFileIconRaw);

const normalizedExtensions = $derived.by(() => {
  return extensions
    .map((extension) => extension.trim().toLowerCase().replace(/^\./, ''))
    .filter((extension) => extension.length > 0);
});

const iconExtensionsMap = new Map<string, string[]>([
  [pdfIcon, ['pdf']],
  [excelIcon, ['xls', 'xlsx', 'csv']]
]);

const extractExtension = (fileName?: string): string => {
  if (!fileName || !fileName.includes('.')) {
    return '';
  }

  return fileName.split('.').pop()?.toLowerCase() || '';
};

const selectedFileExtension = $derived(extractExtension(selectedFile?.name));

const effectiveAcceptAttribute = $derived.by(() => {
  if (accept.trim().length > 0) {
    return accept;
  } else if (normalizedExtensions.length === 0) {
    return '';
  }

  return normalizedExtensions.map((extension) => `.${extension}`).join(',');
});

const resolveFileTypeIcon = (fileExtension: string): string | undefined => {
  if (!fileExtension) {
    return undefined;
  }

  for (const [icon, mappedExtensions] of iconExtensionsMap) {
    if (mappedExtensions.includes(fileExtension)) {
      return icon;
    }
  }

  return undefined;
};

const openNativeFilePicker = () => {
  if (disabled) {
    return;
  }

  hiddenFileInputElement?.click();
};

const handleFileSelection = (event: Event) => {
  const inputElement = event.target as HTMLInputElement;
  const newSelectedFile = inputElement.files?.[0];

  if (!newSelectedFile) {
    return;
  }

  const selectedExtension = extractExtension(newSelectedFile.name);
  const hasExtensionRestriction = normalizedExtensions.length > 0;
  const extensionIsValid = !hasExtensionRestriction || normalizedExtensions.includes(selectedExtension);

  if (!extensionIsValid) {
    Notify.failure(`Extensión no permitida. Permitidas: ${normalizedExtensions.join(', ')}`);
    inputElement.value = '';
    return;
  }

  selectedFile = newSelectedFile;
  onChange?.(newSelectedFile, false);
};

const clearSelectedFile = () => {
  selectedFile = undefined;

  if (hiddenFileInputElement) {
    hiddenFileInputElement.value = '';
  }

  onChange?.(undefined, true);
};
</script>

<div class="inline-flex w-fit min-w-[280px] max-w-[400px] items-center gap-4 rounded-[10px] border border-[#e6e8eb] bg-white p-4">
  <input
    bind:this={hiddenFileInputElement}
    type="file"
    class="hidden"
    accept={effectiveAcceptAttribute}
    disabled={disabled}
    onchange={handleFileSelection}
  />

  <button
    type="button"
    class="flex min-w-0 grow items-center gap-5 rounded-lg border border-[#edf0f2] bg-[#fafbfc] px-6 py-5 text-left disabled:opacity-60"
    onclick={openNativeFilePicker}
    disabled={disabled}
  >
    {#if selectedFile}
      {@const selectedFileTypeIcon = resolveFileTypeIcon(selectedFileExtension)}
      {#if selectedFileTypeIcon}
        <img src={selectedFileTypeIcon} alt="Tipo de archivo" class="h-20 w-20 shrink-0 object-contain" />
      {:else}
        <i class="icon-upload text-[#3c4650]"></i>
      {/if}
      <span class="truncate fs13 leading-[1] text-[#2f3a44]">{selectedFile.name}</span>
    {:else}
      <i class="icon-upload text-[#3c4650]"></i>
      <span class="fs13 leading-[1] text-[#2f3a44]">{buttonLabel}</span>
    {/if}
  </button>

  {#if selectedFile}
    <button
      type="button"
      class="flex h-24 w-24 shrink-0 items-center justify-center rounded-[50%] border border-[#ffcfcf] bg-[#fff3f3] text-[#d72828]"
      aria-label="Quitar archivo"
      onclick={clearSelectedFile}
    >
      <i class="icon-cancel"></i>
    </button>
  {/if}
</div>
