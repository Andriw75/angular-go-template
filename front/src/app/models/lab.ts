export type SlowQueryModo = 'sleep' | 'cpu';

export type SlowQueryEstado =
  | 'ejecutando'
  | 'cancelando'
  | 'cancelada'
  | 'completada'
  | 'error';

export interface SlowQuery {
  id: string;
  owner_id: number;
  modo: SlowQueryModo;
  pid: number;
  estado: SlowQueryEstado;
  segundos: number;
  filas_solicitadas: number;
  filas_procesadas: number;
  iniciado_en: string;
  finalizado_en?: string;
  error?: string;
}

export interface SlowQueryInput {
  modo: SlowQueryModo;
  segundos?: number;
  filas?: number;
}
