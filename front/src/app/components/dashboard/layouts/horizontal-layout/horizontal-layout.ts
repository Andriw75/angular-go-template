import { Component, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { filter } from 'rxjs';
import { MenuService } from '../../../../services/menu.service';
import { BreadcrumbsComponent } from '../../shared/breadcrumbs/breadcrumbs';
import { BrandLogoComponent } from '../../shared/brand-logo/brand-logo';
import { SessionBadgeComponent } from '../../shared/session-badge/session-badge';
import { UserMenuComponent } from '../../shared/user-menu/user-menu';
import { HorizontalSubnavComponent } from '../../horizontal-subnav/horizontal-subnav';
import { HorizontalTabsComponent } from '../../horizontal-tabs/horizontal-tabs';
import { MobileDrilldownComponent } from '../../mobile-drilldown/mobile-drilldown';
import type { MenuItem } from '../../../../models/dashboard';

@Component({
  selector: 'app-horizontal-layout',
  templateUrl: './horizontal-layout.html',
  styleUrl: './horizontal-layout.css',
  imports: [
    RouterOutlet,
    BrandLogoComponent,
    UserMenuComponent,
    SessionBadgeComponent,
    HorizontalTabsComponent,
    HorizontalSubnavComponent,
    BreadcrumbsComponent,
    MobileDrilldownComponent,
  ],
})
export class HorizontalLayoutComponent {
  protected readonly menu = inject(MenuService);
  private readonly router = inject(Router);

  protected readonly currentUrl = signal(this.cleanUrl(this.router.url));

  readonly activeTab = computed(() => this.menu.activeTab(this.currentUrl()));
  readonly trail = computed(() => this.menu.activeTrail(this.currentUrl()));
  readonly secondary = computed(() => this.activeTab()?.submenu ?? []);

  constructor() {
    this.router.events
      .pipe(
        filter((event): event is NavigationEnd => event instanceof NavigationEnd),
        takeUntilDestroyed(),
      )
      .subscribe(() => this.currentUrl.set(this.cleanUrl(this.router.url)));
  }

  onSelect(item: MenuItem): void {
    const leaf = this.menu.firstLeaf(item);
    if (leaf?.path) {
      void this.router.navigateByUrl(leaf.path);
    }
  }

  private cleanUrl(url: string): string {
    return url.split('?')[0].split('#')[0];
  }
}
