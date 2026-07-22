import { inject } from '@angular/core';
import { type HttpInterceptorFn, HttpErrorResponse } from '@angular/common/http';
import { Router } from '@angular/router';
import { catchError, throwError } from 'rxjs';

export const errorInterceptor: HttpInterceptorFn = (req, next) => {
  const router = inject(Router);

  return next(req).pipe(
    catchError((error) => {
      if (error instanceof HttpErrorResponse && error.status === 401) {
        const skip = req.url.includes('/auth/token') || req.url.includes('/auth/me');
        if (!skip) {
          router.navigate(['/login']);
        }
      }
      return throwError(() => error);
    }),
  );
};
