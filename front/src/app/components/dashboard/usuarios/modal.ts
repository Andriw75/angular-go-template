import { Component, Input, Output, EventEmitter, inject, signal, OnInit } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { UsersService, type UserInput } from '../../../services/users.service';
import { ToastService } from '../../../services/toast.service';
import { AuthService } from '../../../services/auth.service';
import type { UserResponse, Permission } from '../../../models/auth';

@Component({
  selector: 'app-user-modal',
  templateUrl: './modal.html',
  styleUrl: './modal.css',
  imports: [FormsModule],
})
export class UserModalComponent implements OnInit {
  private service = inject(UsersService);
  private toast = inject(ToastService);
  protected auth = inject(AuthService);

  @Input() user: UserResponse | null = null;
  @Output() onClose = new EventEmitter<void>();
  @Output() onSaved = new EventEmitter<void>();

  saving = signal(false);
  permisosDisponibles = signal<Permission[]>([]);

  username = '';
  email = '';
  password = '';
  activo = true;
  selectedPermisos: string[] = [];

  get isEditing(): boolean {
    return this.user !== null;
  }

  get isSelf(): boolean {
    return this.user?.id === this.auth.user()?.id;
  }

  ngOnInit(): void {
    this.service.getPermisos().subscribe({
      next: (res) => {
        this.permisosDisponibles.set(res);
      },
    });

    if (this.user) {
      this.username = this.user.username;
      this.email = this.user.email;
      this.activo = this.user.activo;
      this.selectedPermisos = [...this.user.permisos];
    }
  }

  togglePermiso(nombre: string): void {
    if (this.isSelf) return;
    const idx = this.selectedPermisos.indexOf(nombre);
    if (idx >= 0) {
      this.selectedPermisos = this.selectedPermisos.filter((p) => p !== nombre);
    } else {
      this.selectedPermisos = [...this.selectedPermisos, nombre];
    }
  }

  save(): void {
    if (!this.username || !this.email) {
      this.toast.error('Usuario y email son requeridos');
      return;
    }
    if (!this.isEditing && !this.password) {
      this.toast.error('Contraseña es requerida');
      return;
    }

    this.saving.set(true);
    const input: UserInput = {
      username: this.username,
      email: this.email,
      password: this.password,
      activo: this.activo,
      permisos: this.selectedPermisos,
    };

    const obs = this.isEditing
      ? this.service.update(this.user!.id, input)
      : this.service.create(input);

    obs.subscribe({
      next: () => {
        this.saving.set(false);
        this.toast.success(this.isEditing ? 'Usuario actualizado' : 'Usuario creado');
        this.onSaved.emit();
      },
      error: () => {
        this.saving.set(false);
        this.toast.error('Error al guardar usuario');
      },
    });
  }

  cancel(): void {
    this.onClose.emit();
  }
}
