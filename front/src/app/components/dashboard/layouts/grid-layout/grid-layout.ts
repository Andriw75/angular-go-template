import { Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs';
import { MenuService } from '../../../../services/menu.service';
import { BrandLogoComponent } from '../../shared/brand-logo/brand-logo';
import { SessionBadgeComponent } from '../../shared/session-badge/session-badge';
import { UserMenuComponent } from '../../shared/user-menu/user-menu';

@Component({
  selector: 'app-grid-layout',
  templateUrl: './grid-layout.html',
  styleUrl: './grid-layout.css',
  imports: [RouterOutlet, BrandLogoComponent, SessionBadgeComponent, UserMenuComponent],
})
export class GridLayoutComponent {
  private readonly menu = inject(MenuService);
  private readonly router = inject(Router);

  protected readonly currentUrl = signal(this.cleanUrl(this.router.url));
  readonly isHome = computed(() => {
    const url = this.currentUrl();
    return url === '/dashboard' || url === '/dashboard/';
  });
  readonly activeModule = computed(() => this.menu.activeTrail(this.currentUrl()).at(-1));

  constructor() {
    this.router.events
      .pipe(
        filter((event): event is NavigationEnd => event instanceof NavigationEnd),
        takeUntilDestroyed(),
      )
      .subscribe(() => this.currentUrl.set(this.cleanUrl(this.router.url)));
  }

  close(): void {
    void this.router.navigateByUrl('/dashboard');
  }

  private cleanUrl(url: string): string {
    return url.split('?')[0].split('#')[0];
  }
}
