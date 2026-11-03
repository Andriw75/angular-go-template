import { inject } from '@angular/core';
import { CanActivateFn, Router } from '@angular/router';
import { catchError, map, of } from 'rxjs';
import { AuthService } from '../services/auth.service';

function redirectLoggedUser(auth: AuthService, router: Router): void {
  if (auth.user()?.must_change_password) {
    router.navigate(['/change-password']);
  } else {
    router.navigate(['/dashboard']);
  }
}

export const loginGuard: CanActivateFn = () => {
  const auth = inject(AuthService);
  const router = inject(Router);

  if (auth.user()) {
    redirectLoggedUser(auth, router);
    return of(false);
  }

  return auth.me().pipe(
    map((user) => {
      auth.user.set(user);
      redirectLoggedUser(auth, router);
      return false;
    }),
    catchError(() => of(true)),
  );
};
