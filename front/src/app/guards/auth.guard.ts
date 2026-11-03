import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { catchError, map, of } from 'rxjs';
import { AuthService } from '../services/auth.service';

export const authGuard: CanActivateFn = (_route, state) => {
  const auth = inject(AuthService);
  const router = inject(Router);

  // state.url es la URL destino; router.url todavía apunta a la anterior
  // durante la evaluación del guard.
  const target = state.url.split('?')[0];
  const forceChangeIfNeeded = (mustChange: boolean): boolean => {
    if (mustChange && target !== '/change-password') {
      router.navigate(['/change-password']);
      return false;
    }
    return true;
  };

  if (auth.user()) {
    return of(forceChangeIfNeeded(auth.user()!.must_change_password));
  }

  return auth.me().pipe(
    map((user) => {
      auth.user.set(user);
      return forceChangeIfNeeded(user.must_change_password);
    }),
    catchError(() => {
      router.navigate(['/login']);
      return of(false);
    }),
  );
};
