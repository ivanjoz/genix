import PdfJsWorker from "pdfjs-dist/build/pdf.worker?worker";

export function createPdfWorker() {
  return new PdfJsWorker();
}