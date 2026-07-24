import { base } from '$app/paths';
import { tr } from '$core/store.svelte';
import {
  ExcelBuilder as ReusableExcelBuilder,
  buildExcelBuffer as buildReusableExcelBuffer,
  createExcelRuntime,
  downloadExcel as downloadReusableExcel,
  parseExcelFile as parseReusableExcelFile,
  toExcelColumns as toReusableExcelColumns,
  type ExcelBuildOptions,
  type ExcelDownloadOptions,
  type ExcelImportOptions,
} from '@genix/ui/excel';

const genixExcelRuntime = createExcelRuntime({
  wasmUrl: `${base}/vendor/excelize.wasm.bin`,
  applicationName: 'Genix',
  translate: tr,
});

// Business modules keep a zero-configuration builder while the package stays host-neutral.
export class ExcelBuilder<RecordType> extends ReusableExcelBuilder<RecordType> {
  constructor() {
    super(genixExcelRuntime);
  }
}

export const buildExcelBuffer = <RecordType>(options: ExcelBuildOptions<RecordType>) =>
  buildReusableExcelBuffer(genixExcelRuntime, options);

export const downloadExcel = <RecordType>(options: ExcelDownloadOptions<RecordType>) =>
  downloadReusableExcel(genixExcelRuntime, options);

export const parseExcelFile = <RecordType>(options: ExcelImportOptions<RecordType>) =>
  parseReusableExcelFile(genixExcelRuntime, options);

export const toExcelColumns = <RecordType>(
  columns: ExcelImportOptions<RecordType>['columns'],
) => toReusableExcelColumns(genixExcelRuntime, columns);

export * from '@genix/ui/excel';
