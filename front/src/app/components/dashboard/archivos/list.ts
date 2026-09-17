import { Component, inject, signal } from '@angular/core';
import { DatePipe } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ArchivosService } from '../../../services/archivos.service';
import { ToastService } from '../../../services/toast.service';
import { ConfirmService } from '../../../services/confirm.service';
import { PaginationComponent } from '../../../common/pagination/pagination';
import { ArchivoModalComponent } from './modal';
import type { Archivo, Visibilidad } from '../../../models/archivo';

@Component({
  selector: 'app-archivos-list',
  templateUrl: './list.html',
  styleUrl: './list.css',
  imports: [PaginationComponent, ArchivoModalComponent, FormsModule, DatePipe],
})
export class ArchivosListComponent {
  service = inject(ArchivosService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmService);

  archivos = signal<Archivo[]>([]);
  loading = signal(false);
  page = signal(1);
  limit = 20;
  total = signal(0);

  searchQuery = signal('');
  filterVisibilidad = signal('');

  showModal = signal(false);

  constructor() {
    this.load();
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
      error: () => {
        this.loading.set(false);
        this.toast.error('Error al cargar archivos');
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
      error: () => {
        this.toast.error('Error al eliminar archivo');
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
      error: () => {
        this.toast.error('Error al cambiar visibilidad');
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
