import { HttpErrorResponse } from '@angular/common/http';

export function apiErrorMessage(err: unknown, fallback: string): string {
  if (err instanceof HttpErrorResponse) {
    const body = err.error;

    if (body && typeof body === 'object' && typeof body.error === 'string' && body.error.trim()) {
      return body.error;
    }
    if (typeof body === 'string' && body.trim()) {
      return body;
    }
    if (err.status === 0) {
      return 'No se pudo conectar con el servidor.';
    }
    if (err.status === 401) {
      return 'Sesión expirada. Vuelve a iniciar sesión.';
    }
    if (err.status === 403) {
      return 'No tienes permisos para esta acción.';
    }
  }

  return fallback;
}
