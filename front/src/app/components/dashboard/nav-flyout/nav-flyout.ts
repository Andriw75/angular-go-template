import { Component, HostListener, inject, input, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterLink, RouterLinkActive } from '@angular/router';
import { filter } from 'rxjs';
import { MenuService } from '../../../services/menu.service';
import type { MenuItem } from '../../../models/dashboard';

@Component({
  selector: 'app-nav-flyout',
  templateUrl: './nav-flyout.html',
  styleUrl: './nav-flyout.css',
  imports: [RouterLink, RouterLinkActive, NavFlyoutComponent],
})
export class NavFlyoutComponent {
  private readonly menu = inject(MenuService);
  private readonly router = inject(Router);

  readonly item = input.required<MenuItem>();
  readonly activeUrl = input('');
  readonly open = signal(false);

  private closeTimer: ReturnType<typeof setTimeout> | null = null;

  constructor() {
    this.router.events
      .pipe(
        filter((event): event is NavigationEnd => event instanceof NavigationEnd),
        takeUntilDestroyed(),
      )
      .subscribe(() => this.open.set(false));
  }

  toggle(): void {
    this.open.set(true);
  }

  isActive(item: MenuItem): boolean {
    return this.menu.isActive(item, this.activeUrl());
  }

  hasActiveChild(item: MenuItem): boolean {
    if (this.isActive(item)) return true;
    return item.submenu?.some((child) => this.hasActiveChild(child)) ?? false;
  }

  onMouseEnter(): void {
    if (!this.item().submenu?.length) return;
    this.clearClose();
    this.open.set(true);
  }

  onMouseLeave(): void {
    if (!this.item().submenu?.length) return;
    this.clearClose();
    this.closeTimer = setTimeout(() => this.open.set(false), 150);
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    const host = (event.target as HTMLElement).closest('app-nav-flyout');
    if (!host) this.open.set(false);
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.open.set(false);
  }

  private clearClose(): void {
    if (this.closeTimer) {
      clearTimeout(this.closeTimer);
      this.closeTimer = null;
    }
  }
}
