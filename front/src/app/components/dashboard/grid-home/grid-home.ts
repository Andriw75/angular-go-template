import { NgClass } from '@angular/common';
import { Component, computed, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { DashboardConfigService } from '../../../services/dashboard-config.service';
import { MenuService } from '../../../services/menu.service';
import { GridSubmenuComponent } from '../grid-submenu/grid-submenu';
import { ModuleCardComponent } from '../shared/module-card/module-card';
import type { MenuItem } from '../../../models/dashboard';

@Component({
  selector: 'app-grid-home',
  templateUrl: './grid-home.html',
  styleUrl: './grid-home.css',
  imports: [NgClass, ModuleCardComponent, GridSubmenuComponent],
})
export class GridHomeComponent {
  private readonly menu = inject(MenuService);
  private readonly router = inject(Router);
  private readonly config = inject(DashboardConfigService);

  readonly search = signal('');
  readonly submenu = signal<MenuItem | null>(null);
  readonly showSearch = this.config.grid.showSearch;
  readonly columns = this.config.grid.columns;

  readonly modules = computed(() => {
    const term = this.search().trim().toLowerCase();
    return this.menu.visibleMenu().filter((item) => this.matches(item, term));
  });

  readonly hasResults = computed(() => this.modules().length > 0);

  onSearch(event: Event): void {
    this.search.set((event.target as HTMLInputElement).value);
  }

  onSelect(item: MenuItem): void {
    if (item.submenu?.length) {
      this.submenu.set(item);
      return;
    }
    if (item.path) {
      void this.router.navigateByUrl(item.path);
    }
  }

  closeSubmenu(): void {
    this.submenu.set(null);
  }

  private matches(item: MenuItem, term: string): boolean {
    if (!term) return true;
    if (
      item.label.toLowerCase().includes(term) ||
      (item.description ?? '').toLowerCase().includes(term)
    ) {
      return true;
    }
    return item.submenu?.some((child) => this.matches(child, term)) ?? false;
  }
}
