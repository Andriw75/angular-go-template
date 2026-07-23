import { Injectable } from '@angular/core';
import { Observable, Subject } from 'rxjs';
import { environment } from '../../environments/environment';

export interface SSEEvent<T = any> {
  type: string;
  data: T;
}

@Injectable({ providedIn: 'root' })
export class SSEService {
  private eventSource: EventSource | null = null;

  connect<T = any>(path: string): Observable<SSEEvent<T>> {
    const subj = new Subject<SSEEvent<T>>();

    this.disconnect();

    this.eventSource = new EventSource(`${environment.API_URL}${path}`, { withCredentials: true });

    this.eventSource.onmessage = (event) => {
      try {
        const parsed: SSEEvent<T> = JSON.parse(event.data);
        subj.next(parsed);
      } catch {
        // ignore parse errors
      }
    };

    this.eventSource.onerror = () => {
      subj.error(new Error('SSE connection error'));
      this.disconnect();
    };

    return subj.asObservable();
  }

  disconnect(): void {
    if (this.eventSource) {
      this.eventSource.close();
      this.eventSource = null;
    }
  }
}
