import { describe, expect, it } from 'vitest';
import { HttpErrorResponse } from '@angular/common/http';
import { apiErrorMessage } from './api-error';

describe('apiErrorMessage', () => {
  it('usa body.error cuando el backend envía un mensaje', () => {
    const err = new HttpErrorResponse({ status: 422, error: { error: 'dato inválido' } });
    expect(apiErrorMessage(err, 'fallback')).toBe('dato inválido');
  });

  it('usa el body de texto plano', () => {
    const err = new HttpErrorResponse({ status: 400, error: 'texto del servidor' });
    expect(apiErrorMessage(err, 'fallback')).toBe('texto del servidor');
  });

  it('traduce 401 y 403', () => {
    expect(apiErrorMessage(new HttpErrorResponse({ status: 401 }), 'x')).toContain('Sesión');
    expect(apiErrorMessage(new HttpErrorResponse({ status: 403 }), 'x')).toContain('permisos');
  });

  it('cae al fallback con errores desconocidos', () => {
    expect(apiErrorMessage(new Error('boom'), 'fallback')).toBe('fallback');
    expect(apiErrorMessage(null, 'fallback')).toBe('fallback');
  });
});
