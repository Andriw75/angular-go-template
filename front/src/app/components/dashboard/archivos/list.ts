import { Component, computed, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { forkJoin } from 'rxjs';
import { apiErrorMessage } from '../../../common/api-error';
import { PaginationComponent } from '../../../common/pagination/pagination';
import { AuthService } from '../../../services/auth.service';
import { CategoriasService } from '../../../services/categorias.service';
import { ProductosService } from '../../../services/productos.service';
import { ArchivosService } from '../../../services/archivos.service';
import { ToastService } from '../../../services/toast.service';
import { ConfirmService } from '../../../services/confirm.service';
import { ArchivoModalComponent } from './modal';
import type { Archivo, Visibilidad } from '../../../models/archivo';
import type { Imagen } from '../../../models/imagen';

interface EntityImage {
  key: string;
  tipo: 'categoria' | 'producto';
  entidadId: number;
  entidadNombre: string;
  imagen: Imagen;
}

@Component({
  selector: 'app-archivos-list',
  templateUrl: './list.html',
  styleUrl: './list.css',
  imports: [PaginationComponent, ArchivoModalComponent, FormsModule, DatePipe],
})
export class ArchivosListComponent {
  service = inject(ArchivosService);
  private categoriasService = inject(CategoriasService);
  private productosService = inject(ProductosService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmService);
  protected auth = inject(AuthService);

  activeTab = signal<'archivos' | 'imagenes'>('archivos');
  readonly canSeeImages = computed(() => this.auth.hasPermission('shop'));

  archivos = signal<Archivo[]>([]);
  loading = signal(false);
  page = signal(1);
  limit = 20;
  total = signal(0);

  searchQuery = signal('');
  filterVisibilidad = signal('');

  showModal = signal(false);

  entityImages = signal<EntityImage[]>([]);
  imagesLoading = signal(false);
  imageType = signal('');
  imagePublic = signal('');

  readonly filteredEntityImages = computed(() => {
    const tipo = this.imageType();
    const publico = this.imagePublic();

    return this.entityImages().filter((item) => {
      if (tipo && item.tipo !== tipo) return false;
      if (publico && String(item.imagen.publico) !== publico) return false;
      return true;
    });
  });

  constructor() {
    this.load();
  }

  setTab(tab: 'archivos' | 'imagenes'): void {
    if (tab === 'imagenes' && !this.canSeeImages()) return;
    this.activeTab.set(tab);
    if (tab === 'imagenes') {
      this.loadImages();
    }
  }

  load(): void {
    this.loading.set(true);
    const offset = (this.page() - 1) * this.limit;

    this.service.list({ offset, limit: this.limit, q: this.searchQuery() || undefined }).subscribe({
      next: (res) => {
        this.archivos.set(res.data);
        this.total.set(res.total);
        this.loading.set(false);
      },
      error: (err) => {
        this.loading.set(false);
        this.toast.error(apiErrorMessage(err, 'Error al cargar archivos'));
      },
    });
  }

  loadImages(): void {
    if (!this.canSeeImages()) return;

    this.imagesLoading.set(true);
    forkJoin({
      categorias: this.categoriasService.list({ limit: 100 }),
      productos: this.productosService.list({ limit: 100 }),
    }).subscribe({
      next: ({ categorias, productos }) => {
        const images: EntityImage[] = [];

        for (const categoria of categorias.data) {
          for (const imagen of categoria.imagenes) {
            images.push({
              key: `categoria-${categoria.id}-${imagen.id}`,
              tipo: 'categoria',
              entidadId: categoria.id,
              entidadNombre: categoria.nombre,
              imagen,
            });
          }
        }

        for (const producto of productos.data) {
          for (const imagen of producto.imagenes) {
            images.push({
              key: `producto-${producto.id}-${imagen.id}`,
              tipo: 'producto',
              entidadId: producto.id,
              entidadNombre: producto.nombre,
              imagen,
            });
          }
        }

        images.sort(
          (a, b) =>
            a.tipo.localeCompare(b.tipo) ||
            a.entidadNombre.localeCompare(b.entidadNombre) ||
            a.imagen.orden - b.imagen.orden,
        );

        this.entityImages.set(images);
        this.imagesLoading.set(false);
      },
      error: (err) => {
        this.imagesLoading.set(false);
        this.toast.error(apiErrorMessage(err, 'Error al cargar imágenes'));
      },
    });
  }

  imageURL(item: EntityImage): string {
    return this.service.getImagenURL(item.imagen);
  }

  async copyImageLink(item: EntityImage): Promise<void> {
    try {
      await navigator.clipboard.writeText(this.imageURL(item));
      this.toast.success('Link copiado');
    } catch {
      this.toast.error('No se pudo copiar el link');
    }
  }

  openImage(item: EntityImage): void {
    window.open(this.imageURL(item), '_blank');
  }

  async deleteEntityImage(item: EntityImage): Promise<void> {
    const ok = await this.confirm.confirm(
      'Eliminar Imagen',
      `Seguro de eliminar la imagen de "${item.entidadNombre}"?`,
      'danger',
    );
    if (!ok) return;

    const request =
      item.tipo === 'categoria'
        ? this.categoriasService.deleteImagen(item.entidadId, item.imagen.id)
        : this.productosService.deleteImagen(item.entidadId, item.imagen.id);

    request.subscribe({
      next: () => {
        this.entityImages.update((list) => list.filter((x) => x.key !== item.key));
        this.toast.success('Imagen eliminada');
      },
      error: (err) => {
        this.toast.error(apiErrorMessage(err, 'Error al eliminar imagen'));
      },
    });
  }

  onPageChange(p: number): void {
    this.page.set(p);
    this.load();
  }

  search(): void {
    this.page.set(1);
    this.load();
  }

  resetFilters(): void {
    this.searchQuery.set('');
    this.filterVisibilidad.set('');
    this.page.set(1);
    this.load();
  }

  openUpload(): void {
    this.showModal.set(true);
  }

  onModalSaved(): void {
    this.load();
  }

  deleting = signal<number | null>(null);

  async confirmDelete(archivo: Archivo): Promise<void> {
    const ok = await this.confirm.confirm(
      'Eliminar Archivo',
      `Seguro de eliminar "${archivo.nombre}"?`,
      'danger',
    );
    if (!ok) return;

    this.deleting.set(archivo.id);
    this.service.delete(archivo.id).subscribe({
      next: () => {
        this.toast.success('Archivo eliminado');
        this.deleting.set(null);
        this.load();
      },
      error: (err) => {
        this.toast.error(apiErrorMessage(err, 'Error al eliminar archivo'));
        this.deleting.set(null);
      },
    });
  }

  onVisibilidadChange(archivo: Archivo, nuevaVis: Visibilidad): void {
    this.service.updateVisibilidad(archivo.id, nuevaVis).subscribe({
      next: (updated) => {
        this.archivos.update(list => list.map(a => a.id === updated.id ? updated : a));
        this.toast.success('Visibilidad actualizada');
      },
      error: (err) => {
        this.toast.error(apiErrorMessage(err, 'Error al cambiar visibilidad'));
      },
    });
  }

  async copyLink(archivo: Archivo): Promise<void> {
    const url = this.service.getURL(archivo);
    try {
      await navigator.clipboard.writeText(url);
      this.toast.success('Link copiado');
    } catch {
      this.toast.error('No se pudo copiar el link');
    }
  }

  downloadArchivo(archivo: Archivo): void {
    const url = this.service.getURL(archivo);
    const a = document.createElement('a');
    a.href = url;
    a.download = archivo.nombre;
    a.target = '_blank';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  }

  openArchivo(archivo: Archivo): void {
    window.open(this.service.getURL(archivo), '_blank');
  }

  formatSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  isImage(archivo: Archivo): boolean {
    return archivo.tipo.startsWith('image/');
  }

  get filteredArchivos(): Archivo[] {
    if (!this.filterVisibilidad()) return this.archivos();
    return this.archivos().filter(a => a.visibilidad === this.filterVisibilidad());
  }

  get totalPages(): number {
    return Math.ceil(this.total() / this.limit);
  }
}
