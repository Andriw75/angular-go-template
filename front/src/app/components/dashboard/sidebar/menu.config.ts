export interface MenuItem {
  label: string;
  path: string;
  icon?: string;
  permission?: string;
  children?: MenuItem[];
}

export const MENU_ITEMS: MenuItem[] = [
  {
    label: 'Inicio',
    path: '/dashboard',
    icon: 'home',
  },
  {
    label: 'Usuarios',
    path: '/dashboard/usuarios',
    icon: 'users',
    permission: 'admin',
    children: [
      { label: 'Personal', path: '/dashboard/usuarios/personal', permission: 'admin' },
      { label: 'Permisos', path: '/dashboard/usuarios/permisos', permission: 'admin' },
    ],
  },
];
