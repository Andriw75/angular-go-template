import { Injectable, computed, signal } from '@angular/core';
import { Subject } from 'rxjs';
import { environment } from '../../environments/environment';

@Injectable({ providedIn: 'root' })
export class SessionService {
  private expiresAtMs: number | null = null;
  private timer: ReturnType<typeof setInterval> | null = null;
  private warnSeconds = environment.SESSION.WARN_SECONDS;
  private expiredEmitted = false;

  readonly remaining = signal<number | null>(null);
  readonly active = computed(() => this.remaining() !== null);
  readonly warning = computed(() => {
    const r = this.remaining();
    return r !== null && r > 0 && r <= this.warnSeconds;
  });
  readonly critical = computed(() => {
    const r = this.remaining();
    return r !== null && r > 0 && r <= 10;
  });
  readonly label = computed(() => formatSeconds(this.remaining() ?? 0));

  private readonly expiredSubject = new Subject<void>();
  readonly expired$ = this.expiredSubject.asObservable();

  constructor() {
    if (typeof document !== 'undefined') {
      document.addEventListener('visibilitychange', () => {
        if (document.visibilityState === 'visible') {
          this.tick();
        }
      });
    }
  }

  updateFromHeaders(expiresIn: string | null): void {
    if (!environment.SESSION.COUNTDOWN_ENABLED || expiresIn === null) return;

    const seconds = Number(expiresIn);
    if (!Number.isFinite(seconds) || seconds <= 0) {
      this.clear();
      return;
    }

    this.expiresAtMs = Date.now() + seconds * 1000;
    this.expiredEmitted = false;
    this.tick();
    this.startTimer();
  }

  clear(): void {
    this.stopTimer();
    this.expiresAtMs = null;
    this.expiredEmitted = false;
    this.remaining.set(null);
  }

  private startTimer(): void {
    if (this.timer !== null) return;
    this.timer = setInterval(() => this.tick(), 1000);
  }

  private stopTimer(): void {
    if (this.timer === null) return;
    clearInterval(this.timer);
    this.timer = null;
  }

  private tick(): void {
    if (this.expiresAtMs === null) return;

    const remaining = Math.max(0, Math.ceil((this.expiresAtMs - Date.now()) / 1000));
    this.remaining.set(remaining);

    if (remaining <= 0 && !this.expiredEmitted) {
      this.expiredEmitted = true;
      this.stopTimer();
      this.expiredSubject.next();
    }
  }
}

function formatSeconds(total: number): string {
  const seconds = Math.max(0, Math.floor(total));
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const rest = seconds % 60;
  const pad = (value: number): string => value.toString().padStart(2, '0');

  return hours > 0
    ? `${hours}:${pad(minutes)}:${pad(rest)}`
    : `${pad(minutes)}:${pad(rest)}`;
}
