import { Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { FormsModule } from '@angular/forms';
import { AuthService } from '../../services/auth.service';
import { ToastService } from '../../services/toast.service';

@Component({
  selector: 'app-login',
  templateUrl: './login.html',
  styleUrl: './login.css',
  imports: [FormsModule],
})
export class LoginComponent {
  private auth = inject(AuthService);
  private router = inject(Router);
  private toast = inject(ToastService);

  username = signal('');
  password = signal('');
  loading = signal(false);

  async onSubmit(): Promise<void> {
    if (!this.username() || !this.password()) {
      this.toast.error('Ingrese usuario y contraseña');
      return;
    }

    this.loading.set(true);
    try {
      await this.auth.login(this.username(), this.password());
      this.router.navigate(['/dashboard']);
    } catch {
      this.toast.error('Credenciales inválidas');
    } finally {
      this.loading.set(false);
    }
  }
}
