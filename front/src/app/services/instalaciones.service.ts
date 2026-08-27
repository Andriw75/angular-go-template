import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { Instalacion } from '../models/instalacion';

export type InstalacionCreateInput = Omit<Instalacion, 'id'>;
export type InstalacionUpdateInput = Partial<Omit<Instalacion, 'id'>>;

@Injectable({ providedIn: 'root' })
export class InstalacionesService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  // Una sola llamada trae todo el listado; el filtrado ocurre en el cliente (store).
  list(): Observable<Instalacion[]> {
    return this.http.get<Instalacion[]>(`${this.api}/instalaciones`);
  }

  create(input: InstalacionCreateInput): Observable<Instalacion> {
    return this.http.post<Instalacion>(`${this.api}/instalaciones`, input);
  }

  // Solo envía los campos que cambian.
  update(id: number, input: InstalacionUpdateInput): Observable<Instalacion> {
    return this.http.put<Instalacion>(`${this.api}/instalaciones/${id}`, input);
  }

  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.api}/instalaciones/${id}`);
  }
}