import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { Bus, BusInput, PaginatedResponse } from '../models/bus';

export interface BusListParams {
  offset?: number;
  limit?: number;
  q?: string;
  tipo?: string;
  activo?: boolean;
}

@Injectable({ providedIn: 'root' })
export class BusesService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  list(params: BusListParams): Observable<PaginatedResponse<Bus>> {
    const p: Record<string, string> = {};
    if (params.offset != null) p['offset'] = String(params.offset);
    if (params.limit != null) p['limit'] = String(params.limit);
    if (params.q) p['q'] = params.q;
    if (params.tipo) p['tipo'] = params.tipo;
    if (params.activo != null) p['activo'] = String(params.activo);

    const qs = new URLSearchParams(p).toString();
    return this.http.get<PaginatedResponse<Bus>>(`${this.api}/buses?${qs}`);
  }

  getByID(id: number): Observable<Bus> {
    return this.http.get<Bus>(`${this.api}/buses/${id}`);
  }

  create(input: BusInput): Observable<Bus> {
    return this.http.post<Bus>(`${this.api}/buses`, input);
  }

  update(id: number, input: BusInput): Observable<Bus> {
    return this.http.put<Bus>(`${this.api}/buses/${id}`, input);
  }

  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.api}/buses/${id}`);
  }
}
