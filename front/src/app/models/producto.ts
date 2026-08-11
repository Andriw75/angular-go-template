import type { Imagen } from './imagen';

export interface Producto {
  id: number;
  nombre: string;
  descripcion: string;
  precio: number;
  stock: number;
  categoria_id: number;
  activo: boolean;
  imagenes: Imagen[];
  creado_en: string;
  actualizado_en: string;
}

export interface ProductoInput {
  nombre: string;
  descripcion: string;
  precio: number;
  stock: number;
  categoria_id: number;
  activo: boolean;
}

export interface PaginatedProductos {
  data: Producto[];
  total: number;
  offset: number;
  limit: number;
}
