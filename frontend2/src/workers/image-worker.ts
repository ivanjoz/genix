self.onmessage = async (event) => {
  if (event.data.bitmap) {
    const bitmap: ImageBitmap = event.data.bitmap;
    const resolutionMP = event.data.resolution || 1; // Default 1MP (1,000,000 pixels)

    const { width, height } = calculateDimensions(bitmap.width, bitmap.height, resolutionMP);
    
    const canvas = new OffscreenCanvas(width, height);
    const ctx = canvas.getContext("2d")!;
    ctx.drawImage(bitmap, 0, 0, width, height);

    const fileType = event.data.useJpeg ? "image/jpeg" : "image/webp"
    const blob = await canvas.convertToBlob({ type: fileType, quality: 0.8 });
    const dataUrl = await blobToBase64_(blob);

    self.postMessage({ dataUrl });
  } else if (event.data.blob) {
    // fallback path if OffscreenCanvas is not supported
    const blob: Blob = event.data.blob;
    const resolutionMP = event.data.resolution || 1; // Default 1MP (1,000,000 pixels)
    const bitmap = await createImageBitmap(blob);

    const { width, height } = calculateDimensions(bitmap.width, bitmap.height, resolutionMP);
    
    const canvas = new OffscreenCanvas(width, height);
    const ctx = canvas.getContext("2d")!;
    ctx.drawImage(bitmap, 0, 0, width, height);

    const fileType = event.data.useJpeg ? "image/jpeg" : "image/webp"
    const resizedBlob = await canvas.convertToBlob({ type: fileType, quality: 0.8 });
    const dataUrl = await blobToBase64_(resizedBlob);

    self.postMessage({ dataUrl });
  }
};

const calculateDimensions = (
  originalWidth: number, originalHeight: number, targetMP: number
): { width: number, height: number } => {
  const aspectRatio = originalWidth / originalHeight;
  const originalPixels = originalWidth * originalHeight
  
  // Convert megapixels to total pixels
  const targetPixels = targetMP * 1000000; // 1MP = 1,000,000 pixels
  
  if(targetPixels > originalPixels){
    return { width: originalWidth, height: originalHeight }
  }

  // Calculate dimensions that maintain aspect ratio and achieve target pixel count
  let width = Math.sqrt(targetPixels * aspectRatio);
  let height = width / aspectRatio;
  
  // Ensure we don't upscale if the original is smaller than target
  if (width > originalWidth) {
    width = originalWidth;
    height = originalHeight;
  }
  
  return { width: Math.round(width), height: Math.round(height) };
}

function blobToBase64_(blob: Blob): Promise<string> {
  return new Promise((resolve) => {
    const reader = new FileReader();
    reader.onloadend = () => resolve(reader.result as string);
    reader.readAsDataURL(blob);
  });
}