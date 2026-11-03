export interface UserResponse {
  id: number;
  username: string;
  email: string;
  activo: boolean;
  permisos: string[];
  must_change_password: boolean;
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
