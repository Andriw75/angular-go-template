import type { Imagen } from './imagen';

export interface Categoria {
  id: number;
  nombre: string;
  descripcion: string;
  activo: boolean;
  imagenes: Imagen[];
  creado_en: string;
  actualizado_en: string;
}

export interface CategoriaInput {
  nombre: string;
  descripcion: string;
  activo: boolean;
}

export interface PaginatedCategorias {
  data: Categoria[];
  total: number;
  offset: number;
  limit: number;
}
