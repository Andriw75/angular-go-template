import { type Type } from '@angular/core';
import { HomeIcon } from '../../../common/icons/home.icon';
import { UsersIcon } from '../../../common/icons/users.icon';
import { BusIcon } from '../../../common/icons/bus.icon';
import { MapIcon } from '../../../common/icons/map.icon';
import { SettingsIcon } from '../../../common/icons/settings.icon';
import { FileIcon } from '../../../common/icons/file.icon';
import { StoreIcon } from '../../../common/icons/store.icon';
import { CategoriesIcon } from '../../../common/icons/categories.icon';
import { ProductsIcon } from '../../../common/icons/products.icon';

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
    requiredPermission: 'buses',
    icon: { component: BusIcon, inputs: { size: '20' } },
  },
  {
    label: 'Mensajes',
    key: 'mensajes',
    path: '/dashboard/mensajes',
    requiredPermission: 'mensajes_pendientes',
    icon: { component: SettingsIcon, inputs: { size: '20' } },
  },
  {
    label: 'Mapas',
    key: 'mapas',
    path: '/dashboard/mapas',
    icon: { component: MapIcon, inputs: { size: '20' } },
  },
  {
    label: 'Usuarios',
    key: 'usuarios',
    path: '/dashboard/usuarios',
    requiredPermission: 'usuarios',
    icon: { component: UsersIcon, inputs: { size: '20' } },
  },
  {
    label: 'Archivos',
    key: 'archivos',
    requiredPermission: 'archivos',
    icon: { component: FileIcon, inputs: { size: '20' } },
    submenu: [
      {
        label: 'Archivos e imágenes',
        key: 'archivos-list',
        path: '/dashboard/archivos',
        requiredPermission: 'archivos',
        icon: { component: FileIcon, inputs: { size: '18' } },
      },
      {
        label: 'Guía',
        key: 'archivos-guia',
        path: '/dashboard/archivos/guia',
        requiredPermission: 'archivos',
      },
    ],
  },
  {
    label: 'Tienda',
    key: 'shop',
    requiredPermission: 'shop',
    icon: { component: StoreIcon, inputs: { size: '20' } },
    submenu: [
      {
        label: 'Categorias',
        key: 'categorias',
        path: '/dashboard/categorias',
        requiredPermission: 'shop',
        icon: { component: CategoriesIcon, inputs: { size: '18' } },
      },
      {
        label: 'Productos',
        key: 'productos',
        path: '/dashboard/productos',
        requiredPermission: 'shop',
        icon: { component: ProductsIcon, inputs: { size: '18' } },
      },
    ],
  },
];
