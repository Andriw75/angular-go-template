import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { SlowQuery, SlowQueryInput } from '../models/lab';

@Injectable({ providedIn: 'root' })
export class LabService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  start(input: SlowQueryInput): Observable<SlowQuery> {
    return this.http.post<SlowQuery>(`${this.api}/lab/slow-queries`, input);
  }

  runSync(input: SlowQueryInput): Observable<SlowQuery> {
    return this.http.post<SlowQuery>(`${this.api}/lab/run`, input);
  }

  list(): Observable<SlowQuery[]> {
    return this.http.get<SlowQuery[]>(`${this.api}/lab/slow-queries`);
  }

  cancel(id: string): Observable<SlowQuery> {
    return this.http.post<SlowQuery>(`${this.api}/lab/slow-queries/${id}/cancel`, {});
  }
}
