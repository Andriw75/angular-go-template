import {
  Component,
  ElementRef,
  HostListener,
  OnDestroy,
  AfterViewInit,
  computed,
  effect,
  inject,
  input,
  output,
  signal,
  untracked,
  viewChild,
} from '@angular/core';
import { ToastService } from '../../services/toast.service';
import {
  type AspectPreset,
  type CropHandle,
  type CropRect,
  type Size,
  applyAspect,
  aspectRatio,
  clamp,
  clampCrop,
  editedFileName,
  extensionForType,
  fitScale,
  initialCrop,
  moveCrop,
  outputDimensions,
  outputTypeFor,
  resizeCrop,
} from './image-editor.utils';

export interface EditorSource {
  url: string;
  fileName: string;
  mimeType: string;
}

export interface EditorResult {
  file: File;
  width: number;
  height: number;
}

interface DragState {
  mode: 'pan' | 'move' | 'resize';
  handle?: Exclude<CropHandle, 'move'>;
  pointerId: number;
  startX: number;
  startY: number;
  startCrop: CropRect;
  startPan: { x: number; y: number };
}

const MAX_ZOOM = 4;
const MAX_OUTPUT_BYTES = 32 << 20;
const PRESETS: AspectPreset[] = ['free', '1:1', '4:3', '16:9'];
const HANDLES: Exclude<CropHandle, 'move'>[] = ['nw', 'n', 'ne', 'e', 'se', 's', 'sw', 'w'];

@Component({
  selector: 'app-image-editor',
  templateUrl: './image-editor.html',
  styleUrl: './image-editor.css',
})
export class ImageEditorComponent implements AfterViewInit, OnDestroy {
  private toast = inject(ToastService);

  source = input.required<EditorSource>();
  initialAspect = input<AspectPreset>('free');
  maxOutput = input(1600);

  applied = output<EditorResult>();
  cancelled = output<void>();

  readonly presets = PRESETS;
  readonly handles = HANDLES;

  private stageRef = viewChild<ElementRef<HTMLDivElement>>('stage');
  private canvasRef = viewChild<ElementRef<HTMLCanvasElement>>('canvas');

  private originalBitmap: ImageBitmap | null = null;
  private workingBitmap: ImageBitmap | null = null;
  private loadToken = 0;
  private resizeObserver: ResizeObserver | null = null;

  ready = signal(false);
  loading = signal(true);
  error = signal('');
  busy = signal(false);

  zoom = signal(1);
  pan = signal({ x: 0, y: 0 });
  imageSize = signal<Size>({ width: 0, height: 0 });
  stageSize = signal<Size>({ width: 0, height: 0 });
  crop = signal<CropRect>({ x: 0, y: 0, width: 0, height: 0 });
  aspectPreset = signal<AspectPreset>('free');
  drag = signal<DragState | null>(null);

  readonly aspect = computed(() => aspectRatio(this.aspectPreset()));

  readonly baseScale = computed(() =>
    fitScale(
      this.imageSize().width,
      this.imageSize().height,
      this.stageSize().width,
      this.stageSize().height,
    ),
  );

  readonly scale = computed(() => this.baseScale() * this.zoom());

  readonly origin = computed(() => {
    const stage = this.stageSize();
    const image = this.imageSize();
    const scale = this.scale();
    const pan = this.pan();
    return {
      x: (stage.width - image.width * scale) / 2 + pan.x,
      y: (stage.height - image.height * scale) / 2 + pan.y,
    };
  });

  readonly displayCrop = computed(() => {
    const crop = this.crop();
    const origin = this.origin();
    const scale = this.scale();
    return {
      left: origin.x + crop.x * scale,
      top: origin.y + crop.y * scale,
      width: crop.width * scale,
      height: crop.height * scale,
    };
  });

  readonly cropDisplay = computed(() => {
    const crop = this.crop();
    return {
      x: Math.round(crop.x),
      y: Math.round(crop.y),
      width: Math.round(crop.width),
      height: Math.round(crop.height),
    };
  });

  readonly outputSize = computed(() => outputDimensions(this.crop(), this.maxOutput()));
  readonly zoomLabel = computed(() => `${Math.round(this.zoom() * 100)}%`);

  constructor() {
    effect(() => {
      const source = this.source();
      untracked(() => void this.load(source));
    });

    effect(() => {
      this.canvasRef();
      this.ready();
      this.zoom();
      this.pan();
      this.stageSize();
      this.imageSize();
      untracked(() => this.draw());
    });
  }

