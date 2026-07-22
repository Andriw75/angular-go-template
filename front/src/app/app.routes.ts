import { Routes } from '@angular/router';
import { authGuard } from './guards/auth.guard';
import { loginGuard } from './guards/login.guard';

export const routes: Routes = [
  {
    path: 'login',
    canActivate: [loginGuard],
    loadComponent: () => import('./components/login/login').then((m) => m.LoginComponent),
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
        path: 'buses',
        loadComponent: () =>
          import('./components/dashboard/buses/list').then((m) => m.BusesListComponent),
      },
      {
        path: 'usuarios',
        loadComponent: () =>
          import('./components/dashboard/usuarios/list').then((m) => m.UsersListComponent),
      },
      {
        path: '**',
        loadComponent: () =>
          import('./components/dashboard/home/home').then((m) => m.HomeComponent),
      },
      {
        path: '**',
        loadComponent: () => import('./components/dashboard/home/home').then((m) => m.HomeComponent),
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
