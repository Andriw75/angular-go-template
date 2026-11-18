import { Component, effect, inject, input, OnDestroy, output, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import type { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import type { Imagen } from '../../models/imagen';
import { ToastService } from '../../services/toast.service';
import { ImageEditorComponent, type EditorResult, type EditorSource } from './image-editor';
import { isGifName, mimeForName } from './image-editor.utils';

export type CardStatus = 'uploaded' | 'pending-upload' | 'pending-delete';

export interface ImagenCard {
  key: number;
  id: number | null;
  url: string;
  file?: File;
  status: CardStatus;
  serverPath?: string;
  replacedByKey?: number;
}

export interface ImagenesApi {
  uploadImagenes(id: number, files: File[]): Observable<Imagen[]>;
  deleteImagen(id: number, imagenId: number): Observable<void>;
  reorderImagenes(id: number, ids: number[]): Observable<Imagen[]>;
  getByID(id: number): Observable<{ imagenes: Imagen[] }>;
}

const STATUS_LABEL: Record<CardStatus, string> = {
  uploaded: 'En servidor',
  'pending-upload': 'Por subir',
  'pending-delete': 'Por eliminar',
};

@Component({
  selector: 'app-image-manager',
  templateUrl: './image-manager.html',
  styleUrl: './image-manager.css',
  imports: [ImageEditorComponent],
})
export class ImageManagerComponent implements OnDestroy {
  private toast = inject(ToastService);

  entityId = input.required<number>();
  imagenes = input<Imagen[]>([]);
  api = input.required<ImagenesApi>();

  imagesApplied = output<void>();

  cards = signal<ImagenCard[]>([]);
  applying = signal(false);
  draggingKey = signal<number | null>(null);
  zoneActive = signal(false);
  editor = signal<{ card: ImagenCard; source: EditorSource } | null>(null);
  editedKeys = signal<ReadonlySet<number>>(new Set());

  private nextKey = 1;
  private prevCards: ImagenCard[] = [];
  private serverOrderIds: number[] = [];

  constructor() {
    effect(() => {
      const imagenes = this.imagenes();
      this.revokeLocalUrls(this.prevCards);
      const cards = this.toCards(imagenes);
      this.prevCards = cards;
      this.cards.set(cards);
      this.editedKeys.set(new Set());
      this.serverOrderIds = cards.filter((card) => card.id != null).map((card) => card.id!);
    });
  }

  ngOnDestroy(): void {
    this.revokeLocalUrls(this.cards());
  }

  private toCards(imagenes: Imagen[]): ImagenCard[] {
    return [...imagenes]
      .sort((a, b) => a.orden - b.orden)
      .map((img) => ({
        key: this.nextKey++,
        id: img.id,
        // Públicas: nginx (MEDIA_URL). Restringidas: API (valida estado/permiso).
        url: (img.publico ? environment.MEDIA_URL : environment.API_URL) + img.url,
        serverPath: img.url,
        status: 'uploaded' as CardStatus,
      }));
  }

  private revokeLocalUrls(cards: ImagenCard[]): void {
    for (const c of cards) {
      if (c.file && c.url.startsWith('blob:')) {
        URL.revokeObjectURL(c.url);
      }
    }
  }

  onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const files = input.files ? Array.from(input.files) : [];
    if (!files.length) return;

    const newCards: ImagenCard[] = files.map((f) => ({
      key: this.nextKey++,
      id: null,
      url: URL.createObjectURL(f),
      file: f,
      status: 'pending-upload',
    }));
    this.cards.update((list) => [...list, ...newCards]);
    input.value = '';
  }

  removeOrRestore(card: ImagenCard): void {
    if (card.status === 'pending-delete' && card.replacedByKey != null) {
      const replacement = this.cards().find((c) => c.key === card.replacedByKey);
      if (replacement?.status === 'pending-upload') {
        if (replacement.url.startsWith('blob:')) URL.revokeObjectURL(replacement.url);
        this.cards.update((list) =>
          list
            .filter((c) => c.key !== replacement.key)
            .map((c) =>
              c.key === card.key
                ? { ...c, status: 'uploaded' as CardStatus, replacedByKey: undefined }
                : c,
            ),
        );
        this.dropEditedKeys(card.key, replacement.key);
        return;
      }
    }

    this.cards.update((list) => {
      if (card.status === 'pending-upload') {
        if (card.url.startsWith('blob:')) URL.revokeObjectURL(card.url);
        return list
          .filter((c) => c.key !== card.key)
          .map((c) => (c.replacedByKey === card.key ? { ...c, replacedByKey: undefined } : c));
      }
      if (card.status === 'uploaded') {
        return list.map((c) =>
          c.key === card.key ? { ...c, status: 'pending-delete' as CardStatus } : c,
        );
      }
      return list.map((c) =>
        c.key === card.key
          ? { ...c, status: 'uploaded' as CardStatus, replacedByKey: undefined }
          : c,
      );
    });

    if (card.status === 'pending-upload') this.dropEditedKeys(card.key);
  }

  private dropEditedKeys(...keys: number[]): void {
    this.editedKeys.update((set) => {
      const next = new Set(set);
      for (const key of keys) next.delete(key);
      return next;
    });
  }

  statusLabel(card: ImagenCard): string {
    if (card.status === 'pending-delete' && card.replacedByKey != null) return 'Por reemplazar';
    return STATUS_LABEL[card.status];
  }

  openEditor(card: ImagenCard): void {
    if (this.applying() || card.status === 'pending-delete') return;

    const fileName = card.file?.name ?? this.fileNameFromPath(card.serverPath);
    if (isGifName(fileName)) {
      this.toast.info('Las imágenes GIF no se pueden editar.');
      return;
    }

    const source: EditorSource = {
      url: card.file ? card.url : environment.API_URL + (card.serverPath ?? ''),
      fileName,
      mimeType: card.file?.type || mimeForName(fileName),
    };
    this.editor.set({ card, source });
  }

  onEditorApplied(result: EditorResult): void {
    const current = this.editor();
    if (!current) return;

    const card = this.cards().find((c) => c.key === current.card.key) ?? current.card;
    const url = URL.createObjectURL(result.file);

    if (card.status === 'pending-upload') {
      if (card.url.startsWith('blob:')) URL.revokeObjectURL(card.url);
      this.cards.update((list) =>
        list.map((c) =>
          c.key === card.key ? { ...c, file: result.file, url, serverPath: undefined } : c,
        ),
      );
      this.editedKeys.update((set) => new Set(set).add(card.key));
    } else {
      const newKey = this.nextKey++;
      this.cards.update((list) => {
        const index = list.findIndex((c) => c.key === card.key);
        if (index < 0) return list;
        const next = [...list];
        next.splice(
          index,
          1,
          { ...list[index], status: 'pending-delete' as CardStatus, replacedByKey: newKey },
          {
            key: newKey,
            id: null,
            url,
            file: result.file,
            status: 'pending-upload' as CardStatus,
          },
        );
        return next;
      });
      this.editedKeys.update((set) => new Set(set).add(newKey));
    }

    this.editor.set(null);
  }

  onEditorCancelled(): void {
    this.editor.set(null);
  }

  private fileNameFromPath(path: string | undefined): string {
    if (!path) return 'imagen';
    const clean = path.split('?')[0];
    return clean.slice(clean.lastIndexOf('/') + 1) || 'imagen';
  }

  onCardDragStart(event: DragEvent, card: ImagenCard): void {
    this.draggingKey.set(card.key);
    event.dataTransfer!.effectAllowed = 'move';
    event.dataTransfer!.setData('text/plain', String(card.key));
  }

  onCardDragOver(event: DragEvent): void {
    event.preventDefault();
    event.dataTransfer!.dropEffect = 'move';
  }

  onCardDrop(event: DragEvent, target: ImagenCard): void {
    event.preventDefault();
    const from = this.draggingKey();
    this.draggingKey.set(null);
    if (from == null) return;

    this.cards.update((list) => {
      const fromIdx = list.findIndex((c) => c.key === from);
      const toIdx = list.findIndex((c) => c.key === target.key);
      if (fromIdx < 0 || toIdx < 0 || fromIdx === toIdx) return list;
      const arr = [...list];
      const [moved] = arr.splice(fromIdx, 1);
      arr.splice(toIdx, 0, moved);
      return arr;
    });
  }

  onCardDragEnd(): void {
    this.draggingKey.set(null);
  }

  onZoneDragOver(event: DragEvent): void {
    if (this.applying() || !this.hasFiles(event)) return;
    event.preventDefault();
    event.dataTransfer!.dropEffect = 'copy';
    this.zoneActive.set(true);
  }

  onZoneDragLeave(): void {
    this.zoneActive.set(false);
  }

  onZoneDrop(event: DragEvent): void {
    if (this.applying() || !this.hasFiles(event)) return;
    event.preventDefault();
    this.zoneActive.set(false);
    const files = event.dataTransfer ? Array.from(event.dataTransfer.files) : [];
    if (!files.length) return;

    const newCards: ImagenCard[] = files.map((f) => ({
      key: this.nextKey++,
      id: null,
      url: URL.createObjectURL(f),
      file: f,
      status: 'pending-upload',
    }));
    this.cards.update((list) => [...list, ...newCards]);
  }

  private hasFiles(event: DragEvent): boolean {
    const types = event.dataTransfer?.types ?? [];
    return Array.from(types).includes('Files');
  }

  get pendingUploads(): number {
    return this.cards().filter((c) => c.status === 'pending-upload').length;
  }

  get pendingDeletes(): number {
    return this.cards().filter((c) => c.status === 'pending-delete').length;
  }

  get hasPending(): boolean {
    return this.pendingUploads + this.pendingDeletes > 0 || this.orderChanged;
  }

  get orderChanged(): boolean {
    const current = this.cards()
      .filter((c) => c.status === 'uploaded')
      .map((c) => c.id!);
    if (current.length !== this.serverOrderIds.length) return false;
    return current.some((id, i) => id !== this.serverOrderIds[i]);
  }

  get pendingSummary(): string {
    const parts: string[] = [];
    if (this.pendingUploads) parts.push(`${this.pendingUploads} subir`);
    if (this.pendingDeletes) parts.push(`${this.pendingDeletes} eliminar`);
    if (this.orderChanged) parts.push('reordenar');
    return parts.length ? `Aplicar cambios (${parts.join(' · ')})` : 'Aplicar cambios';
  }

  async applyImages(): Promise<void> {
    const id = this.entityId();
    this.applying.set(true);
    try {
      const current = this.cards();
      const toUpload = current.filter((c) => c.status === 'pending-upload');
      const toDelete = current.filter((c) => c.status === 'pending-delete');

      // 1. Subir las pendientes (incluye reemplazos editados) antes de borrar.
      const known = new Set(current.filter((c) => c.id != null).map((c) => c.id!));
      let uploadedIds: number[] = [];
      if (toUpload.length) {
        const files = toUpload.map((c) => c.file!);
        const all = await firstValueFrom(this.api().uploadImagenes(id, files));
        uploadedIds = all.filter((img) => !known.has(img.id)).map((img) => img.id);
      }

      // 2. Eliminar las marcadas en el servidor solo si la subida salió bien.
      for (const c of toDelete) {
        await firstValueFrom(this.api().deleteImagen(id, c.id!));
      }

      // 3. Reordenar: ids de servidor en el orden visual actual
      const orderIds: number[] = [];
      for (const c of this.cards()) {
        if (c.status === 'uploaded') {
          orderIds.push(c.id!);
        } else if (c.status === 'pending-upload') {
          const newId = uploadedIds.shift();
          if (newId != null) orderIds.push(newId);
        }
      }
      if (orderIds.length) {
        await firstValueFrom(this.api().reorderImagenes(id, orderIds));
      }

      // 4. Refrescar desde el servidor
      const fresh = await firstValueFrom(this.api().getByID(id));
      this.revokeLocalUrls(this.cards());
      this.cards.set(this.toCards(fresh.imagenes ?? []));
      this.editedKeys.set(new Set());
      this.serverOrderIds = (fresh.imagenes ?? []).map((img) => img.id);

      this.toast.success('Cambios de imágenes aplicados');
      this.imagesApplied.emit();
    } catch {
      this.toast.error('Error al aplicar cambios de imágenes');
    } finally {
      this.applying.set(false);
    }
  }

  async discardChanges(): Promise<void> {
    if (this.applying()) return;
    try {
      const fresh = await firstValueFrom(this.api().getByID(this.entityId()));
      this.revokeLocalUrls(this.cards());
      this.cards.set(this.toCards(fresh.imagenes ?? []));
      this.editedKeys.set(new Set());
      this.serverOrderIds = (fresh.imagenes ?? []).map((img) => img.id);
      this.toast.success('Cambios descartados');
    } catch {
      this.toast.error('Error al descartar cambios');
    }
  }
}