  ngAfterViewInit(): void {
    const stage = this.stageRef()?.nativeElement;
    if (!stage) return;
    this.updateStageSize(stage);
    if (typeof ResizeObserver === 'undefined') return;
    this.resizeObserver = new ResizeObserver(() => this.updateStageSize(stage));
    this.resizeObserver.observe(stage);
  }

  ngOnDestroy(): void {
    this.resizeObserver?.disconnect();
    this.loadToken++;
    const original = this.originalBitmap;
    const working = this.workingBitmap;
    if (working && working !== original) working.close();
    original?.close();
  }

  private updateStageSize(stage: HTMLElement): void {
    const rect = stage.getBoundingClientRect();
    const width = Math.round(rect.width);
    const height = Math.round(rect.height);
    if (width === this.stageSize().width && height === this.stageSize().height) return;
    this.stageSize.set({ width, height });
    this.pan.set(this.clampPan(this.pan().x, this.pan().y));
  }

  private async load(source: EditorSource): Promise<void> {
    const token = ++this.loadToken;
    this.loading.set(true);
    this.error.set('');
    this.ready.set(false);

    try {
      const response = await fetch(source.url, { credentials: 'include' });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const blob = await response.blob();
      const bitmap = await createImageBitmap(blob, { imageOrientation: 'from-image' });
      if (token !== this.loadToken) {
        bitmap.close();
        return;
      }

      const previousOriginal = this.originalBitmap;
      const previousWorking = this.workingBitmap;
      this.originalBitmap = bitmap;
      this.workingBitmap = bitmap;
      if (previousWorking && previousWorking !== previousOriginal) previousWorking.close();
      previousOriginal?.close();

      this.imageSize.set({ width: bitmap.width, height: bitmap.height });
      this.aspectPreset.set(this.initialAspect());
      this.resetView();
      this.ready.set(true);
    } catch {
      if (token === this.loadToken) this.error.set('No se pudo cargar la imagen para editar.');
    } finally {
      if (token === this.loadToken) this.loading.set(false);
    }
  }

  private resetView(): void {
    this.zoom.set(1);
    this.pan.set({ x: 0, y: 0 });
    const image = this.imageSize();
    this.crop.set(initialCrop(image.width, image.height, this.aspect()));
  }

  private clampPan(x: number, y: number): { x: number; y: number } {
    const stage = this.stageSize();
    const image = this.imageSize();
    const scale = this.scale();
    const maxX = Math.max(0, (image.width * scale - stage.width) / 2);
    const maxY = Math.max(0, (image.height * scale - stage.height) / 2);
    return { x: clamp(x, -maxX, maxX), y: clamp(y, -maxY, maxY) };
  }

  private draw(): void {
    const canvas = this.canvasRef()?.nativeElement;
    const bitmap = this.workingBitmap;
    const stage = this.stageSize();
    const image = this.imageSize();
    if (!canvas || !bitmap || !this.ready() || !stage.width || !stage.height || !image.width)
      return;

    const dpr = Math.min(window.devicePixelRatio || 1, 2);
    canvas.width = Math.round(stage.width * dpr);
    canvas.height = Math.round(stage.height * dpr);
    canvas.style.width = `${stage.width}px`;
    canvas.style.height = `${stage.height}px`;

    const context = canvas.getContext('2d');
    if (!context) return;
    context.setTransform(dpr, 0, 0, dpr, 0, 0);
    context.clearRect(0, 0, stage.width, stage.height);
    context.imageSmoothingEnabled = true;
    context.imageSmoothingQuality = 'high';

    const origin = this.origin();
    const scale = this.scale();
    context.drawImage(bitmap, origin.x, origin.y, image.width * scale, image.height * scale);
  }

  private beginDrag(
    event: PointerEvent,
    mode: DragState['mode'],
    handle?: Exclude<CropHandle, 'move'>,
  ): void {
    if (!this.ready() || this.busy() || event.button !== 0) return;
    event.preventDefault();
    event.stopPropagation();

    const stage = this.stageRef()?.nativeElement;
    if (!stage) return;
    try {
      stage.setPointerCapture(event.pointerId);
    } catch {
      // Algunos navegadores no permiten capturar el puntero; el arrastre sigue funcionando.
    }

    this.drag.set({
      mode,
      handle,
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      startCrop: this.crop(),
      startPan: this.pan(),
    });
  }

  onStagePointerDown(event: PointerEvent): void {
    this.beginDrag(event, 'pan');
  }

  onCropPointerDown(event: PointerEvent): void {
    this.beginDrag(event, 'move');
  }

  onHandlePointerDown(event: PointerEvent, handle: Exclude<CropHandle, 'move'>): void {
    this.beginDrag(event, 'resize', handle);
  }

