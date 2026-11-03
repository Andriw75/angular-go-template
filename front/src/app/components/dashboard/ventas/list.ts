import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { apiErrorMessage } from '../../../common/api-error';
import { PaginationComponent } from '../../../common/pagination/pagination';
import { ConfirmService } from '../../../services/confirm.service';
import { ToastService } from '../../../services/toast.service';
import { VentasService } from '../../../services/ventas.service';
import { VentaModalComponent } from './modal';
import type { Venta } from '../../../models/venta';

@Component({
  selector: 'app-ventas-list',
  templateUrl: './list.html',
  styleUrl: './list.css',
  imports: [VentaModalComponent, PaginationComponent, FormsModule],
})
export class VentasListComponent {
  private service = inject(VentasService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmService);

  ventas = signal<Venta[]>([]);
  loading = signal(false);
  page = signal(1);
  limit = 10;
  total = signal(0);

  q = '';
  serie = '';
  estado = '';

  showModal = signal(false);
  selected = signal<Venta | null>(null);

  constructor() {
    this.load();
  }

  load(): void {
    this.loading.set(true);
    const offset = (this.page() - 1) * this.limit;
    this.service.list({
      offset, limit: this.limit, q: this.q || undefined,
      serie: this.serie || undefined, estado: this.estado || undefined,
    }).subscribe({
      next: (res) => {
        this.ventas.set(res.data);
        this.total.set(res.total);
        this.loading.set(false);
      },
      error: (err) => {
        this.loading.set(false);
        this.toast.error(apiErrorMessage(err, 'Error al cargar ventas'));
      },
    });
  }

  search(): void {
    this.page.set(1);
    this.load();
  }

  resetFilters(): void {
    this.q = '';
    this.serie = '';
    this.estado = '';
    this.page.set(1);
    this.load();
  }

  onPageChange(p: number): void {
    this.page.set(p);
    this.load();
  }

  openCreate(): void {
    this.selected.set(null);
    this.showModal.set(true);
  }

  openEdit(venta: Venta): void {
    if (venta.estado !== 'borrador') return;
    this.selected.set(venta);
    this.showModal.set(true);
  }

  onSaved(): void {
    this.showModal.set(false);
    this.load();
  }

  emitir(venta: Venta): void {
    this.service.emit(venta.id).subscribe({
      next: (v) => {
        this.toast.success(`Venta emitida: ${v.codigo}`);
        this.load();
      },
      error: (err) => this.toast.error(apiErrorMessage(err, 'Error al emitir venta')),
    });
  }

  async anular(venta: Venta): Promise<void> {
    const ok = await this.confirm.confirm('Anular venta', `¿Anular ${venta.codigo}?`, 'danger');
    if (!ok) return;
    this.service.annul(venta.id, 'Anulada desde el listado').subscribe({
      next: () => { this.toast.success('Venta anulada'); this.load(); },
      error: (err) => this.toast.error(apiErrorMessage(err, 'Error al anular venta')),
    });
  }

  async eliminar(venta: Venta): Promise<void> {
    const ok = await this.confirm.confirm('Eliminar venta', `¿Eliminar el borrador ${venta.id}?`, 'danger');
    if (!ok) return;
    this.service.delete(venta.id).subscribe({
      next: () => { this.toast.success('Venta eliminada'); this.load(); },
      error: (err) => this.toast.error(apiErrorMessage(err, 'Error al eliminar venta')),
    });
  }

  get totalPages(): number {
    return Math.max(1, Math.ceil(this.total() / this.limit));
  }
}
