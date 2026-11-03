import { Injectable, OnDestroy, signal } from '@angular/core';
import { Subscription } from 'rxjs';
import { AuthService } from './auth.service';
import { hasAnySsePermission } from './sse-channels';
import { SSEService, type SSEEvent } from './sse.service';
import type { MensajePendiente } from '../models/mensaje';

@Injectable({ providedIn: 'root' })
export class MensajesStoreService implements OnDestroy {
  readonly activos = signal<MensajePendiente[]>([]);
  readonly loading = signal(false);

  private sseSub: Subscription | null = null;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;

  constructor(
    private sse: SSEService,
    private auth: AuthService,
  ) {}

  init(): void {
    if (!hasAnySsePermission(this.auth.user())) {
      this.disconnect();
      return;
    }
    if (this.sseSub) return;

    this.clearRetry();
    this.loading.set(true);

    this.sseSub = this.sse
      .connect<MensajePendiente[] | MensajePendiente | { id: number }>('/sse/events')
      .subscribe({
        next: (event: SSEEvent) => {
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
        },
        error: () => {
          this.sseSub = null;
          this.loading.set(false);
          this.scheduleRetry();
        },
      });
  }

  disconnect(): void {
    this.clearRetry();
    this.sseSub?.unsubscribe();
    this.sseSub = null;
    this.sse.disconnect();
    this.activos.set([]);
    this.loading.set(false);
  }

  ngOnDestroy(): void {
    this.disconnect();
  }

  private scheduleRetry(): void {
    this.clearRetry();
    if (!hasAnySsePermission(this.auth.user())) return;
    this.retryTimer = setTimeout(() => this.init(), 3000);
  }

  private clearRetry(): void {
    if (this.retryTimer) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
  }
}
