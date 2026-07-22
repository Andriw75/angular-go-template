import { HttpClient } from '@angular/common/http';
import { Injectable, inject, signal } from '@angular/core';
import { firstValueFrom } from 'rxjs';
import { environment } from '../../environments/environment';
import type { UserResponse } from '../models/auth';

@Injectable({ providedIn: 'root' })
export class AuthService {
  private http = inject(HttpClient);
  user = signal<UserResponse | null>(null);

  async login(username: string, password: string): Promise<UserResponse> {
    const form = new FormData();
    form.set('username', username);
    form.set('password', password);

    const user = await firstValueFrom(
      this.http.post<UserResponse>(`${environment.API_URL}/auth/token`, form),
    );
    this.user.set(user);
    return user;
  }

  async me(): Promise<UserResponse> {
    const user = await firstValueFrom(
      this.http.get<UserResponse>(`${environment.API_URL}/auth/me`),
    );
    this.user.set(user);
    return user;
  }

  async logout(): Promise<void> {
    await firstValueFrom(
      this.http.post(`${environment.API_URL}/auth/logout`, {}),
    );
    this.clear();
  }

  clear(): void {
    this.user.set(null);
  }

  hasPermission(permiso: string): boolean {
    return this.user()?.permisos?.includes(permiso) ?? false;
  }
}
