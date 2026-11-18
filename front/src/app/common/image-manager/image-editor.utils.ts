export interface CropRect {
  x: number;
  y: number;
  width: number;
  height: number;
}

export interface Size {
  width: number;
  height: number;
}

export type CropHandle = 'move' | 'n' | 'ne' | 'e' | 'se' | 's' | 'sw' | 'w' | 'nw';

export type AspectPreset = 'free' | '1:1' | '4:3' | '16:9';

export const MIN_CROP_SIZE = 48;

export const ASPECT_RATIOS: Record<Exclude<AspectPreset, 'free'>, number> = {
  '1:1': 1,
  '4:3': 4 / 3,
  '16:9': 16 / 9,
};

export function aspectRatio(preset: AspectPreset): number | null {
  if (preset === 'free') return null;
  return ASPECT_RATIOS[preset];
}

export function clamp(value: number, min: number, max: number): number {
  if (max < min) return min;
  return Math.min(max, Math.max(min, value));
}

export function fitScale(imgW: number, imgH: number, stageW: number, stageH: number): number {
  if (imgW <= 0 || imgH <= 0 || stageW <= 0 || stageH <= 0) return 1;
  return Math.min(stageW / imgW, stageH / imgH);
}

export function rotateSize(size: Size, degrees: number): Size {
  const normalized = ((degrees % 360) + 360) % 360;
  if (normalized === 90 || normalized === 270) {
    return { width: size.height, height: size.width };
  }
  return { ...size };
}

function fitSizeToBounds(
  width: number,
  height: number,
  aspect: number,
  boundsW: number,
  boundsH: number,
): Size {
  let w = Math.min(width, boundsW);
  let h = w / aspect;
  if (h > boundsH) {
    h = boundsH;
    w = h * aspect;
  }
  return { width: Math.max(1, w), height: Math.max(1, h) };
}

export function clampCrop(rect: CropRect, boundsW: number, boundsH: number): CropRect {
  const width = clamp(rect.width, Math.min(MIN_CROP_SIZE, boundsW), boundsW);
  const height = clamp(rect.height, Math.min(MIN_CROP_SIZE, boundsH), boundsH);
  return {
    x: clamp(rect.x, 0, Math.max(0, boundsW - width)),
    y: clamp(rect.y, 0, Math.max(0, boundsH - height)),
    width,
    height,
  };
}

export function initialCrop(boundsW: number, boundsH: number, aspect: number | null): CropRect {
  if (!aspect || aspect <= 0) {
    return { x: 0, y: 0, width: boundsW, height: boundsH };
  }
  const fitted = fitSizeToBounds(boundsW, boundsH, aspect, boundsW, boundsH);
  return {
    x: (boundsW - fitted.width) / 2,
    y: (boundsH - fitted.height) / 2,
    width: fitted.width,
    height: fitted.height,
  };
}

export function applyAspect(
  rect: CropRect,
  aspect: number | null,
  boundsW: number,
  boundsH: number,
): CropRect {
  if (!aspect || aspect <= 0) return clampCrop(rect, boundsW, boundsH);

  const centerX = rect.x + rect.width / 2;
  const centerY = rect.y + rect.height / 2;

  let width = rect.width;
  let height = width / aspect;
  if (height > rect.height) {
    height = rect.height;
    width = height * aspect;
  }

  const fitted = fitSizeToBounds(width, height, aspect, boundsW, boundsH);
  return clampCrop(
    {
      x: centerX - fitted.width / 2,
      y: centerY - fitted.height / 2,
      width: fitted.width,
      height: fitted.height,
    },
    boundsW,
    boundsH,
  );
}

export function moveCrop(
  rect: CropRect,
  dx: number,
  dy: number,
  boundsW: number,
  boundsH: number,
): CropRect {
  return {
    x: clamp(rect.x + dx, 0, Math.max(0, boundsW - rect.width)),
    y: clamp(rect.y + dy, 0, Math.max(0, boundsH - rect.height)),
    width: rect.width,
    height: rect.height,
  };
}

export function resizeCrop(
  handle: Exclude<CropHandle, 'move'>,
  rect: CropRect,
  dx: number,
  dy: number,
  aspect: number | null,
  boundsW: number,
  boundsH: number,
): CropRect {
  const min = Math.min(MIN_CROP_SIZE, boundsW, boundsH);

  let left = rect.x;
  let top = rect.y;
  let right = rect.x + rect.width;
  let bottom = rect.y + rect.height;

  if (handle.includes('w')) left = clamp(left + dx, 0, right - min);
  if (handle.includes('e')) right = clamp(right + dx, left + min, boundsW);
  if (handle.includes('n')) top = clamp(top + dy, 0, bottom - min);
  if (handle.includes('s')) bottom = clamp(bottom + dy, top + min, boundsH);

  if (!aspect || aspect <= 0) {
    return clampCrop(
      { x: left, y: top, width: right - left, height: bottom - top },
      boundsW,
      boundsH,
    );
  }

  let width = right - left;
  let height = bottom - top;
  if (width / height > aspect) height = width / aspect;
  else width = height * aspect;

  const fitted = fitSizeToBounds(width, height, aspect, boundsW, boundsH);
  width = Math.max(fitted.width, min);
  height = width / aspect;
  if (height < min) {
    height = Math.min(min, boundsH);
    width = height * aspect;
  }
  if (width > boundsW) {
    width = boundsW;
    height = width / aspect;
  }
  if (height > boundsH) {
    height = boundsH;
    width = height * aspect;
  }

  let x: number;
  if (handle.includes('w')) x = rect.x + rect.width - width;
  else if (handle.includes('e')) x = rect.x;
  else x = rect.x + (rect.width - width) / 2;

  let y: number;
  if (handle.includes('n')) y = rect.y + rect.height - height;
  else if (handle.includes('s')) y = rect.y;
  else y = rect.y + (rect.height - height) / 2;

  return {
    x: clamp(x, 0, Math.max(0, boundsW - width)),
    y: clamp(y, 0, Math.max(0, boundsH - height)),
    width,
    height,
  };
}

export function outputDimensions(crop: CropRect, maxOutput: number): Size {
  const longest = Math.max(crop.width, crop.height);
  const scale = longest > maxOutput ? maxOutput / longest : 1;
  return {
    width: Math.max(1, Math.round(crop.width * scale)),
    height: Math.max(1, Math.round(crop.height * scale)),
  };
}

export function outputTypeFor(mimeType: string): string {
  const mime = mimeType.toLowerCase();
  if (mime === 'image/png') return 'image/png';
  if (mime === 'image/webp') return 'image/webp';
  return 'image/jpeg';
}

export function extensionForType(mimeType: string): string {
  if (mimeType === 'image/png') return 'png';
  if (mimeType === 'image/webp') return 'webp';
  return 'jpg';
}

export function mimeForName(name: string): string {
  const ext = name.slice(name.lastIndexOf('.') + 1).toLowerCase();
  switch (ext) {
    case 'png':
      return 'image/png';
    case 'webp':
      return 'image/webp';
    case 'gif':
      return 'image/gif';
    default:
      return 'image/jpeg';
  }
}

export function isGifName(name: string): boolean {
  return /\.gif$/i.test(name);
}

export function editedFileName(name: string, extension: string): string {
  const base = (name.split(/[\\/]/).pop() ?? name).replace(/\.[^.]+$/, '');
  const safe = base.replace(/[^\w.-]+/g, '-').replace(/^-+|-+$/g, '');
  return `${safe || 'imagen'}-edit.${extension}`;
}
