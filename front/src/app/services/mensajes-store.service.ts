import { Injectable, OnDestroy, signal } from '@angular/core';
import { Subscription } from 'rxjs';
import { SSEService, type SSEEvent } from './sse.service';
import type { MensajePendiente } from '../models/mensaje';

@Injectable({ providedIn: 'root' })
export class MensajesStoreService implements OnDestroy {
  readonly activos = signal<MensajePendiente[]>([]);
  readonly loading = signal(false);

  private sseSub: Subscription | null = null;

  constructor(private sse: SSEService) {}

  init(): void {
    if (this.sseSub) return;
    this.loading.set(true);

    this.sseSub = this.sse.connect<MensajePendiente[] | MensajePendiente | { id: number }>('/mensajes_pendientes/events').subscribe({
      next: (event: SSEEvent) => {
        switch (event.type) {
          case 'current': {
            const list = event.data as MensajePendiente[];
            this.activos.set(list);
            this.loading.set(false);
            break;
          }
          case 'create': {
            const m = event.data as MensajePendiente;
            this.activos.update(list => [m, ...list]);
            break;
          }
          case 'update': {
            const m = event.data as MensajePendiente;
            this.activos.update(list => list.map(x => x.id === m.id ? m : x));
            break;
          }
          case 'delete': {
            const { id } = event.data as { id: number };
            this.activos.update(list => list.filter(x => x.id !== id));
            break;
          }
        }
      },
      error: () => {
        this.sseSub = null;
        this.loading.set(false);
        setTimeout(() => this.init(), 3000);
      },
    });
  }

  disconnect(): void {
    this.sseSub?.unsubscribe();
    this.sseSub = null;
    this.sse.disconnect();
    this.activos.set([]);
  }

  ngOnDestroy(): void {
    this.disconnect();
  }
}
