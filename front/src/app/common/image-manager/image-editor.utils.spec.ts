import { describe, expect, it } from 'vitest';
import {
  applyAspect,
  clampCrop,
  editedFileName,
  extensionForType,
  initialCrop,
  mimeForName,
  moveCrop,
  outputDimensions,
  outputTypeFor,
  resizeCrop,
  rotateSize,
} from './image-editor.utils';

describe('clampCrop', () => {
  it('ajusta tamaño y posición a los límites', () => {
    expect(clampCrop({ x: -10, y: -10, width: 2000, height: 2000 }, 800, 600)).toEqual({
      x: 0,
      y: 0,
      width: 800,
      height: 600,
    });
  });

  it('respeta el tamaño mínimo en imágenes pequeñas', () => {
    const rect = clampCrop({ x: 0, y: 0, width: 10, height: 10 }, 30, 20);
    expect(rect.width).toBe(30);
    expect(rect.height).toBe(20);
  });
});

describe('initialCrop', () => {
  it('usa toda la imagen en modo libre', () => {
    expect(initialCrop(400, 300, null)).toEqual({ x: 0, y: 0, width: 400, height: 300 });
  });

  it('centra el mayor rectángulo con la proporción pedida', () => {
    expect(initialCrop(400, 100, 1)).toEqual({ x: 150, y: 0, width: 100, height: 100 });
  });
});

describe('applyAspect', () => {
  it('reduce el lado sobrante manteniendo el centro', () => {
    const rect = applyAspect({ x: 0, y: 0, width: 400, height: 100 }, 1, 400, 100);
    expect(rect).toEqual({ x: 150, y: 0, width: 100, height: 100 });
  });

  it('no modifica nada en modo libre', () => {
    const original = { x: 10, y: 20, width: 100, height: 50 };
    expect(applyAspect(original, null, 400, 400)).toEqual(original);
  });
});

describe('moveCrop', () => {
  it('mueve dentro de los límites', () => {
    expect(moveCrop({ x: 0, y: 0, width: 100, height: 100 }, 50, 50, 300, 300)).toEqual({
      x: 50,
      y: 50,
      width: 100,
      height: 100,
    });
    expect(moveCrop({ x: 200, y: 200, width: 100, height: 100 }, 50, 50, 300, 300)).toEqual({
      x: 200,
      y: 200,
      width: 100,
      height: 100,
    });
  });
});

describe('resizeCrop', () => {
  it('redimensiona desde el borde este', () => {
    const rect = resizeCrop(
      'e',
      { x: 100, y: 100, width: 200, height: 200 },
      50,
      0,
      null,
      1000,
      1000,
    );
    expect(rect).toEqual({ x: 100, y: 100, width: 250, height: 200 });
  });

  it('no baja del tamaño mínimo', () => {
    const rect = resizeCrop(
      'e',
      { x: 0, y: 0, width: 200, height: 200 },
      -500,
      0,
      null,
      1000,
      1000,
    );
    expect(rect.width).toBe(48);
  });

  it('mantiene la esquina opuesta fija con proporción', () => {
    const rect = resizeCrop('se', { x: 0, y: 0, width: 200, height: 100 }, 0, 0, 1, 1000, 1000);
    expect(rect).toEqual({ x: 0, y: 0, width: 200, height: 200 });
  });

  it('ancla el borde izquierdo al arrastrar el handle este con proporción', () => {
    const rect = resizeCrop('e', { x: 100, y: 100, width: 200, height: 200 }, 50, 0, 1, 1000, 1000);
    expect(rect).toEqual({ x: 100, y: 75, width: 250, height: 250 });
  });

  it('ancla el borde derecho y centra en vertical al arrastrar el handle oeste', () => {
    const rect = resizeCrop(
      'w',
      { x: 100, y: 100, width: 200, height: 200 },
      -50,
      0,
      1,
      1000,
      1000,
    );
    expect(rect).toEqual({ x: 50, y: 75, width: 250, height: 250 });
  });

  it('respeta los límites de la imagen al crecer con proporción', () => {
    const rect = resizeCrop(
      'se',
      { x: 0, y: 0, width: 100, height: 100 },
      900,
      900,
      16 / 9,
      200,
      200,
    );
    expect(rect.width).toBeLessThanOrEqual(200);
    expect(rect.height).toBeLessThanOrEqual(200);
    expect(rect.width / rect.height).toBeCloseTo(16 / 9, 5);
  });
});

describe('outputDimensions', () => {
  it('no amplía imágenes menores al límite', () => {
    expect(outputDimensions({ x: 0, y: 0, width: 800, height: 600 }, 1600)).toEqual({
      width: 800,
      height: 600,
    });
  });

  it('escala el lado mayor al límite', () => {
    expect(outputDimensions({ x: 0, y: 0, width: 3200, height: 1600 }, 1600)).toEqual({
      width: 1600,
      height: 800,
    });
  });
});

describe('rotateSize', () => {
  it('intercambia dimensiones en 90 y 270 grados', () => {
    expect(rotateSize({ width: 800, height: 600 }, 90)).toEqual({ width: 600, height: 800 });
    expect(rotateSize({ width: 800, height: 600 }, 270)).toEqual({ width: 600, height: 800 });
  });

  it('mantiene dimensiones en 180 grados', () => {
    expect(rotateSize({ width: 800, height: 600 }, 180)).toEqual({ width: 800, height: 600 });
  });
});

describe('formato de salida', () => {
  it('conserva PNG y WebP y usa JPEG para el resto', () => {
    expect(outputTypeFor('image/png')).toBe('image/png');
    expect(outputTypeFor('image/webp')).toBe('image/webp');
    expect(outputTypeFor('image/gif')).toBe('image/jpeg');
    expect(extensionForType('image/png')).toBe('png');
    expect(extensionForType('image/webp')).toBe('webp');
    expect(extensionForType('image/jpeg')).toBe('jpg');
  });

  it('deduce el mime desde el nombre', () => {
    expect(mimeForName('foto.PNG')).toBe('image/png');
    expect(mimeForName('foto.webp')).toBe('image/webp');
    expect(mimeForName('foto.jpeg')).toBe('image/jpeg');
    expect(mimeForName('foto.sin-extension')).toBe('image/jpeg');
  });

  it('genera un nombre seguro para el archivo editado', () => {
    expect(editedFileName('foto final.JPG', 'jpg')).toBe('foto-final-edit.jpg');
    expect(editedFileName('..\\ruta\\imagen.png', 'png')).toBe('imagen-edit.png');
    expect(editedFileName('$$$.png', 'png')).toBe('imagen-edit.png');
  });
});
