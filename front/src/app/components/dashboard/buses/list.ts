import { DecimalPipe } from '@angular/common';
import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { forkJoin } from 'rxjs';
import { BusesService } from '../../../services/buses.service';
import { ToastService } from '../../../services/toast.service';
import { ConfirmService } from '../../../services/confirm.service';
import { PaginationComponent } from '../../../common/pagination/pagination';
import { BusModalComponent } from './modal';
import type { Bus } from '../../../models/bus';

@Component({
  selector: 'app-buses-list',
  templateUrl: './list.html',
  styleUrl: './list.css',
  imports: [PaginationComponent, BusModalComponent, DecimalPipe, FormsModule],
})
export class BusesListComponent {
  private service = inject(BusesService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmService);

  buses = signal<Bus[]>([]);
  loading = signal(false);
  page = signal(1);
  limit = 10;
  total = signal(0);
  offset = signal(0);

  searchQuery = signal('');
  filterTipo = signal('');
  filterActivo = signal('');

  showModal = signal(false);
  selectedBus = signal<Bus | null>(null);

  tipos = ['', 'BUS', 'VAN', 'MINIBUS', 'MICROBUS'];

  constructor() {
    this.loadWithCount();
  }

  private getFilters() {
    return {
      q: this.searchQuery() || undefined,
      tipo: this.filterTipo() || undefined,
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
        this.buses.set(res.data.data);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.toast.error('Error al cargar buses');
      },
    });
  }

  load(): void {
    this.loading.set(true);
    const filters = this.getFilters();
    const offset = (this.page() - 1) * this.limit;

    this.service.list({ ...filters, offset, limit: this.limit }).subscribe({
      next: (res) => {
        this.buses.set(res.data);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.toast.error('Error al cargar buses');
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
    this.filterTipo.set('');
    this.filterActivo.set('');
    this.page.set(1);
    this.loadWithCount();
  }

  openCreate(): void {
    this.selectedBus.set(null);
    this.showModal.set(true);
  }

  openEdit(bus: Bus): void {
    this.selectedBus.set(bus);
    this.showModal.set(true);
  }

  onModalSaved(): void {
    this.loadWithCount();
  }

  async confirmDelete(bus: Bus): Promise<void> {
    const ok = await this.confirm.confirm(
      'Eliminar Bus',
      `¿Seguro de eliminar el bus "${bus.nombre}" (${bus.placa})?`,
    );
    if (!ok) return;

    this.service.delete(bus.id).subscribe({
      next: () => {
        this.toast.success('Bus eliminado');
        this.loadWithCount();
      },
      error: () => this.toast.error('Error al eliminar bus'),
    });
  }

  get totalPages(): number {
    return Math.ceil(this.total() / this.limit);
  }
}
