export interface UserResponse {
  id: number;
  username: string;
  email: string;
  permisos: string[];
}

export interface LoginInput {
  username: string;
  password: string;
}
