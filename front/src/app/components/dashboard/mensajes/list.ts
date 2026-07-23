import { Component, inject, signal, OnInit, computed } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { MensajesService } from '../../../services/mensajes.service';
import { MensajesStoreService } from '../../../services/mensajes-store.service';
import { ToastService } from '../../../services/toast.service';
import { ConfirmService } from '../../../services/confirm.service';
import { AuthService } from '../../../services/auth.service';
import { PaginationComponent } from '../../../common/pagination/pagination';
import type { MensajePendiente, MensajeInput } from '../../../models/mensaje';
import { DatePipe } from '@angular/common';

@Component({
  selector: 'app-mensajes-list',
  templateUrl: './list.html',
  styleUrl: './list.css',
  imports: [FormsModule, PaginationComponent, DatePipe],
})
export class MensajesListComponent implements OnInit {
  private service = inject(MensajesService);
  protected store = inject(MensajesStoreService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmService);
  protected auth = inject(AuthService);

  activeTab = signal<'activas' | 'pasadas'>('activas');

  // Activas — usa el store global (SSE persistente)
  activasPage = signal(1);
  activasPageSize = 10;

  filterEstado = signal('');
  filterTelefono = signal('');

  filteredActivas = computed(() => {
    let items = this.store.activos();
    const fe = this.filterEstado();
    const ft = this.filterTelefono().trim().toLowerCase();
    if (fe) items = items.filter(m => m.estado === fe);
    if (ft) items = items.filter(m => m.telefono.toLowerCase().includes(ft));
    return items;
  });

  totalActivasPages = computed(() => Math.max(1, Math.ceil(this.filteredActivas().length / this.activasPageSize)));

  visibleActivas = computed(() => {
    const start = (this.activasPage() - 1) * this.activasPageSize;
    return this.filteredActivas().slice(start, start + this.activasPageSize);
  });

  // Pasadas (API pagination)
  pasadas = signal<MensajePendiente[]>([]);
  pasadasPage = signal(1);
  pasadasTotal = signal(0);
  pasadasPageSize = 10;

  totalPasadasPages = computed(() => Math.ceil(this.pasadasTotal() / this.pasadasPageSize));

  // Create modal
  showCreate = signal(false);
  createTelefono = '';
  createHoraDesactivacion = '';

  ngOnInit(): void {
    this.activasPage.set(1);
  }

  onTabChange(tab: 'activas' | 'pasadas'): void {
    this.activeTab.set(tab);
    if (tab === 'pasadas') this.loadPasadas();
  }

  onActivasPageChange(p: number): void {
    this.activasPage.set(p);
  }

  onPasadasPageChange(p: number): void {
    this.pasadasPage.set(p);
    this.loadPasadas();
  }

  loadPasadas(): void {
    const offset = (this.pasadasPage() - 1) * this.pasadasPageSize;
    this.service.listPasadas(offset, this.pasadasPageSize).subscribe({
      next: (res) => {
        this.pasadas.set(res.data);
        this.pasadasTotal.set(res.total);
      },
      error: () => this.toast.error('Error al cargar pasadas'),
    });
  }

  openCreate(): void {
    this.showCreate.set(true);
    this.createTelefono = '';
    this.createHoraDesactivacion = '';
  }

  creating = signal(false);

  createMensaje(): void {
    if (!this.createTelefono || !this.createHoraDesactivacion) {
      this.toast.error('Teléfono y hora de desactivación son requeridos');
      return;
    }
    this.creating.set(true);
    this.service.create({ telefono: this.createTelefono, hora_desactivacion: this.createHoraDesactivacion }).subscribe({
      next: () => {
        this.toast.success('Mensaje creado');
        this.creating.set(false);
        this.showCreate.set(false);
      },
      error: () => {
        this.toast.error('Error al crear mensaje');
        this.creating.set(false);
      },
    });
  }

  saving = signal<string | null>(null);

  async asignarse(m: MensajePendiente): Promise<void> {
    const ok = await this.confirm.confirm('Asignarse', `¿Asignarte el mensaje de ${m.telefono}?`, 'warning');
    if (!ok) return;
    this.saving.set(`asignar-${m.id}`);
    this.service.update(m.id, { usuario_acargo: this.auth.user()?.username }).subscribe({
      next: () => {
        this.toast.success('Mensaje asignado');
        this.saving.set(null);
      },
      error: () => {
        this.toast.error('Error al asignar');
        this.saving.set(null);
      },
    });
  }

  async finalizar(m: MensajePendiente): Promise<void> {
    const ok = await this.confirm.confirm('Finalizar', `¿Finalizar el mensaje de ${m.telefono}?`, 'warning');
    if (!ok) return;
    this.saving.set(`finalizar-${m.id}`);
    this.service.update(m.id, { finalizar: true }).subscribe({
      next: () => {
        this.toast.success('Mensaje finalizado');
        this.saving.set(null);
      },
      error: () => {
        this.toast.error('Error al finalizar');
        this.saving.set(null);
      },
    });
  }

  async deleteMensaje(m: MensajePendiente): Promise<void> {
    const ok = await this.confirm.confirm('Eliminar', `¿Eliminar el mensaje de ${m.telefono}?`, 'danger');
    if (!ok) return;
    this.saving.set(`delete-${m.id}`);
    this.service.delete(m.id).subscribe({
      next: () => {
        this.toast.success('Mensaje eliminado');
        this.saving.set(null);
      },
      error: () => {
        this.toast.error('Error al eliminar');
        this.saving.set(null);
      },
    });
  }
}
