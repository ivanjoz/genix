import { encode } from '@jsquash/avif';

console.log('🔧 Image worker loaded and ready')

let nativeAvifSupport: boolean | null = null;

const checkNativeAvifSupport = async () => {
  if (nativeAvifSupport !== null) return nativeAvifSupport;
  try {
    const canvas = new OffscreenCanvas(1, 1);
    const blob = await canvas.convertToBlob({ type: 'image/avif' });
    nativeAvifSupport = blob.type === 'image/avif';
  } catch (e) {
    nativeAvifSupport = false;
  }
  return nativeAvifSupport;
}

self.onmessage = async (event) => {
  const { id, bitmap: inputBitmap, blob: inputBlob, resolution, useJpeg, useAvif } = event.data;
  
  console.log('📨 Worker received message:', { 
    id,
    hasBitmap: !!inputBitmap, 
    hasBlob: !!inputBlob,
    resolution,
    useJpeg,
    useAvif
  })

  try {
    let bitmap: ImageBitmap;
    if (inputBitmap) {
      console.log('🖼️ Processing bitmap...')
      bitmap = inputBitmap;
    } else if (inputBlob) {
      console.log('🖼️ Processing blob...')
      const blob: Blob = inputBlob;
      console.log('🔄 Creating ImageBitmap from blob...')
      bitmap = await createImageBitmap(blob);
      console.log('✅ ImageBitmap created:', bitmap.width, 'x', bitmap.height)
    } else {
      console.error('❌ No bitmap or blob provided in message')
      self.postMessage({ id, error: 'No bitmap or blob provided' });
      return;
    }

    let targetResolution = resolution || 1000
    if(targetResolution > 2500){ targetResolution = 2500 }
    
    const { width, height } = calculateDimensions(bitmap.width, bitmap.height, targetResolution);
    console.log('📏 Calculated dimensions:', { width, height })

    const canvas = new OffscreenCanvas(width, height);
    const ctx = canvas.getContext("2d")!;
    ctx.drawImage(bitmap, 0, 0, width, height);

    let blob: Blob;
    if (useAvif) {
      console.log('🥑 Attempting AVIF conversion...')
      if (await checkNativeAvifSupport()) {
        console.log('🚀 Using native AVIF support')
        blob = await canvas.convertToBlob({ type: 'image/avif', quality: 0.8 });
      } else {
        console.log('📦 Using @jsquash/avif fallback')
        const imageData = ctx.getImageData(0, 0, width, height);
        const avifBuffer = await encode(imageData);
        blob = new Blob([avifBuffer], { type: 'image/avif' });
      }
    } else {
      const fileType = useJpeg ? "image/jpeg" : "image/webp"
      console.log(`🎨 Converting to ${fileType}...`)
      blob = await canvas.convertToBlob({ type: fileType, quality: 0.8 });
    }

    console.log('🎨 Blob created:', blob.size, 'bytes', blob.type)
    const dataUrl = await blobToBase64_(blob);
    console.log('✅ Data URL created, length:', dataUrl.length)

    self.postMessage({ id, dataUrl });
    console.log('📤 Response sent to main thread')
    
    // Close bitmap to free memory
    bitmap.close();
  } catch (error) {
    console.error('❌ Error in worker:', error)
    self.postMessage({ id, error: String(error) })
  }
};

const calculateDimensions = (
  originalWidth: number, originalHeight: number, targetMP: number
): { width: number, height: number } => {
  const aspectRatio = originalWidth / originalHeight;
  const originalPixels = originalWidth * originalHeight
  
  // Convert 1D pixel to 2D pixels
  const targetPixels = targetMP * targetMP
  
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
