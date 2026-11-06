import { Injectable, computed, inject } from '@angular/core';
import { MENU } from '../components/dashboard/sidebar/menu.config';
import type { MenuItem } from '../models/dashboard';
import { AuthService } from './auth.service';

@Injectable({ providedIn: 'root' })
export class MenuService {
  private readonly auth = inject(AuthService);

  readonly visibleMenu = computed(() =>
    this.filterItems(MENU, this.auth.user()?.permisos ?? []),
  );

  activeTab(url: string): MenuItem | undefined {
    const clean = this.cleanUrl(url);
    return this.visibleMenu().find((item) => this.containsPath(item, clean));
  }

  activeTrail(url: string): MenuItem[] {
    const clean = this.cleanUrl(url);
    for (const item of this.visibleMenu()) {
      if (this.isActive(item, clean)) return [item];
      if (item.submenu) {
        const trail = this.findTrail(item.submenu, clean);
        if (trail.length > 0) return [item, ...trail];
      }
    }
    return [];
  }

  isActive(item: MenuItem, url: string): boolean {
    if (!item.path) return false;
    const clean = this.cleanUrl(url);
    if (item.exact) return clean === item.path;
    return clean === item.path || clean.startsWith(item.path + '/');
  }

  firstLeaf(item: MenuItem): MenuItem | undefined {
    if (item.path) return item;
    for (const child of item.submenu ?? []) {
      const leaf = this.firstLeaf(child);
      if (leaf) return leaf;
    }
    return undefined;
  }

  private containsPath(item: MenuItem, url: string): boolean {
    if (this.isActive(item, url)) return true;
    return item.submenu?.some((child) => this.containsPath(child, url)) ?? false;
  }

  private findTrail(items: MenuItem[], url: string): MenuItem[] {
    for (const item of items) {
      if (this.isActive(item, url)) return [item];
      if (item.submenu) {
        const trail = this.findTrail(item.submenu, url);
        if (trail.length > 0) return [item, ...trail];
      }
    }
    return [];
  }

  private filterItems(items: MenuItem[], permissions: string[]): MenuItem[] {
    return items
      .map((item) =>
        item.submenu
          ? { ...item, submenu: this.filterItems(item.submenu, permissions) }
          : item,
      )
      .filter((item) => {
        if (item.submenu) return item.submenu.length > 0 || !!item.path;
        return !item.requiredPermission || permissions.includes(item.requiredPermission);
      });
  }

  private cleanUrl(url: string): string {
    return url.split('?')[0].split('#')[0];
  }
}
