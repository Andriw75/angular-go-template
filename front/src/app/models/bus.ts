export interface Bus {
  id: number;
  placa: string;
  nombre: string;
  marca: string;
  modelo: string;
  anio: number;
  capacidad: number;
  tipo: string;
  activo: boolean;
  fecha_compra: string;
  ultimo_mantenimiento: string | null;
  precio: number;
  peso: number;
  color: string;
  descripcion: string;
  creado_en: string;
  actualizado_en: string;
}

export interface BusInput {
  placa: string;
  nombre: string;
  marca: string;
  modelo: string;
  anio: number;
  capacidad: number;
  tipo: string;
  activo: boolean;
  fecha_compra: string;
  ultimo_mantenimiento: string | null;
  precio: number;
  peso: number;
  color: string;
  descripcion: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  offset: number;
  limit: number;
}
