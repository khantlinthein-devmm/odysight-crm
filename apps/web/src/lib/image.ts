// Shrinks phone photos before upload: a 12 MP camera shot (3-8 MB) becomes a
// ~1920 px JPEG of a few hundred KB, which uploads quickly on mobile data and
// stays well under the server's size limit. Anything that cannot be decoded
// or re-encoded here is sent unchanged.

export const MAX_PHOTO_EDGE = 1920;
export const PHOTO_QUALITY = 0.82;
/** Small images are left alone. */
export const COMPRESS_ABOVE_BYTES = 600 * 1024;

/** Target size that keeps the aspect ratio and fits within `maxEdge`. */
export function scaledSize(width: number, height: number, maxEdge = MAX_PHOTO_EDGE): { width: number; height: number } {
  const longest = Math.max(width, height);
  if (longest <= maxEdge) return { width, height };
  const k = maxEdge / longest;
  return { width: Math.round(width * k), height: Math.round(height * k) };
}

const COMPRESSIBLE = new Set(["image/jpeg", "image/png", "image/webp", "image/heic", "image/heif"]);

export async function compressPhoto(file: File): Promise<File> {
  try {
    if (!COMPRESSIBLE.has(file.type) || file.size <= COMPRESS_ABOVE_BYTES) return file;
    if (typeof createImageBitmap !== "function" || typeof document === "undefined") return file;
    // from-image applies the camera's EXIF rotation, so photos stay upright.
    const bitmap = await createImageBitmap(file, { imageOrientation: "from-image" });
    const { width, height } = scaledSize(bitmap.width, bitmap.height);
    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const ctx = canvas.getContext("2d");
    if (!ctx) return file;
    ctx.drawImage(bitmap, 0, 0, width, height);
    bitmap.close?.();
    const blob = await new Promise<Blob | null>((resolve) => canvas.toBlob(resolve, "image/jpeg", PHOTO_QUALITY));
    if (!blob || blob.size >= file.size) return file;
    const name = file.name.replace(/\.[^.]+$/, "") + ".jpg";
    return new File([blob], name, { type: "image/jpeg", lastModified: file.lastModified });
  } catch {
    return file;
  }
}
