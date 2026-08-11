import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { Producto, ProductoInput, PaginatedProductos } from '../models/producto';

export interface ProductoListParams {
  offset?: number;
  limit?: number;
  q?: string;
  categoria_id?: number;
  activo?: boolean;
}

export interface ProductoCountParams {
  q?: string;
  categoria_id?: number;
  activo?: boolean;
}

@Injectable({ providedIn: 'root' })
export class ProductosService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  count(params: ProductoCountParams): Observable<number> {
    const p: Record<string, string> = {};
    if (params.q) p['q'] = params.q;
    if (params.categoria_id != null) p['categoria_id'] = String(params.categoria_id);
    if (params.activo != null) p['activo'] = String(params.activo);
    const qs = new URLSearchParams(p).toString();
    return this.http.get<number>(`${this.api}/productos/count?${qs}`);
  }

  list(params: ProductoListParams): Observable<PaginatedProductos> {
    const p: Record<string, string> = {};
    if (params.offset != null) p['offset'] = String(params.offset);
    if (params.limit != null) p['limit'] = String(params.limit);
    if (params.q) p['q'] = params.q;
    if (params.categoria_id != null) p['categoria_id'] = String(params.categoria_id);
    if (params.activo != null) p['activo'] = String(params.activo);

    const qs = new URLSearchParams(p).toString();
    return this.http.get<PaginatedProductos>(`${this.api}/productos?${qs}`);
  }

  getByID(id: number): Observable<Producto> {
    return this.http.get<Producto>(`${this.api}/productos/${id}`);
  }

  create(input: ProductoInput): Observable<Producto> {
    return this.http.post<Producto>(`${this.api}/productos`, input);
  }

  update(id: number, input: ProductoInput): Observable<Producto> {
    return this.http.put<Producto>(`${this.api}/productos/${id}`, input);
  }

  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.api}/productos/${id}`);
  }
}
