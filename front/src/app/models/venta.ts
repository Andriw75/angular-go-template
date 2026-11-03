export interface VentaItem {
  id: number;
  producto_id?: number;
  descripcion: string;
  cantidad: number;
  precio_unitario: number;
  descuento: number;
  impuesto_porcentaje: number;
  subtotal: number;
  impuesto: number;
  total: number;
}

export interface Venta {
  id: number;
  serie: string;
  correlativo: number;
  codigo: string;
  cliente_nombre: string;
  cliente_documento: string;
  cliente_direccion: string;
  fecha: string;
  moneda: string;
  subtotal: number;
  descuento: number;
  impuesto: number;
  total: number;
  estado: 'borrador' | 'emitido' | 'anulado';
  observaciones: string;
  creado_en: string;
  actualizado_en: string;
  items: VentaItem[];
}

export interface VentaItemInput {
  producto_id?: number;
  descripcion: string;
  cantidad: number;
  precio_unitario: number;
  descuento: number;
  impuesto_porcentaje: number;
}

export interface VentaInput {
  serie: string;
  cliente_nombre: string;
  cliente_documento: string;
  cliente_direccion: string;
  fecha: string;
  moneda: string;
  observaciones: string;
  items: VentaItemInput[];
}

export interface PaginatedVentas {
  data: Venta[];
  total: number;
  offset: number;
  limit: number;
}
