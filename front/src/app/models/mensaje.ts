export interface MensajePendiente {
  id: number;
  telefono: string;
  hora_solicitada: string;
  hora_desactivacion: string;
  usuario_acargo: string | null;
  hora_usuario_asignado: string | null;
  estado: string;
}

export interface MensajeInput {
  telefono: string;
  hora_desactivacion: string;
}

export interface MensajeUpdateInput {
  usuario_acargo?: string;
  finalizar?: boolean;
}

export interface PaginatedMensajes {
  data: MensajePendiente[];
  total: number;
  offset: number;
  limit: number;
}
