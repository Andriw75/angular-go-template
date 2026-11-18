import { DestroyRef, Injectable, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import {
  NavigationCancel,
  NavigationEnd,
  NavigationError,
  NavigationStart,
  Router,
} from '@angular/router';

const BAR_DELAY_MS = 120;
const OVERLAY_DELAY_MS = 700;

@Injectable({ providedIn: 'root' })
export class RouteLoadingService {
  private readonly router = inject(Router);
  private readonly destroyRef = inject(DestroyRef);

  private barTimer: number | undefined;
  private overlayTimer: number | undefined;
  private started = false;

  readonly barVisible = signal(false);
  readonly overlayVisible = signal(false);

  init(): void {
    if (this.started) return;
    this.started = true;

    this.router.events.pipe(takeUntilDestroyed(this.destroyRef)).subscribe((event) => {
      if (event instanceof NavigationStart) {
        this.begin();
      } else if (
        event instanceof NavigationEnd ||
        event instanceof NavigationCancel ||
        event instanceof NavigationError
      ) {
        this.finish();
      }
    });

    this.destroyRef.onDestroy(() => this.clearTimers());
  }

  private begin(): void {
    this.clearTimers();
    this.barTimer = window.setTimeout(() => this.barVisible.set(true), BAR_DELAY_MS);
    this.overlayTimer = window.setTimeout(() => this.overlayVisible.set(true), OVERLAY_DELAY_MS);
  }

  private finish(): void {
    this.clearTimers();
    this.barVisible.set(false);
    this.overlayVisible.set(false);
  }

  private clearTimers(): void {
    if (this.barTimer !== undefined) {
      window.clearTimeout(this.barTimer);
      this.barTimer = undefined;
    }
    if (this.overlayTimer !== undefined) {
      window.clearTimeout(this.overlayTimer);
      this.overlayTimer = undefined;
    }
  }
}
