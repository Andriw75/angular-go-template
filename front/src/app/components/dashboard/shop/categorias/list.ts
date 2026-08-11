import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { forkJoin } from 'rxjs';
import { CategoriasService } from '../../../../services/categorias.service';
import { ProductosService } from '../../../../services/productos.service';
import { ToastService } from '../../../../services/toast.service';
import { ConfirmService } from '../../../../services/confirm.service';
import { PaginationComponent } from '../../../../common/pagination/pagination';
import { CategoriaModalComponent } from './modal';
import type { Categoria } from '../../../../models/categoria';

@Component({
  selector: 'app-categorias-list',
  templateUrl: './list.html',
  styleUrl: './list.css',
  imports: [PaginationComponent, CategoriaModalComponent, FormsModule],
})
export class CategoriasListComponent {
  private service = inject(CategoriasService);
  private productosService = inject(ProductosService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmService);

  categorias = signal<Categoria[]>([]);
  loading = signal(false);
  page = signal(1);
  limit = 10;
  total = signal(0);

  searchQuery = signal('');
  filterActivo = signal('');

  showModal = signal(false);
  selectedCategoria = signal<Categoria | null>(null);

  constructor() {
    this.loadWithCount();
  }

  private getFilters() {
    return {
      q: this.searchQuery() || undefined,
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
        this.categorias.set(res.data.data);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.toast.error('Error al cargar categorías');
      },
    });
  }

  load(): void {
    this.loading.set(true);
    const filters = this.getFilters();
    const offset = (this.page() - 1) * this.limit;

    this.service.list({ ...filters, offset, limit: this.limit }).subscribe({
      next: (res) => {
        this.categorias.set(res.data);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.toast.error('Error al cargar categorías');
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
    this.filterActivo.set('');
    this.page.set(1);
    this.loadWithCount();
  }

  openCreate(): void {
    this.selectedCategoria.set(null);
    this.showModal.set(true);
  }

  openEdit(categoria: Categoria): void {
    this.selectedCategoria.set(categoria);
    this.showModal.set(true);
  }

  onModalSaved(): void {
    this.loadWithCount();
  }

  deleting = signal<number | null>(null);

  async confirmDelete(categoria: Categoria): Promise<void> {
    const count = await new Promise<number>((resolve) => {
      this.productosService.count({ categoria_id: categoria.id }).subscribe({
        next: (n) => resolve(n),
        error: () => resolve(0),
      });
    });

    if (count > 0) {
      const ok = await this.confirm.confirm(
        'Eliminar Categoría',
        `La categoría "${categoria.nombre}" tiene ${count} producto(s) asociado(s). Se eliminarán junto con la categoría. ¿Continuar?`,
        'danger',
      );
      if (!ok) return;

      this.deleting.set(categoria.id);
      this.service.delete(categoria.id, true).subscribe({
        next: () => {
          this.toast.success('Categoría y sus productos eliminados');
          this.deleting.set(null);
          this.loadWithCount();
        },
        error: () => {
          this.toast.error('Error al eliminar categoría');
          this.deleting.set(null);
        },
      });
      return;
    }

    const ok = await this.confirm.confirm(
      'Eliminar Categoría',
      `¿Seguro de eliminar la categoría "${categoria.nombre}"?`,
      'danger',
    );
    if (!ok) return;

    this.deleting.set(categoria.id);
    this.service.delete(categoria.id).subscribe({
      next: () => {
        this.toast.success('Categoría eliminada');
        this.deleting.set(null);
        this.loadWithCount();
      },
      error: () => {
        this.toast.error('Error al eliminar categoría');
        this.deleting.set(null);
      },
    });
  }

  get totalPages(): number {
    return Math.ceil(this.total() / this.limit);
  }
}
