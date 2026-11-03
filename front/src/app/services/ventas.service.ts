import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { Venta, VentaInput, PaginatedVentas } from '../models/venta';

@Injectable({ providedIn: 'root' })
export class VentasService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  list(params: { offset?: number; limit?: number; q?: string; serie?: string; estado?: string; desde?: string; hasta?: string }): Observable<PaginatedVentas> {
    const p: Record<string, string> = {};
    if (params.offset != null) p['offset'] = String(params.offset);
    if (params.limit != null) p['limit'] = String(params.limit);
    if (params.q) p['q'] = params.q;
    if (params.serie) p['serie'] = params.serie;
    if (params.estado) p['estado'] = params.estado;
    if (params.desde) p['desde'] = params.desde;
    if (params.hasta) p['hasta'] = params.hasta;
    return this.http.get<PaginatedVentas>(`${this.api}/ventas?${new URLSearchParams(p).toString()}`);
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
