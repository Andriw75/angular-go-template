import { type Type } from '@angular/core';
import { HomeIcon } from '../../../common/icons/home.icon';
import { UsersIcon } from '../../../common/icons/users.icon';
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
    label: 'Usuarios',
    key: 'usuarios',
    path: '/dashboard/usuarios',
    requiredPermission: 'usuarios',
    icon: { component: UsersIcon, inputs: { size: '20' } },
  },
];
