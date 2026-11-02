export type Visibilidad = 'publico' | 'autenticado' | 'propietario';

export interface Archivo {
  id: number;
  url: string;
  nombre: string;
  tipo: string;
  tamano: number;
  visibilidad: Visibilidad;
  publico: boolean;
  creado_en: string;
}

export interface PaginatedArchivos {
  data: Archivo[];
  total: number;
}
