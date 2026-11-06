import { Component, HostListener, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router } from '@angular/router';
import { filter } from 'rxjs';
import { MenuService } from '../../../services/menu.service';
import type { MenuItem } from '../../../models/dashboard';

interface DrilldownLevel {
  title: string;
  items: MenuItem[];
}

@Component({
  selector: 'app-mobile-drilldown',
  templateUrl: './mobile-drilldown.html',
  styleUrl: './mobile-drilldown.css',
})
export class MobileDrilldownComponent {
  private readonly menu = inject(MenuService);
  private readonly router = inject(Router);

  readonly open = signal(false);
  readonly levels = signal<DrilldownLevel[]>([]);
  readonly current = computed<DrilldownLevel>(
    () => this.levels().at(-1) ?? { title: 'Módulos', items: [] },
  );

  constructor() {
    this.router.events
      .pipe(
        filter((event): event is NavigationEnd => event instanceof NavigationEnd),
        takeUntilDestroyed(),
      )
      .subscribe(() => this.close());
  }

  openMenu(): void {
    this.levels.set([{ title: 'Módulos', items: this.menu.visibleMenu() }]);
    this.open.set(true);
  }

  select(item: MenuItem): void {
    if (item.submenu?.length) {
      this.levels.update((levels) => [
        ...levels,
        { title: item.label, items: item.submenu ?? [] },
      ]);
      return;
    }
    if (item.path) {
      this.close();
      void this.router.navigateByUrl(item.path);
    }
  }

  back(): void {
    this.levels.update((levels) => levels.slice(0, -1));
  }

  close(): void {
    this.open.set(false);
    this.levels.set([]);
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.close();
  }
}
