import { DecimalPipe } from '@angular/common';
import { Component, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
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
    this.load();
  }

  load(): void {
    this.loading.set(true);
    const offset = (this.page() - 1) * this.limit;

    this.service.list({
      offset,
      limit: this.limit,
      q: this.searchQuery() || undefined,
      tipo: this.filterTipo() || undefined,
      activo: this.filterActivo() ? this.filterActivo() === 'true' : undefined,
    }).subscribe({
      next: (res) => {
        this.buses.set(res.data);
        this.total.set(res.total);
        this.offset.set(res.offset);
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
    this.load();
  }

  resetFilters(): void {
    this.searchQuery.set('');
    this.filterTipo.set('');
    this.filterActivo.set('');
    this.page.set(1);
    this.load();
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
    this.load();
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
        this.load();
      },
      error: () => this.toast.error('Error al eliminar bus'),
    });
  }

  get totalPages(): number {
    return Math.ceil(this.total() / this.limit);
  }
}
