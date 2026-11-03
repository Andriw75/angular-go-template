import { Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { AuthService } from '../../services/auth.service';
import { SpinnerIcon } from '../../common/icons/spinner.icon';

@Component({
  selector: 'app-change-password',
  templateUrl: './change-password.html',
  styleUrl: './change-password.css',
  imports: [SpinnerIcon],
})
export class ChangePasswordComponent {
  private auth = inject(AuthService);
  private router = inject(Router);

  currentPassword = signal('');
  newPassword = signal('');
  confirmPassword = signal('');
  error = signal('');
  isLoading = signal(false);

  handleSubmit(e: Event): void {
    e.preventDefault();

    if (!this.currentPassword() || !this.newPassword()) {
      this.error.set('Completa todos los campos.');
      return;
    }
    if (this.newPassword().length < 12) {
      this.error.set('La nueva contraseña debe tener al menos 12 caracteres.');
      return;
    }
    if (this.newPassword() !== this.confirmPassword()) {
      this.error.set('Las contraseñas nuevas no coinciden.');
      return;
    }
    if (this.newPassword() === this.currentPassword()) {
      this.error.set('La nueva contraseña debe ser diferente a la actual.');
      return;
    }

    this.isLoading.set(true);
    this.error.set('');

    this.auth.changePassword(this.currentPassword(), this.newPassword()).subscribe({
      next: () => {
        this.auth.user.set(null);
        this.router.navigate(['/login'], { queryParams: { changed: '1' } });
      },
      error: (err) => {
        this.isLoading.set(false);
        this.error.set(err?.error?.error ?? 'No se pudo cambiar la contraseña.');
      },
    });
  }
}
