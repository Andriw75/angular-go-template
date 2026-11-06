import type { Type } from '@angular/core';

export type DashboardVariant = 'sidebar' | 'horizontal' | 'grid';

export type HorizontalOverflow = 'menu-scroll' | 'menu' | 'scroll';

export interface DashboardIcon {
  component: Type<unknown>;
  inputs?: Record<string, unknown>;
}

export interface MenuItem {
  label: string;
  key: string;
  path?: string;
  icon?: DashboardIcon;
  requiredPermission?: string;
  description?: string;
  group?: string;
  badge?: string;
  exact?: boolean;
  submenu?: MenuItem[];
}

export interface DashboardConfig {
  VARIANT: DashboardVariant;
  HORIZONTAL: {
    overflow: HorizontalOverflow;
    sticky: boolean;
  };
  GRID: {
    columns: number;
    showSearch: boolean;
  };
}
