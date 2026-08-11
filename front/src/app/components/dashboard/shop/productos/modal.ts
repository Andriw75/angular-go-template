import { Component, effect, inject, input, OnDestroy, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { firstValueFrom } from 'rxjs';
import { ProductosService } from '../../../../services/productos.service';
import { CategoriasService } from '../../../../services/categorias.service';
import { ToastService } from '../../../../services/toast.service';
import { environment } from '../../../../../environments/environment';
import type { Producto, ProductoInput } from '../../../../models/producto';
import type { Categoria } from '../../../../models/categoria';
import type { Imagen } from '../../../../models/imagen';

type CardStatus = 'uploaded' | 'pending-upload' | 'pending-delete';

interface ImagenCard {
  key: number;
  id: number | null;
  url: string;
  file?: File;
  status: CardStatus;
}

const STATUS_LABEL: Record<CardStatus, string> = {
  'uploaded': 'En servidor',
  'pending-upload': 'Por subir',
  'pending-delete': 'Por eliminar',
};

@Component({
  selector: 'app-producto-modal',
  templateUrl: './modal.html',
  styleUrl: './modal.css',
  imports: [FormsModule],
})
export class ProductoModalComponent implements OnDestroy {
  private service = inject(ProductosService);
  private categoriasService = inject(CategoriasService);
  private toast = inject(ToastService);

  producto = input<Producto | null>(null);
  onClose = output<void>();
  onSaved = output<void>();

  isEditing = signal(false);
  saving = signal(false);
  error = signal('');
  categorias = signal<Categoria[]>([]);

  cards = signal<ImagenCard[]>([]);
  applying = signal(false);
  draggingKey = signal<number | null>(null);

  form: ProductoInput = {
    nombre: '',
    descripcion: '',
    precio: 0,
    stock: 0,
    categoria_id: 0,
    activo: true,
  };

  private nextKey = 1;
  private prevCards: ImagenCard[] = [];
  private serverOrderIds: number[] = [];

  constructor() {
    this.loadCategorias();

    effect(() => {
      const p = this.producto();
      this.revokeLocalUrls(this.prevCards);
      let cards: ImagenCard[];
      if (p) {
        this.isEditing.set(true);
        this.form = {
          nombre: p.nombre,
          descripcion: p.descripcion,
          precio: p.precio,
          stock: p.stock,
          categoria_id: p.categoria_id,
          activo: p.activo,
        };
        cards = this.toCards(p.imagenes ?? []);
      } else {
        this.isEditing.set(false);
        this.form = {
          nombre: '',
          descripcion: '',
          precio: 0,
          stock: 0,
          categoria_id: 0,
          activo: true,
        };
        cards = [];
      }
      this.prevCards = cards;
      this.cards.set(cards);
      this.serverOrderIds = cards.filter((card) => card.id != null).map((card) => card.id!);
      this.error.set('');
    });
  }

  ngOnDestroy(): void {
    this.revokeLocalUrls(this.cards());
  }

  private loadCategorias(): void {
    this.categoriasService.list({ limit: 100 }).subscribe({
      next: (res) => this.categorias.set(res.data),
    });
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

  private get entityId(): number {
    return this.producto()!.id;
  }

  close(): void {
    this.onClose.emit();
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

  onDragStart(event: DragEvent, card: ImagenCard): void {
    this.draggingKey.set(card.key);
    event.dataTransfer!.effectAllowed = 'move';
    event.dataTransfer!.setData('text/plain', String(card.key));
  }

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    event.dataTransfer!.dropEffect = 'move';
  }

  onDrop(event: DragEvent, target: ImagenCard): void {
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

  onDragEnd(): void {
    this.draggingKey.set(null);
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
    const id = this.entityId;
    this.applying.set(true);
    try {
      const current = this.cards();
      const toUpload = current.filter((c) => c.status === 'pending-upload');
      const toDelete = current.filter((c) => c.status === 'pending-delete');

      // 1. Eliminar las marcadas en el servidor
      for (const c of toDelete) {
        await firstValueFrom(this.service.deleteImagen(id, c.id!));
      }

      // 2. Subir las pendientes (en orden), en un solo request
      let uploadedIds: number[] = [];
      if (toUpload.length) {
        const known = new Set(current.filter((c) => c.id != null).map((c) => c.id!));
        const files = toUpload.map((c) => c.file!);
        const all = await firstValueFrom(this.service.uploadImagenes(id, files));
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
        await firstValueFrom(this.service.reorderImagenes(id, orderIds));
      }

      // 4. Refrescar desde el servidor
      const fresh = await firstValueFrom(this.service.getByID(id));
      this.revokeLocalUrls(this.cards());
      this.cards.set(this.toCards(fresh.imagenes ?? []));
      this.serverOrderIds = (fresh.imagenes ?? []).map((img) => img.id);

      this.toast.success('Cambios de imágenes aplicados');
    } catch {
      this.toast.error('Error al aplicar cambios de imágenes');
    } finally {
      this.applying.set(false);
    }
  }

  async discardChanges(): Promise<void> {
    if (this.applying()) return;
    try {
      const fresh = await firstValueFrom(this.service.getByID(this.entityId));
      this.revokeLocalUrls(this.cards());
      this.cards.set(this.toCards(fresh.imagenes ?? []));
      this.serverOrderIds = (fresh.imagenes ?? []).map((img) => img.id);
      this.toast.success('Cambios descartados');
    } catch {
      this.toast.error('Error al descartar cambios');
    }
  }

  onSubmit(): void {
    if (!this.form.nombre) {
      this.error.set('Nombre es obligatorio');
      return;
    }
    if (!this.form.categoria_id) {
      this.error.set('Debe seleccionar una categoría');
      return;
    }
    if (this.form.precio < 0 || this.form.stock < 0) {
      this.error.set('Precio y stock no pueden ser negativos');
      return;
    }

    this.saving.set(true);
    this.error.set('');

    const obs = this.isEditing()
      ? this.service.update(this.producto()!.id, this.form)
      : this.service.create(this.form);

    obs.subscribe({
      next: () => {
        this.toast.success(this.isEditing() ? 'Producto actualizado' : 'Producto creado');
        this.saving.set(false);
        this.onSaved.emit();
        this.close();
      },
      error: () => {
        this.saving.set(false);
        this.error.set('Error al guardar el producto');
      },
    });
  }
}
