import { Injectable, OnDestroy, signal } from '@angular/core';
import { Subscription } from 'rxjs';
import { AuthService } from './auth.service';
import { SSEService, type SSEEvent } from './sse.service';
import type { MensajePendiente } from '../models/mensaje';

@Injectable({ providedIn: 'root' })
export class MensajesStoreService implements OnDestroy {
  readonly activos = signal<MensajePendiente[]>([]);
  readonly loading = signal(false);

  private sseSub: Subscription | null = null;

  constructor(
    private sse: SSEService,
    private auth: AuthService,
  ) {}

  init(): void {
    if (!this.auth.hasPermission('mensajes_pendientes')) {
      this.disconnect();
      return;
    }
    if (this.sseSub) return;

    this.loading.set(true);
    this.sseSub = this.sse.events$.subscribe((event: SSEEvent) => {
      if (event.module !== 'mensajes_pendientes' || event.name !== 'activos') return;

      switch (event.type) {
        case 'current': {
          this.activos.set(event.data as MensajePendiente[]);
          this.loading.set(false);
          break;
        }
        case 'create': {
          const m = event.data as MensajePendiente;
          this.activos.update((list) => [m, ...list.filter((x) => x.id !== m.id)]);
          break;
        }
        case 'update': {
          const m = event.data as MensajePendiente;
          this.activos.update((list) => {
            const index = list.findIndex((x) => x.id === m.id);
            if (index === -1) return [m, ...list];
            const copy = [...list];
            copy[index] = m;
            return copy;
          });
          break;
        }
        case 'delete': {
          const { id } = event.data as { id: number };
          this.activos.update((list) => list.filter((x) => x.id !== id));
          break;
        }
      }
    });
  }

  disconnect(): void {
    this.sseSub?.unsubscribe();
    this.sseSub = null;
    this.activos.set([]);
    this.loading.set(false);
  }

  ngOnDestroy(): void {
    this.disconnect();
  }
}
