import { HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { inject } from '@angular/core';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';
import { AuthService } from '../services/auth.service';
import { SessionService } from '../services/session.service';

const SKIP_PATHS = ['/auth/token', '/auth/me'];

export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  const skip = SKIP_PATHS.some((p) => req.url.includes(p));
  const auth = inject(AuthService);
  const router = inject(Router);
  const session = inject(SessionService);

  return next(req).pipe(
    catchError((err) => {
      if (err instanceof HttpErrorResponse && err.status === 401) {
        session.clear();
        if (!skip && auth.user()) {
          auth.user.set(null);
          router.navigate(['/login']);
        }
      }
      return throwError(() => err);
    }),
  );
};
