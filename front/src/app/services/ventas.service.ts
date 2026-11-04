import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { Venta, VentaInput, PaginatedVentas } from '../models/venta';

export interface VentaListParams {
  offset?: number;
  limit?: number;
  q?: string;
  serie?: string;
  estado?: string;
  desde?: string;
  hasta?: string;
}

export interface VentaCountParams {
  q?: string;
  serie?: string;
  estado?: string;
  desde?: string;
  hasta?: string;
}

@Injectable({ providedIn: 'root' })
export class VentasService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  count(params: VentaCountParams): Observable<number> {
    return this.http.get<number>(`${this.api}/ventas/count?${this.buildParams(params).toString()}`);
  }

  list(params: VentaListParams): Observable<PaginatedVentas> {
    return this.http.get<PaginatedVentas>(`${this.api}/ventas?${this.buildParams(params).toString()}`);
  }

  private buildParams(params: VentaListParams): URLSearchParams {
    const p = new URLSearchParams();
    if (params.offset != null) p.set('offset', String(params.offset));
    if (params.limit != null) p.set('limit', String(params.limit));
    if (params.q) p.set('q', params.q);
    if (params.serie) p.set('serie', params.serie);
    if (params.estado) p.set('estado', params.estado);
    if (params.desde) p.set('desde', params.desde);
    if (params.hasta) p.set('hasta', params.hasta);
    return p;
  }

  getByID(id: number): Observable<Venta> {
    return this.http.get<Venta>(`${this.api}/ventas/${id}`);
  }

  create(input: VentaInput): Observable<Venta> {
    return this.http.post<Venta>(`${this.api}/ventas`, input);
  }

  update(id: number, input: VentaInput): Observable<Venta> {
    return this.http.put<Venta>(`${this.api}/ventas/${id}`, input);
  }

  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.api}/ventas/${id}`);
  }

  emit(id: number): Observable<Venta> {
    return this.http.post<Venta>(`${this.api}/ventas/${id}/emitir`, {});
  }

  annul(id: number, motivo: string): Observable<Venta> {
    return this.http.post<Venta>(`${this.api}/ventas/${id}/anular`, { motivo });
  }
}
