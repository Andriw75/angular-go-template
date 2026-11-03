export interface SseChannel {
  module: string;
  name: string;
  permission?: string | null;
}

// Canales que viajan por el SSE global. Debe coincidir con sseChannels del backend.
// permission vacio/null => solo requiere sesion autenticada.
export const SSE_CHANNELS: SseChannel[] = [
  { module: 'mensajes_pendientes', name: 'activos', permission: 'mensajes_pendientes' },
  { module: 'laboratorio', name: 'tareas', permission: null },
];

export function hasAnySseAccess(user: { permisos?: string[] } | null): boolean {
  if (!user) return false;
  return SSE_CHANNELS.some(
    (channel) => !channel.permission || user.permisos?.includes(channel.permission) === true,
  );
}
