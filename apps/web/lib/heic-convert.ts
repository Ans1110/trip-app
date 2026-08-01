export type HeicConvertOptions = {
  quality?: number; // 0..1, forwarded to heic-to; default 0.9
};

export function isHeic(file: { name: string; type: string }): boolean {
  const mime = file.type.toLowerCase();
  if (mime === "image/heic" || mime === "image/heif") return true;
  const name = file.name.toLowerCase();
  return name.endsWith(".heic") || name.endsWith(".heif");
}

export function renameToJpeg(name: string): string {
  const dot = name.lastIndexOf(".");
  if (dot <= 0) return `${name}.jpg`;
  return `${name.slice(0, dot)}.jpg`;
}

export async function convertHeicToJpeg(
  file: File,
  opts: HeicConvertOptions = {},
): Promise<File> {
  const { heicTo } = await import("heic-to");
  const blob = await heicTo({
    blob: file,
    type: "image/jpeg",
    quality: opts.quality ?? 0.9,
  });
  return new File([blob], renameToJpeg(file.name), {
    type: "image/jpeg",
    lastModified: file.lastModified,
  });
}
