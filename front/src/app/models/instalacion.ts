export const INSTALACION_TIPOS = ['CAMARA', 'PANEL', 'BANNER', 'SENSOR', 'ROUTER'] as const;
export type InstalacionTipo = (typeof INSTALACION_TIPOS)[number];

export const INSTALACION_ESTADOS = [
  'ACTIVO',
  'INACTIVO',
  'MANTENIMIENTO',
  'MALOGRADO',
  'PENDIENTE_INSTALACION',
  'PENDIENTE_REPARACION',
] as const;
export type InstalacionEstado = (typeof INSTALACION_ESTADOS)[number];

export const INSTALACION_ZONAS = ['CENTRO', 'NORTE', 'SUR', 'ESTE', 'OESTE'] as const;
export type InstalacionZona = (typeof INSTALACION_ZONAS)[number];

export interface Instalacion {
  id: number;
  tipo: InstalacionTipo;
  nombre: string;
  estado: InstalacionEstado;
  lat: number;
  lng: number;
  direccion: string;
  zona: InstalacionZona;
  fecha_instalacion: string;
  serial: string;

  descripcion?: string;
  fabricante?: string;
  modelo?: string;
  instalador?: string;
  proveedor?: string;
  ultima_conexion?: string | null;
  firmware?: string;
  ip?: string;
  mac?: string;
  resolucion?: string;
  contenido?: string;
  senal?: number;
  bateria?: number;
  potencia?: number;
  contacto?: string;
}

export const ESTADO_CONFIG: Record<InstalacionEstado, { label: string; color: string; background: string }> = {
  ACTIVO: { label: 'Activo', color: '#15803d', background: '#dcfce7' },
  INACTIVO: { label: 'Inactivo', color: '#475569', background: '#e2e8f0' },
  MANTENIMIENTO: { label: 'Mantenimiento', color: '#b45309', background: '#fef3c7' },
  MALOGRADO: { label: 'Malogrado', color: '#b91c1c', background: '#fee2e2' },
  PENDIENTE_INSTALACION: { label: 'Pend. instalación', color: '#1d4ed8', background: '#dbeafe' },
  PENDIENTE_REPARACION: { label: 'Pend. reparación', color: '#c2410c', background: '#ffedd5' },
};