import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { Categoria, CategoriaInput, PaginatedCategorias } from '../models/categoria';
import type { Imagen } from '../models/imagen';

export interface CategoriaListParams {
  offset?: number;
  limit?: number;
  q?: string;
  activo?: boolean;
}

export interface CategoriaCountParams {
  q?: string;
  activo?: boolean;
}

@Injectable({ providedIn: 'root' })
export class CategoriasService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  count(params: CategoriaCountParams): Observable<number> {
    const p: Record<string, string> = {};
    if (params.q) p['q'] = params.q;
    if (params.activo != null) p['activo'] = String(params.activo);
    const qs = new URLSearchParams(p).toString();
    return this.http.get<number>(`${this.api}/categorias/count?${qs}`);
  }

  list(params: CategoriaListParams): Observable<PaginatedCategorias> {
    const p: Record<string, string> = {};
    if (params.offset != null) p['offset'] = String(params.offset);
    if (params.limit != null) p['limit'] = String(params.limit);
    if (params.q) p['q'] = params.q;
    if (params.activo != null) p['activo'] = String(params.activo);

    const qs = new URLSearchParams(p).toString();
    return this.http.get<PaginatedCategorias>(`${this.api}/categorias?${qs}`);
  }

  getByID(id: number): Observable<Categoria> {
    return this.http.get<Categoria>(`${this.api}/categorias/${id}`);
  }

  create(input: CategoriaInput): Observable<Categoria> {
    return this.http.post<Categoria>(`${this.api}/categorias`, input);
  }

  update(id: number, input: Partial<CategoriaInput>): Observable<Categoria> {
    return this.http.put<Categoria>(`${this.api}/categorias/${id}`, input);
  }

  delete(id: number, cascade = false): Observable<void> {
    return this.http.delete<void>(`${this.api}/categorias/${id}?cascade=${cascade}`);
  }

  uploadImagenes(id: number, files: File[]): Observable<Imagen[]> {
    const formData = new FormData();
    for (const f of files) {
      formData.append('images', f, f.name);
    }
    return this.http.post<Imagen[]>(`${this.api}/categorias/${id}/imagenes`, formData);
  }

  deleteImagen(id: number, imagenId: number): Observable<void> {
    return this.http.delete<void>(`${this.api}/categorias/${id}/imagenes/${imagenId}`);
  }

  reorderImagenes(id: number, ids: number[]): Observable<Imagen[]> {
    return this.http.put<Imagen[]>(`${this.api}/categorias/${id}/imagenes/orden`, { ids });
  }
}