  onStagePointerMove(event: PointerEvent): void {
    const drag = this.drag();
    if (!drag || drag.pointerId !== event.pointerId) return;
    event.preventDefault();

    const dx = event.clientX - drag.startX;
    const dy = event.clientY - drag.startY;

    if (drag.mode === 'pan') {
      this.pan.set(this.clampPan(drag.startPan.x + dx, drag.startPan.y + dy));
      return;
    }

    const scale = this.scale();
    if (scale <= 0) return;
    const image = this.imageSize();
    const imageDx = dx / scale;
    const imageDy = dy / scale;

    if (drag.mode === 'move') {
      this.crop.set(moveCrop(drag.startCrop, imageDx, imageDy, image.width, image.height));
    } else if (drag.handle) {
      this.crop.set(
        resizeCrop(
          drag.handle,
          drag.startCrop,
          imageDx,
          imageDy,
          this.aspect(),
          image.width,
          image.height,
        ),
      );
    }
  }

  onStagePointerUp(event: PointerEvent): void {
    const drag = this.drag();
    if (!drag || drag.pointerId !== event.pointerId) return;
    this.drag.set(null);
    const stage = this.stageRef()?.nativeElement;
    if (stage?.hasPointerCapture(event.pointerId)) stage.releasePointerCapture(event.pointerId);
  }

  onWheel(event: WheelEvent): void {
    if (!this.ready()) return;
    event.preventDefault();
    const stage = event.currentTarget as HTMLElement;
    const rect = stage.getBoundingClientRect();
    const factor = event.deltaY < 0 ? 1.15 : 1 / 1.15;
    this.setZoom(this.zoom() * factor, event.clientX - rect.left, event.clientY - rect.top);
  }

  private setZoom(next: number, anchorX?: number, anchorY?: number): void {
    if (!this.ready()) return;
    const target = clamp(next, 1, MAX_ZOOM);
    const previous = this.zoom();
    if (Math.abs(target - previous) < 0.001) return;

    const stage = this.stageSize();
    const image = this.imageSize();
    const base = this.baseScale();
    if (!stage.width || !stage.height || !image.width || !base) return;

    const fromX = anchorX ?? stage.width / 2;
    const fromY = anchorY ?? stage.height / 2;
    const previousScale = base * previous;
    const previousOrigin = this.origin();
    const imageX = (fromX - previousOrigin.x) / previousScale;
    const imageY = (fromY - previousOrigin.y) / previousScale;

    this.zoom.set(target);

    const scale = base * target;
    const originX = fromX - imageX * scale;
    const originY = fromY - imageY * scale;
    const centeredX = (stage.width - image.width * scale) / 2;
    const centeredY = (stage.height - image.height * scale) / 2;
    this.pan.set(this.clampPan(originX - centeredX, originY - centeredY));
  }

  onZoomInput(event: Event): void {
    const value = Number((event.target as HTMLInputElement).value);
    if (Number.isFinite(value)) this.setZoom(value);
  }

  setPreset(preset: AspectPreset): void {
    if (!this.ready()) return;
    this.aspectPreset.set(preset);
    const image = this.imageSize();
    this.crop.set(applyAspect(this.crop(), aspectRatio(preset), image.width, image.height));
  }

  onCoordinateChange(key: 'x' | 'y' | 'width' | 'height', event: Event): void {
    if (!this.ready()) return;
    const value = Number((event.target as HTMLInputElement).value);
    if (!Number.isFinite(value)) return;

    const image = this.imageSize();
    const current = this.crop();
    let next: CropRect;

    switch (key) {
      case 'x':
        next = { ...current, x: Math.round(value) };
        break;
      case 'y':
        next = { ...current, y: Math.round(value) };
        break;
      case 'width':
        next = { ...current, width: Math.max(1, Math.round(value)) };
        break;
      case 'height':
        next = { ...current, height: Math.max(1, Math.round(value)) };
        break;
    }

    const aspect = this.aspect();
    if (aspect && (key === 'width' || key === 'height')) {
      next = applyAspect(next, aspect, image.width, image.height);
    } else {
      next = clampCrop(next, image.width, image.height);
    }
    this.crop.set(next);
  }

  reset(): void {
    if (!this.originalBitmap || this.busy()) return;
    if (this.workingBitmap && this.workingBitmap !== this.originalBitmap)
      this.workingBitmap.close();
    this.workingBitmap = this.originalBitmap;
    this.imageSize.set({ width: this.originalBitmap.width, height: this.originalBitmap.height });
    this.resetView();
  }

