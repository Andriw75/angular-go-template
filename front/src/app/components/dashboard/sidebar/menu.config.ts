import { HomeIcon } from '../../../common/icons/home.icon';
import { FlowIcon } from '../../../common/icons/flow.icon';
import { UsersIcon } from '../../../common/icons/users.icon';
import { BusIcon } from '../../../common/icons/bus.icon';
import { MapIcon } from '../../../common/icons/map.icon';
import { SettingsIcon } from '../../../common/icons/settings.icon';
import { FileIcon } from '../../../common/icons/file.icon';
import { StoreIcon } from '../../../common/icons/store.icon';
import { CategoriesIcon } from '../../../common/icons/categories.icon';
import { ProductsIcon } from '../../../common/icons/products.icon';
import { ReportIcon } from '../../../common/icons/report.icon';
import { ReceiptIcon } from '../../../common/icons/receipt.icon';
import { DatabaseIcon } from '../../../common/icons/database.icon';
import type { MenuItem } from '../../../models/dashboard';

export const MENU: MenuItem[] = [
  {
    label: 'Inicio',
    key: 'home',
    path: '/dashboard',
    exact: true,
    group: 'General',
    description: 'Pantalla principal',
    icon: { component: HomeIcon, inputs: { size: '20' } },
  },
  {
    label: 'Flujos',
    key: 'flujos',
    path: '/dashboard/flujos',
    group: 'General',
    description: 'Diagramas de procesos',
    icon: { component: FlowIcon, inputs: { size: '20' } },
  },
  {
    label: 'Buses',
    key: 'buses',
    path: '/dashboard/buses',
    requiredPermission: 'buses',
    group: 'Operación',
    description: 'Gestión de flota',
    icon: { component: BusIcon, inputs: { size: '20' } },
  },
  {
    label: 'Mensajes',
    key: 'mensajes',
    path: '/dashboard/mensajes',
    requiredPermission: 'mensajes_pendientes',
    group: 'Operación',
    description: 'Mensajes pendientes',
    icon: { component: SettingsIcon, inputs: { size: '20' } },
  },
  {
    label: 'Mapas',
    key: 'mapas',
    path: '/dashboard/mapas',
    group: 'Operación',
    description: 'Instalaciones y ubicaciones',
    icon: { component: MapIcon, inputs: { size: '20' } },
  },
  {
    label: 'Usuarios',
    key: 'usuarios',
    path: '/dashboard/usuarios',
    requiredPermission: 'usuarios',
    group: 'Administración',
    description: 'Cuentas y permisos',
    icon: { component: UsersIcon, inputs: { size: '20' } },
  },
  {
    label: 'Archivos',
    key: 'archivos',
    requiredPermission: 'archivos',
    group: 'Administración',
    description: 'Archivos e imágenes',
    icon: { component: FileIcon, inputs: { size: '20' } },
    submenu: [
      {
        label: 'Archivos e imágenes',
        key: 'archivos-list',
        path: '/dashboard/archivos',
        requiredPermission: 'archivos',
        description: 'Subida y administración',
        icon: { component: FileIcon, inputs: { size: '18' } },
      },
      {
        label: 'Guía',
        key: 'archivos-guia',
        path: '/dashboard/archivos/guia',
        requiredPermission: 'archivos',
        description: 'Uso del gestor de archivos',
        icon: { component: FileIcon, inputs: { size: '18' } },
      },
    ],
  },
  {
    label: 'Tienda',
    key: 'shop',
    requiredPermission: 'shop',
    group: 'Administración',
    description: 'Catálogo y ventas',
    icon: { component: StoreIcon, inputs: { size: '20' } },
    submenu: [
      {
        label: 'Catálogo',
        key: 'shop-catalogo',
        icon: { component: CategoriesIcon, inputs: { size: '18' } },
        submenu: [
          {
            label: 'Categorías',
            key: 'categorias',
            path: '/dashboard/categorias',
            requiredPermission: 'shop',
            description: 'Categorías de productos',
            icon: { component: CategoriesIcon, inputs: { size: '18' } },
          },
          {
            label: 'Productos',
            key: 'productos',
            path: '/dashboard/productos',
            requiredPermission: 'shop',
            description: 'Productos del catálogo',
            icon: { component: ProductsIcon, inputs: { size: '18' } },
          },
        ],
      },
      {
        label: 'Operaciones',
        key: 'shop-operaciones',
        icon: { component: ReceiptIcon, inputs: { size: '18' } },
        submenu: [
          {
            label: 'Ventas',
            key: 'ventas',
            path: '/dashboard/ventas',
            requiredPermission: 'ventas',
            description: 'Ventas y comprobantes',
            icon: { component: ReceiptIcon, inputs: { size: '18' } },
          },
        ],
      },
    ],
  },
  {
    label: 'Reportes',
    key: 'reportes',
    path: '/dashboard/reportes',
    requiredPermission: 'reportes',
    group: 'Análisis',
    description: 'Reportes y exportaciones',
    icon: { component: ReportIcon, inputs: { size: '20' } },
  },
  {
    label: 'Laboratorio',
    key: 'laboratorio',
    path: '/dashboard/laboratorio',
    group: 'Análisis',
    description: 'Consultas y diagnóstico',
    icon: { component: DatabaseIcon, inputs: { size: '20' } },
  },
];
