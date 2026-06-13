import { domToCanvas } from 'modern-screenshot';

// Captures the builder's page content (.builder-canvas) into a cropped PNG blob for
// use as the page thumbnail. The crop is width-driven: whole sections are included
// from the top until their cumulative height reaches at least the canvas width, so
// the result is square-or-taller. Builder-only chrome (section outlines + labels) is
// hidden during the capture via the `capturing` class (see EcommerceBuilder.svelte).
// Returns null when there is nothing to capture.
export const captureShowcaseBlob = async (): Promise<Blob | null> => {
  const canvas = document.querySelector<HTMLElement>('.builder-canvas');
  if (!canvas) return null;

  const width = canvas.clientWidth;
  const sections = Array.from(canvas.querySelectorAll<HTMLElement>('.section-wrapper'));
  if (width === 0 || sections.length === 0) return null;

  // Include whole sections from the top until we reach >= 1x the width (square min).
  let cropHeightCss = 0;
  for (const section of sections) {
    cropHeightCss += section.offsetHeight;
    if (cropHeightCss >= width) break;
  }

  canvas.classList.add('capturing');
  try {
    const fullCanvas = await domToCanvas(canvas, { backgroundColor: '#ffffff' });
    // domToCanvas may scale the output (devicePixelRatio); map CSS px -> canvas px.
    const scale = fullCanvas.width / width;
    const cropHeightPx = Math.min(Math.round(cropHeightCss * scale), fullCanvas.height);

    const cropped = document.createElement('canvas');
    cropped.width = fullCanvas.width;
    cropped.height = cropHeightPx;
    const ctx = cropped.getContext('2d')!;
    ctx.drawImage(fullCanvas, 0, 0, fullCanvas.width, cropHeightPx, 0, 0, fullCanvas.width, cropHeightPx);

    return await new Promise<Blob | null>((resolve) => cropped.toBlob(resolve, 'image/png'));
  } finally {
    canvas.classList.remove('capturing');
  }
};
