import { HttpClient, HttpParams } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { ReporteBuses, ReporteVentas, Documento, DocumentoResult } from '../models/reporte';

export interface ReporteFilters {
  q?: string;
  tipo?: string;
  activo?: boolean;
  desde?: string;
  hasta?: string;
  serie?: string;
  estado?: string;
  modo?: 'actual' | 'generado';
  cantidad?: number;
}

export interface DocumentoList {
  data: Documento[];
  total: number;
  offset: number;
  limit: number;
}

@Injectable({ providedIn: 'root' })
export class ReportesService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  private params(filters: ReporteFilters): HttpParams {
    let p = new HttpParams();
    for (const [k, v] of Object.entries(filters)) {
      if (v !== undefined && v !== null && v !== '') p = p.set(k, String(v));
    }
    return p;
  }

  buses(filters: ReporteFilters): Observable<ReporteBuses> {
    return this.http.get<ReporteBuses>(`${this.api}/reportes/buses`, { params: this.params(filters) });
  }

  busesCount(filters: ReporteFilters): Observable<{ total: number }> {
    return this.http.get<{ total: number }>(`${this.api}/reportes/buses/count`, { params: this.params(filters) });
  }

  ventas(filters: ReporteFilters): Observable<ReporteVentas> {
    return this.http.get<ReporteVentas>(`${this.api}/reportes/ventas`, { params: this.params(filters) });
  }

  ventasCount(filters: ReporteFilters): Observable<{ total: number }> {
    return this.http.get<{ total: number }>(`${this.api}/reportes/ventas/count`, { params: this.params(filters) });
  }

  downloadBusesPdf(filters: ReporteFilters, template: string): Observable<Blob> {
    return this.http.get(`${this.api}/reportes/buses/pdf`, {
      params: this.params(filters).set('template', template),
      responseType: 'blob',
    });
  }

  downloadBusesXlsx(filters: ReporteFilters): Observable<Blob> {
    return this.http.get(`${this.api}/reportes/buses/xlsx`, { params: this.params(filters), responseType: 'blob' });
  }

  downloadVentaPdf(id: number, template: string): Observable<Blob> {
    return this.http.get(`${this.api}/reportes/ventas/${id}/pdf`, {
      params: new HttpParams().set('template', template),
      responseType: 'blob',
    });
  }

  downloadVentaXlsx(id: number): Observable<Blob> {
    return this.http.get(`${this.api}/reportes/ventas/${id}/xlsx`, { responseType: 'blob' });
  }

  generateBusesDocument(body: Record<string, unknown>): Observable<DocumentoResult> {
    return this.http.post<DocumentoResult>(`${this.api}/documentos/buses`, body);
  }

  generateVentaDocument(id: number, body: Record<string, unknown>): Observable<DocumentoResult> {
    return this.http.post<DocumentoResult>(`${this.api}/documentos/ventas/${id}`, body);
  }

  job(id: string): Observable<DocumentoResult> {
    return this.http.get<DocumentoResult>(`${this.api}/documentos/jobs/${id}`);
  }

  documents(params: { offset?: number; limit?: number; q?: string; tipo?: string; serie?: string; estado?: string }): Observable<DocumentoList> {
    let p = new HttpParams();
    for (const [k, v] of Object.entries(params)) {
      if (v !== undefined && v !== null && v !== '') p = p.set(k, String(v));
    }
    return this.http.get<DocumentoList>(`${this.api}/documentos`, { params: p });
  }

  annulDocument(id: number, motivo: string): Observable<Documento> {
    return this.http.post<Documento>(`${this.api}/documentos/${id}/anular`, { motivo });
  }

  documentFile(id: number, formato: 'pdf' | 'xlsx'): Observable<Blob> {
    return this.http.get(`${this.api}/documentos/${id}/file`, {
      params: new HttpParams().set('formato', formato),
      responseType: 'blob',
    });
  }
}
