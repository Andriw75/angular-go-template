import { inject } from '@angular/core';
import { type CanActivateFn, Router } from '@angular/router';
import { AuthService } from '../services/auth.service';
import { ToastService } from '../services/toast.service';

export const permissionGuard: CanActivateFn = (route) => {
  const auth = inject(AuthService);
  const router = inject(Router);
  const toast = inject(ToastService);

  const required = route.data?.['permission'] as string | undefined;
  if (!required || auth.hasPermission(required)) {
    return true;
  }

  toast.error('No tienes permiso para acceder a esta sección');
  return router.createUrlTree(['/dashboard']);
};
