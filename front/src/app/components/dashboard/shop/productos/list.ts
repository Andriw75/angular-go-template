import { Component, inject, signal } from '@angular/core';
import { DecimalPipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { forkJoin } from 'rxjs';
import { ProductosService } from '../../../../services/productos.service';
import { CategoriasService } from '../../../../services/categorias.service';
import { ToastService } from '../../../../services/toast.service';
import { ConfirmService } from '../../../../services/confirm.service';
import { PaginationComponent } from '../../../../common/pagination/pagination';
import { ProductoModalComponent } from './modal';
import type { Producto } from '../../../../models/producto';
import type { Categoria } from '../../../../models/categoria';

@Component({
  selector: 'app-productos-list',
  templateUrl: './list.html',
  styleUrl: './list.css',
  imports: [PaginationComponent, ProductoModalComponent, DecimalPipe, FormsModule],
})
export class ProductosListComponent {
  private service = inject(ProductosService);
  private categoriasService = inject(CategoriasService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmService);

  productos = signal<Producto[]>([]);
  categorias = signal<Categoria[]>([]);
  loading = signal(false);
  page = signal(1);
  limit = 10;
  total = signal(0);

  searchQuery = signal('');
  filterCategoria = signal('');
  filterActivo = signal('');

  showModal = signal(false);
  selectedProducto = signal<Producto | null>(null);

  constructor() {
    this.loadCategorias();
    this.loadWithCount();
  }

  private loadCategorias(): void {
    this.categoriasService.list({ limit: 100 }).subscribe({
      next: (res) => this.categorias.set(res.data),
    });
  }

  categoriaNombre(id: number): string {
    return this.categorias().find((c) => c.id === id)?.nombre ?? `#${id}`;
  }

  private getFilters() {
    return {
      q: this.searchQuery() || undefined,
      categoria_id: this.filterCategoria() ? Number(this.filterCategoria()) : undefined,
      activo: this.filterActivo() ? this.filterActivo() === 'true' : undefined,
    };
  }

  loadWithCount(): void {
    this.loading.set(true);
    const filters = this.getFilters();
    const offset = (this.page() - 1) * this.limit;

    forkJoin({
      count: this.service.count(filters),
      data: this.service.list({ ...filters, offset, limit: this.limit }),
    }).subscribe({
      next: (res) => {
        this.total.set(res.count);
        this.productos.set(res.data.data);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.toast.error('Error al cargar productos');
      },
    });
  }

  load(): void {
    this.loading.set(true);
    const filters = this.getFilters();
    const offset = (this.page() - 1) * this.limit;

    this.service.list({ ...filters, offset, limit: this.limit }).subscribe({
      next: (res) => {
        this.productos.set(res.data);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.toast.error('Error al cargar productos');
      },
    });
  }

  onPageChange(p: number): void {
    this.page.set(p);
    this.load();
  }

  search(): void {
    this.page.set(1);
    this.loadWithCount();
  }

  resetFilters(): void {
    this.searchQuery.set('');
    this.filterCategoria.set('');
    this.filterActivo.set('');
    this.page.set(1);
    this.loadWithCount();
  }

  openCreate(): void {
    this.selectedProducto.set(null);
    this.showModal.set(true);
  }

  openEdit(producto: Producto): void {
    this.selectedProducto.set(producto);
    this.showModal.set(true);
  }

  onModalSaved(): void {
    this.loadWithCount();
  }

  deleting = signal<number | null>(null);

  async confirmDelete(producto: Producto): Promise<void> {
    const ok = await this.confirm.confirm(
      'Eliminar Producto',
      `¿Seguro de eliminar el producto "${producto.nombre}"?`,
      'danger',
    );
    if (!ok) return;
    this.deleting.set(producto.id);

    this.service.delete(producto.id).subscribe({
      next: () => {
        this.toast.success('Producto eliminado');
        this.deleting.set(null);
        this.loadWithCount();
      },
      error: () => {
        this.toast.error('Error al eliminar producto');
        this.deleting.set(null);
      },
    });
  }

  get totalPages(): number {
    return Math.ceil(this.total() / this.limit);
  }
}
