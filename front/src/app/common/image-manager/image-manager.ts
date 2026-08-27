import { Component, effect, inject, input, OnDestroy, output, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import type { Observable } from 'rxjs';
import { environment } from '../../../environments/environment';
import type { Imagen } from '../../models/imagen';
import { ToastService } from '../../services/toast.service';

export type CardStatus = 'uploaded' | 'pending-upload' | 'pending-delete';

export interface ImagenCard {
  key: number;
  id: number | null;
  url: string;
  file?: File;
  status: CardStatus;
}

export interface ImagenesApi {
  uploadImagenes(id: number, files: File[]): Observable<Imagen[]>;
  deleteImagen(id: number, imagenId: number): Observable<void>;
  reorderImagenes(id: number, ids: number[]): Observable<Imagen[]>;
  getByID(id: number): Observable<{ imagenes: Imagen[] }>;
}

const STATUS_LABEL: Record<CardStatus, string> = {
  'uploaded': 'En servidor',
  'pending-upload': 'Por subir',
  'pending-delete': 'Por eliminar',
};

@Component({
  selector: 'app-image-manager',
  templateUrl: './image-manager.html',
  styleUrl: './image-manager.css',
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
        url: environment.MEDIA_URL + img.url,
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
    this.cards.update((list) => {
      if (card.status === 'pending-upload') {
        if (card.url.startsWith('blob:')) URL.revokeObjectURL(card.url);
        return list.filter((c) => c.key !== card.key);
      }
      if (card.status === 'uploaded') {
        return list.map((c) => (c.key === card.key ? { ...c, status: 'pending-delete' as CardStatus } : c));
      }
      return list.map((c) => (c.key === card.key ? { ...c, status: 'uploaded' as CardStatus } : c));
    });
  }

  statusLabel(card: ImagenCard): string {
    return STATUS_LABEL[card.status];
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
    const current = this.cards().filter((c) => c.status === 'uploaded').map((c) => c.id!);
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

      // 1. Eliminar las marcadas en el servidor
      for (const c of toDelete) {
        await firstValueFrom(this.api().deleteImagen(id, c.id!));
      }

      // 2. Subir las pendientes (en orden), en un solo request
      let uploadedIds: number[] = [];
      if (toUpload.length) {
        const known = new Set(current.filter((c) => c.id != null).map((c) => c.id!));
        const files = toUpload.map((c) => c.file!);
        const all = await firstValueFrom(this.api().uploadImagenes(id, files));
        uploadedIds = all.filter((img) => !known.has(img.id)).map((img) => img.id);
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
      this.serverOrderIds = (fresh.imagenes ?? []).map((img) => img.id);
      this.toast.success('Cambios descartados');
    } catch {
      this.toast.error('Error al descartar cambios');
    }
  }
}
