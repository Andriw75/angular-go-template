import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { MensajePendiente, MensajeInput, MensajeUpdateInput, PaginatedMensajes } from '../models/mensaje';

@Injectable({ providedIn: 'root' })
export class MensajesService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  list(): Observable<MensajePendiente[]> {
    return this.http.get<MensajePendiente[]>(`${this.api}/mensajes_pendientes`);
  }

  getByID(id: number): Observable<MensajePendiente> {
    return this.http.get<MensajePendiente>(`${this.api}/mensajes_pendientes/${id}`);
  }

  create(input: MensajeInput): Observable<MensajePendiente> {
    return this.http.post<MensajePendiente>(`${this.api}/mensajes_pendientes`, input);
  }

  update(id: number, input: MensajeUpdateInput): Observable<MensajePendiente> {
    return this.http.put<MensajePendiente>(`${this.api}/mensajes_pendientes/${id}`, input);
  }

  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.api}/mensajes_pendientes/${id}`);
  }

  listPasadas(offset: number, limit: number): Observable<PaginatedMensajes> {
    return this.http.get<PaginatedMensajes>(`${this.api}/mensajes_pendientes/pasadas?offset=${offset}&limit=${limit}`);
  }

  countPasadas(): Observable<number> {
    return this.http.get<number>(`${this.api}/mensajes_pendientes/pasadas/count`);
  }
}
