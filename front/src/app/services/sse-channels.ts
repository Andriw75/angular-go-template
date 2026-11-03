export interface SseChannel {
  module: string;
  name: string;
  permission: string;
}

// Canales que viajan por el SSE global. Debe coincidir con sseChannels del backend.
export const SSE_CHANNELS: SseChannel[] = [
  { module: 'mensajes_pendientes', name: 'activos', permission: 'mensajes_pendientes' },
];

export function hasAnySsePermission(user: { permisos?: string[] } | null): boolean {
  return SSE_CHANNELS.some((channel) => user?.permisos?.includes(channel.permission) ?? false);
}
