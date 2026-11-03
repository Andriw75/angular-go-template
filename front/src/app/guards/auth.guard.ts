import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { catchError, map, of } from 'rxjs';
import { AuthService } from '../services/auth.service';

export const authGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  const forceChangeIfNeeded = (mustChange: boolean): boolean => {
    if (mustChange && router.url !== '/change-password') {
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
