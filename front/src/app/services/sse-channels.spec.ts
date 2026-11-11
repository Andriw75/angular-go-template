import { describe, expect, it } from 'vitest';
import { hasAnySseAccess } from './sse-channels';

describe('hasAnySseAccess', () => {
  it('sin usuario no hay acceso', () => {
    expect(hasAnySseAccess(null)).toBe(false);
  });

  it('un canal sin permiso exige solo sesión', () => {
    expect(hasAnySseAccess({ permisos: [] })).toBe(true);
  });

  it('reconoce el permiso de mensajes', () => {
    expect(hasAnySseAccess({ permisos: ['mensajes_pendientes'] })).toBe(true);
  });

  it('sin permisos declarados el canal abierto basta', () => {
    expect(hasAnySseAccess({})).toBe(true);
  });
});
