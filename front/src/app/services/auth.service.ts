import { HttpClient } from '@angular/common/http';
import { Injectable, inject, signal } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { UserResponse } from '../models/auth';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  user = signal<UserResponse | null>(null);

  me(): Observable<UserResponse> {
    return this.http.get<UserResponse>(`${this.api}/auth/me`);
  }

  login(username: string, password: string): Observable<UserResponse> {
    const body = new URLSearchParams();
    body.set('username', username);
    body.set('password', password);
    return this.http.post<UserResponse>(`${this.api}/auth/token`, body.toString(), {
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    });
  }

  changePassword(currentPassword: string, newPassword: string): Observable<{ message: string }> {
    return this.http.post<{ message: string }>(`${this.api}/auth/change-password`, {
      current_password: currentPassword,
      new_password: newPassword,
    });
  }

  logout(): Observable<void> {
    return this.http.post<void>(`${this.api}/auth/logout`, {});
  }

  hasPermission(permiso: string): boolean {
    return this.user()?.permisos?.includes(permiso) ?? false;
  }
}
