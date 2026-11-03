import { Injectable } from '@angular/core';
import { Subject, type Observable } from 'rxjs';
import { environment } from '../../environments/environment';

export interface SSEEvent<T = any> {
  module: string;
  name: string;
  type: string;
  data: T;
}

@Injectable({ providedIn: 'root' })
export class SSEService {
  private source: EventSource | null = null;
  private subject = new Subject<SSEEvent>();
  private enabled = false;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private retryDelay = 3000;

  readonly events$: Observable<SSEEvent> = this.subject.asObservable();

  // Abre o cierra la conexion global segun los permisos/canales disponibles.
  setEnabled(enabled: boolean): void {
    this.enabled = enabled;
    if (enabled) this.open();
    else this.close();
  }

  disconnect(): void {
    this.enabled = false;
    this.close();
  }

  private open(): void {
    if (this.source || !this.enabled) return;
    this.clearRetry();

    const source = new EventSource(`${environment.API_URL}/sse/events`, { withCredentials: true });
    this.source = source;

    source.onmessage = (event) => {
      try {
        const parsed: SSEEvent = JSON.parse(event.data);
        this.retryDelay = 3000;
        this.subject.next(parsed);
      } catch {
        // ignore parse errors
      }
    };

    source.onerror = () => {
      this.scheduleReconnect();
    };
  }

  private scheduleReconnect(): void {
    this.closeSource();
    if (!this.enabled) return;

    this.clearRetry();
    const delay = this.retryDelay;
    this.retryDelay = Math.min(this.retryDelay * 2, 30000);
    this.retryTimer = setTimeout(() => {
      this.retryTimer = null;
      this.open();
    }, delay);
  }

  private close(): void {
    this.enabled = false;
    this.clearRetry();
    this.closeSource();
    this.retryDelay = 3000;
  }

  private closeSource(): void {
    if (this.source) {
      this.source.close();
      this.source = null;
    }
  }

  private clearRetry(): void {
    if (this.retryTimer) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
  }
}
