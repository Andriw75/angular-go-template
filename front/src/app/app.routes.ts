import { Routes } from '@angular/router';
import { authGuard } from './guards/auth.guard';
import { loginGuard } from './guards/login.guard';
import { permissionGuard } from './guards/permission.guard';

export const routes: Routes = [
  {
    path: 'login',
    canActivate: [loginGuard],
    loadComponent: () => import('./components/login/login').then((m) => m.LoginComponent),
  },
  {
    path: 'change-password',
    canActivate: [authGuard],
    loadComponent: () =>
      import('./components/change-password/change-password').then((m) => m.ChangePasswordComponent),
  },
  {
    path: 'dashboard',
    canActivate: [authGuard],
    loadComponent: () => import('./components/dashboard/layout/layout').then((m) => m.DashboardLayoutComponent),
    children: [
      {
        path: '',
        loadComponent: () => import('./components/dashboard/home/home').then((m) => m.HomeComponent),
      },
      {
        path: 'flujos',
        loadComponent: () =>
          import('./components/dashboard/flujos/flujos').then((m) => m.FlujosComponent),
      },
      {
        path: 'buses',
        canActivate: [permissionGuard],
        data: { permission: 'buses' },
        loadComponent: () =>
          import('./components/dashboard/buses/list').then((m) => m.BusesListComponent),
      },
      {
        path: 'usuarios',
        canActivate: [permissionGuard],
        data: { permission: 'usuarios' },
        loadComponent: () =>
          import('./components/dashboard/usuarios/list').then((m) => m.UsersListComponent),
      },
      {
        path: 'mensajes',
        canActivate: [permissionGuard],
        data: { permission: 'mensajes_pendientes' },
        loadComponent: () =>
          import('./components/dashboard/mensajes/list').then((m) => m.MensajesListComponent),
      },
      {
        path: 'mapas',
        loadComponent: () =>
          import('./components/dashboard/mapas/mapa').then((m) => m.MapasComponent),
      },
      {
        path: 'categorias',
        canActivate: [permissionGuard],
        data: { permission: 'shop' },
        loadComponent: () =>
          import('./components/dashboard/shop/categorias/list').then((m) => m.CategoriasListComponent),
      },
      {
        path: 'productos',
        canActivate: [permissionGuard],
        data: { permission: 'shop' },
        loadComponent: () =>
          import('./components/dashboard/shop/productos/list').then((m) => m.ProductosListComponent),
      },
      {
        path: 'archivos',
        canActivate: [permissionGuard],
        data: { permission: 'archivos' },
        loadComponent: () =>
          import('./components/dashboard/archivos/list').then((m) => m.ArchivosListComponent),
      },
      {
        path: 'archivos/guia',
        canActivate: [permissionGuard],
        data: { permission: 'archivos' },
        loadComponent: () =>
          import('./components/dashboard/archivos/info').then((m) => m.ArchivosInfoComponent),
      },
      {
        path: 'ventas',
        canActivate: [permissionGuard],
        data: { permission: 'ventas' },
        loadComponent: () =>
          import('./components/dashboard/ventas/list').then((m) => m.VentasListComponent),
      },
      {
        path: 'reportes',
        canActivate: [permissionGuard],
        data: { permission: 'reportes' },
        loadComponent: () =>
          import('./components/dashboard/reportes/reportes').then((m) => m.ReportesComponent),
      },
      {
        path: 'laboratorio',
        loadComponent: () =>
          import('./components/dashboard/laboratorio/laboratorio').then((m) => m.LaboratorioComponent),
      },
      {
        path: '**',
        loadComponent: () =>
          import('./components/dashboard/home/home').then((m) => m.HomeComponent),
      },
    ],
  },
  {
    path: '',
    pathMatch: 'full',
    redirectTo: 'login',
  },
  {
    path: '**',
    redirectTo: 'login',
  },
];
