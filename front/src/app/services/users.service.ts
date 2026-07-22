import { HttpClient } from '@angular/common/http';
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';
import type { UserResponse, Permission } from '../models/auth';

export interface UserInput {
  username: string;
  email: string;
  password?: string;
  activo: boolean;
  permisos: string[];
}

@Injectable({ providedIn: 'root' })
export class UsersService {
  private readonly api = environment.API_URL;
  private http = inject(HttpClient);

  list(): Observable<UserResponse[]> {
    return this.http.get<UserResponse[]>(`${this.api}/users`);
  }

  getByID(id: number): Observable<UserResponse> {
    return this.http.get<UserResponse>(`${this.api}/users/${id}`);
  }

  create(input: UserInput): Observable<UserResponse> {
    return this.http.post<UserResponse>(`${this.api}/users`, input);
  }

  update(id: number, input: UserInput): Observable<UserResponse> {
    return this.http.put<UserResponse>(`${this.api}/users/${id}`, input);
  }

  delete(id: number): Observable<void> {
    return this.http.delete<void>(`${this.api}/users/${id}`);
  }

  getPermisos(): Observable<Permission[]> {
    return this.http.get<Permission[]>(`${this.api}/permisos`);
  }
}
