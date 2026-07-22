export interface UserResponse {
  id: number;
  username: string;
  email: string;
  activo: boolean;
  permisos: string[];
}

export interface Permission {
  id: number;
  nombre: string;
  descripcion: string;
}

export interface LoginInput {
  username: string;
  password: string;
}
