import { type Type } from '@angular/core';
import { HomeIcon } from '../../../common/icons/home.icon';
import { UsersIcon } from '../../../common/icons/users.icon';
import { SettingsIcon } from '../../../common/icons/settings.icon';
import { DatabaseIcon } from '../../../common/icons/database.icon';
import { BusIcon } from '../../../common/icons/bus.icon';

export interface IconConfig {
  component: Type<any>;
  inputs?: Record<string, any>;
}

export interface MenuItem {
  label: string;
  key: string;
  path?: string;
  icon?: IconConfig;
  requiredPermission?: string;
  submenu?: MenuItem[];
}

export const MENU: MenuItem[] = [
  {
    label: 'Inicio',
    key: 'home',
    path: '/dashboard',
    icon: { component: HomeIcon, inputs: { size: '20' } },
  },
  {
    label: 'Buses',
    key: 'buses',
    path: '/dashboard/buses',
    icon: { component: BusIcon, inputs: { size: '20' } },
  },
  {
    label: 'Admin',
    key: 'admin',
    requiredPermission: 'admin',
    icon: { component: UsersIcon, inputs: { size: '20' } },
    submenu: [
      {
        label: 'Personal',
        key: 'personal',
        path: '/dashboard/usuarios/personal',
        icon: { component: UsersIcon, inputs: { size: '16' } },
        requiredPermission: 'admin',
      },
      {
        label: 'Permisos',
        key: 'permisos',
        path: '/dashboard/usuarios/permisos',
        icon: { component: SettingsIcon, inputs: { size: '16' } },
        requiredPermission: 'admin',
      },
    ],
  },
  {
    label: 'Datos',
    key: 'datos',
    icon: { component: DatabaseIcon, inputs: { size: '20' } },
    submenu: [
      {
        label: 'Reportes',
        key: 'reportes',
        path: '/dashboard/datos/reportes',
        icon: { component: DatabaseIcon, inputs: { size: '16' } },
      },
    ],
  },
];
