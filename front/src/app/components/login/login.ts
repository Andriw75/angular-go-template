import { Component, OnInit, inject, signal } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { AuthService } from '../../services/auth.service';
import { SpinnerIcon } from '../../common/icons/spinner.icon';

@Component({
  selector: 'app-login',
  templateUrl: './login.html',
  styleUrl: './login.css',
  imports: [SpinnerIcon],
})
export class LoginComponent implements OnInit {
  private auth = inject(AuthService);
  private router = inject(Router);
  private route = inject(ActivatedRoute);

  username = signal('');
  password = signal('');
  error = signal('');
  isLoading = signal(false);

  ngOnInit(): void {
    if (this.route.snapshot.queryParamMap.get('expired') === '1') {
      this.error.set('La sesión ha caducado. Vuelve a iniciar sesión.');
    }
  }

  handleSubmit(e: Event): void {
    e.preventDefault();

    const user = this.username().trim();
    const pass = this.password().trim();

    if (!user || !pass) {
      const missing: string[] = [];
      if (!user) missing.push('Usuario');
      if (!pass) missing.push('Contraseña');
      this.error.set(`Complete los campos: ${missing.join(', ')}`);
      return;
    }

    this.isLoading.set(true);
    this.error.set('');

    this.auth.login(user, pass).subscribe({
      next: (u) => {
        this.auth.user.set(u);
        this.router.navigate(['/dashboard']);
      },
      error: () => {
        this.isLoading.set(false);
        this.error.set('Usuario o contraseña incorrectos');
      },
    });
  }
}
