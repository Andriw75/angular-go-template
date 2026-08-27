import { Injectable, computed, inject, signal } from '@angular/core';
import { Observable, tap } from 'rxjs';
import {
  InstalacionesService,
  type InstalacionCreateInput,
  type InstalacionUpdateInput,
} from './instalaciones.service';
import type {
  Instalacion,
  InstalacionEstado,
  InstalacionTipo,
  InstalacionZona,
} from '../models/instalacion';

@Injectable({ providedIn: 'root' })
export class InstalacionesStoreService {
  private service = inject(InstalacionesService);

  readonly items = signal<Instalacion[]>([]);
  readonly loading = signal(false);
  readonly selectedId = signal<number | null>(null);

  readonly q = signal('');
  readonly tipo = signal<InstalacionTipo | ''>('');
  readonly estado = signal<InstalacionEstado | ''>('');
  readonly zona = signal<InstalacionZona | ''>('');

  readonly total = computed(() => this.items().length);

  readonly filtered = computed(() => {
    const q = this.q().trim().toLowerCase();
    return this.items().filter((i) => {
      if (this.tipo() && i.tipo !== this.tipo()) return false;
      if (this.estado() && i.estado !== this.estado()) return false;
      if (this.zona() && i.zona !== this.zona()) return false;
      if (q) {
        const haystack = [i.nombre, i.direccion, i.serial, i.contenido ?? '', i.descripcion ?? '']
          .join(' ')
          .toLowerCase();
        if (!haystack.includes(q)) return false;
      }
      return true;
    });
  });

  readonly countsByEstado = computed(() => {
    const map = new Map<InstalacionEstado, number>();
    for (const i of this.items()) {
      map.set(i.estado, (map.get(i.estado) ?? 0) + 1);
    }
    return map;
  });

  load(): void {
    this.loading.set(true);
    this.service.list().subscribe({
      next: (items) => {
        this.items.set(items);
        this.loading.set(false);
      },
      error: () => this.loading.set(false),
    });
  }

  create(input: InstalacionCreateInput): Observable<Instalacion> {
    this.loading.set(true);
    return this.service.create(input).pipe(
      tap({
        next: (created) => {
          this.loading.set(false);
          this.items.update((list) => [created, ...list]);
          this.selectedId.set(created.id);
        },
        error: () => this.loading.set(false),
      }),
    );
  }

  update(id: number, patch: InstalacionUpdateInput): Observable<Instalacion> {
    return this.service.update(id, patch).pipe(
      tap({
        next: (updated) => {
          this.items.update((list) => list.map((x) => (x.id === id ? updated : x)));
          this.selectedId.set(id);
        },
      }),
    );
  }

  remove(id: number): Observable<void> {
    return this.service.delete(id).pipe(
      tap({
        next: () => {
          this.items.update((list) => list.filter((x) => x.id !== id));
          if (this.selectedId() === id) this.selectedId.set(null);
        },
      }),
    );
  }

  select(id: number | null): void {
    this.selectedId.set(id);
  }

  toggleEstado(estado: InstalacionEstado): void {
    this.estado.set(this.estado() === estado ? '' : estado);
  }

  resetFilters(): void {
    this.q.set('');
    this.tipo.set('');
    this.estado.set('');
    this.zona.set('');
  }
}