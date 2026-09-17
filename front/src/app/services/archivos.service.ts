import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { Archivo, PaginatedArchivos, Visibilidad } from '../models/archivo';

export interface ArchivoListParams {
  offset?: number;
  limit?: number;
  q?: string;
}

@Injectable({ providedIn: 'root' })
export class ArchivosService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  list(params: ArchivoListParams): Observable<PaginatedArchivos> {
    const p: Record<string, string> = {};
    if (params.offset != null) p['offset'] = String(params.offset);
    if (params.limit != null) p['limit'] = String(params.limit);
    if (params.q) p['q'] = params.q;
    const qs = new URLSearchParams(p).toString();
    return this.http.get<PaginatedArchivos>(`${this.api}/archivos?${qs}`);
  }

  upload(files: File[], visibilidad: Visibilidad): Observable<Archivo[]> {
    const formData = new FormData();
    for (const f of files) {
      formData.append('files', f, f.name);
    }
    formData.append('visibilidad', visibilidad);
    return this.http.post<Archivo[]>(`${this.api}/archivos`, formData);
  }

  updateVisibilidad(id: number, visibilidad: Visibilidad): Observable<Archivo> {
    return this.http.put<Archivo>(`${this.api}/archivos/${id}/visibilidad`, { visibilidad });
  }

  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.api}/archivos/${id}`);
  }

  getURL(archivo: Archivo): string {
    return `${this.api}${archivo.url}`;
  }
}
