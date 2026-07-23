import { Component, inject, signal } from '@angular/core';
import { UsersService } from '../../../services/users.service';
import { ToastService } from '../../../services/toast.service';
import { ConfirmService } from '../../../services/confirm.service';
import { AuthService } from '../../../services/auth.service';
import { UserModalComponent } from './modal';
import type { UserResponse } from '../../../models/auth';

@Component({
  selector: 'app-users-list',
  templateUrl: './list.html',
  styleUrl: './list.css',
  imports: [UserModalComponent],
})
export class UsersListComponent {
  private service = inject(UsersService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmService);
  protected auth = inject(AuthService);

  users = signal<UserResponse[]>([]);
  loading = signal(false);
  showModal = signal(false);
  selectedUser = signal<UserResponse | null>(null);

  constructor() {
    this.load();
  }

  load(): void {
    this.loading.set(true);
    this.service.list().subscribe({
      next: (res) => {
        this.users.set(res);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.toast.error('Error al cargar usuarios');
      },
    });
  }

  openCreate(): void {
    this.selectedUser.set(null);
    this.showModal.set(true);
  }

  openEdit(user: UserResponse): void {
    this.selectedUser.set(user);
    this.showModal.set(true);
  }

  onModalSaved(): void {
    this.showModal.set(false);
    this.load();
  }

  deleting = signal<number | null>(null);

  async confirmDelete(user: UserResponse): Promise<void> {
    if (user.id === this.auth.user()?.id) {
      this.toast.error('No puedes eliminarte a ti mismo');
      return;
    }
    const ok = await this.confirm.confirm(
      'Eliminar Usuario',
      `¿Seguro de eliminar al usuario "${user.username}"?`,
      'danger',
    );
    if (!ok) return;
    this.deleting.set(user.id);

    this.service.delete(user.id).subscribe({
      next: () => {
        this.toast.success('Usuario eliminado');
        this.deleting.set(null);
        this.load();
      },
      error: () => {
        this.toast.error('Error al eliminar usuario');
        this.deleting.set(null);
      },
    });
  }
}
