import { domToCanvas } from 'modern-screenshot';

// Captures the builder's page content (.builder-canvas) into a cropped PNG blob for
// use as the page thumbnail. The crop is width-driven: whole sections are included
// from the top until their cumulative height reaches at least the canvas width, so
// the result is square-or-taller. Builder-only chrome (section outlines + labels) is
// hidden during the capture via the `capturing` class (see EcommerceBuilder.svelte).
// Returns null when there is nothing to capture.
export const captureShowcaseBlob = async (): Promise<Blob | null> => {
  const canvas = document.querySelector<HTMLElement>('.builder-canvas');
  // No .builder-canvas in the document means mobile preview mode (the canvas is
  // mounted inside MobilePreviewFrame's iframe) — nothing to screenshot here.
  if (!canvas) {
    console.warn('[showcase] no .builder-canvas in the document (mobile preview mode?) — capture skipped');
    return null;
  }

  const width = canvas.clientWidth;
  const sections = Array.from(canvas.querySelectorAll<HTMLElement>('.section-wrapper'));
  if (width === 0 || sections.length === 0) {
    console.warn('[showcase] nothing to capture::', { width, sections: sections.length });
    return null;
  }

  // Include whole sections from the top until we reach >= 1x the width (square min).
  let cropHeightCss = 0;
  for (const section of sections) {
    cropHeightCss += section.offsetHeight;
    if (cropHeightCss >= width) break;
  }
  console.debug('[showcase] capturing::', { width, sections: sections.length, cropHeightCss, scrollHeight: canvas.scrollHeight });

  canvas.classList.add('capturing');
  try {
    const fullCanvas = await domToCanvas(canvas, { backgroundColor: '#ffffff' });
    // domToCanvas may scale the output (devicePixelRatio); map CSS px -> canvas px.
    const scale = fullCanvas.width / width;
    const cropHeightPx = Math.min(Math.round(cropHeightCss * scale), fullCanvas.height);
    console.debug('[showcase] domToCanvas done::', { canvasWidth: fullCanvas.width, canvasHeight: fullCanvas.height, scale, cropHeightPx });

    const cropped = document.createElement('canvas');
    cropped.width = fullCanvas.width;
    cropped.height = cropHeightPx;
    const ctx = cropped.getContext('2d')!;
    ctx.drawImage(fullCanvas, 0, 0, fullCanvas.width, cropHeightPx, 0, 0, fullCanvas.width, cropHeightPx);

    const blob = await new Promise<Blob | null>((resolve) => cropped.toBlob(resolve, 'image/png'));
    console.debug('[showcase] png blob::', blob ? `${blob.size} bytes` : 'null (toBlob failed)');
    // Expose the raw capture as a clickable URL so what the screenshot actually
    // contains can be compared against the live canvas (before any conversion).
    if (blob) { console.debug('[showcase] captured png preview::', URL.createObjectURL(blob)); }
    return blob;
  } catch (error) {
    // domToCanvas can reject (huge foreignObject SVG, unfetchable asset). Log it and
    // let the caller continue — the page save must not depend on the thumbnail.
    console.error('[showcase] domToCanvas failed::', error);
    return null;
  } finally {
    canvas.classList.remove('capturing');
  }
};
