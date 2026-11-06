import { Component, HostListener, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { LogoutIcon } from '../../../../common/icons/logout.icon';
import { UserSolidIcon } from '../../../../common/icons/user-solid.icon';
import { AuthService } from '../../../../services/auth.service';
import { MensajesStoreService } from '../../../../services/mensajes-store.service';
import { SessionService } from '../../../../services/session.service';

@Component({
  selector: 'app-user-menu',
  templateUrl: './user-menu.html',
  styleUrl: './user-menu.css',
  imports: [UserSolidIcon, LogoutIcon],
})
export class UserMenuComponent {
  protected readonly auth = inject(AuthService);
  private readonly router = inject(Router);
  private readonly mensajesStore = inject(MensajesStoreService);
  private readonly session = inject(SessionService);

  readonly open = signal(false);
  readonly loggingOut = signal(false);

  toggle(): void {
    this.open.update((value) => !value);
  }

  logout(): void {
    this.mensajesStore.disconnect();
    this.session.clear();
    this.loggingOut.set(true);
    this.auth.logout().subscribe({
      next: () => this.auth.user.set(null),
      error: () => this.loggingOut.set(false),
      complete: () => {
        this.loggingOut.set(false);
        this.open.set(false);
        this.router.navigate(['/login']);
      },
    });
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    const host = (event.target as HTMLElement).closest('app-user-menu');
    if (!host) this.open.set(false);
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.open.set(false);
  }
}
