import type { Bus } from './bus';
import type { Venta } from './venta';

export interface ReporteBusesResumen {
  total: number;
  activos: number;
  inactivos: number;
  precio_total: number;
  peso_total: number;
  capacidad_promedio: number;
}

export interface ReporteBusesTipo {
  tipo: string;
  cantidad: number;
  precio_total: number;
  capacidad_promedio: number;
}

export interface ReporteBuses {
  generado_en: string;
  filtros: Record<string, unknown>;
  resumen: ReporteBusesResumen;
  por_tipo: ReporteBusesTipo[];
  data: Bus[];
}

export interface ReporteVentasResumen {
  total: number;
  emitidas: number;
  anuladas: number;
  monto_total: number;
  impuesto_total: number;
  clientes: number;
}

export interface ReporteVentas {
  generado_en: string;
  filtros: Record<string, unknown>;
  resumen: ReporteVentasResumen;
  data: Venta[];
}

export interface DocumentoArchivo {
  id: number;
  formato: 'pdf' | 'xlsx';
  file_name: string;
  mime: string;
  tamano: number;
  version: number;
  url: string;
  creado_en: string;
}

export interface DocumentoEvento {
  id: number;
  accion: string;
  detalle: string;
  user_id: number;
  creado_en: string;
}

export interface Documento {
  id: number;
  tipo: string;
  serie: string;
  anio: number;
  correlativo: number;
  codigo: string;
  politica: 'inmutable' | 'actualizable';
  estado: 'emitido' | 'anulado' | 'borrador';
  plantilla: string;
  plantilla_version: string;
  version: number;
  creado_en: string;
  actualizado_en: string;
  emitido_en: string;
  anulado_en?: string;
  motivo_anulacion?: string;
  archivos: DocumentoArchivo[];
  eventos: DocumentoEvento[];
}

export interface DocumentoResult {
  job_id?: string;
  exists?: boolean;
  documento?: Documento;
  estado?: string;
  progreso?: number;
  id?: string;
  error?: string;
}