  async rotate(): Promise<void> {
    const bitmap = this.workingBitmap;
    if (!bitmap || this.busy()) return;
    this.busy.set(true);

    try {
      const image = this.imageSize();
      const canvas = document.createElement('canvas');
      canvas.width = image.height;
      canvas.height = image.width;
      const context = canvas.getContext('2d');
      if (!context) throw new Error('canvas no disponible');

      context.translate(image.height / 2, image.width / 2);
      context.rotate(Math.PI / 2);
      context.drawImage(bitmap, -image.width / 2, -image.height / 2);

      const rotated = await createImageBitmap(canvas);
      if (this.workingBitmap && this.workingBitmap !== this.originalBitmap)
        this.workingBitmap.close();
      this.workingBitmap = rotated;
      this.imageSize.set({ width: rotated.width, height: rotated.height });
      this.resetView();
    } catch {
      this.toast.error('No se pudo rotar la imagen.');
    } finally {
      this.busy.set(false);
    }
  }

  async apply(): Promise<void> {
    const bitmap = this.workingBitmap;
    if (!bitmap || this.busy()) return;
    this.busy.set(true);

    try {
      const crop = clampCrop(this.crop(), bitmap.width, bitmap.height);
      const dimensions = outputDimensions(crop, this.maxOutput());
      const type = outputTypeFor(this.source().mimeType);

      const canvas = document.createElement('canvas');
      canvas.width = dimensions.width;
      canvas.height = dimensions.height;
      const context = canvas.getContext('2d');
      if (!context) throw new Error('canvas no disponible');

      if (type === 'image/jpeg') {
        context.fillStyle = '#ffffff';
        context.fillRect(0, 0, dimensions.width, dimensions.height);
      }
      context.imageSmoothingEnabled = true;
      context.imageSmoothingQuality = 'high';
      const sx = Math.round(crop.x);
      const sy = Math.round(crop.y);
      const sw = Math.max(1, Math.min(Math.round(crop.width), bitmap.width - sx));
      const sh = Math.max(1, Math.min(Math.round(crop.height), bitmap.height - sy));
      context.drawImage(bitmap, sx, sy, sw, sh, 0, 0, dimensions.width, dimensions.height);

      let blob = await canvasToBlob(canvas, type, 0.9);
      if (blob.size > MAX_OUTPUT_BYTES) blob = await canvasToBlob(canvas, type, 0.7);
      if (blob.size > MAX_OUTPUT_BYTES) throw new Error('imagen demasiado grande');

      const actualType = blob.type || type;
      const file = new File(
        [blob],
        editedFileName(this.source().fileName, extensionForType(actualType)),
        {
          type: actualType,
          lastModified: Date.now(),
        },
      );
      this.applied.emit({ file, width: dimensions.width, height: dimensions.height });
    } catch {
      this.toast.error('No se pudo generar la imagen editada.');
    } finally {
      this.busy.set(false);
    }
  }

  cancel(): void {
    if (this.busy()) return;
    this.cancelled.emit();
  }

  isPresetActive(preset: AspectPreset): boolean {
    return this.aspectPreset() === preset;
  }

  presetLabel(preset: AspectPreset): string {
    return preset === 'free' ? 'Libre' : preset;
  }

  @HostListener('document:keydown', ['$event'])
  onKeydown(event: KeyboardEvent): void {
    if (event.key === 'Escape') {
      event.preventDefault();
      this.cancel();
      return;
    }

    const target = event.target as HTMLElement | null;
    const typing = target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA';

    if (event.key === 'Enter' && !typing) {
      event.preventDefault();
      void this.apply();
      return;
    }

    if (!typing && event.key.startsWith('Arrow') && this.ready()) {
      event.preventDefault();
      const step = event.shiftKey ? 10 : 1;
      const dx = event.key === 'ArrowLeft' ? -step : event.key === 'ArrowRight' ? step : 0;
      const dy = event.key === 'ArrowUp' ? -step : event.key === 'ArrowDown' ? step : 0;
      const image = this.imageSize();
      this.crop.set(moveCrop(this.crop(), dx, dy, image.width, image.height));
    }
  }
}

function canvasToBlob(canvas: HTMLCanvasElement, type: string, quality: number): Promise<Blob> {
  return new Promise((resolve, reject) => {
    try {
      canvas.toBlob(
        (blob) => (blob ? resolve(blob) : reject(new Error('toBlob devolvió null'))),
        type,
        quality,
      );
    } catch (error) {
      reject(error instanceof Error ? error : new Error('toBlob falló'));
    }
  });
}
