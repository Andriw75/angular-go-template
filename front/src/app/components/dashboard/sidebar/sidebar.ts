import { Component, inject, signal } from '@angular/core';
import { Router } from '@angular/router';
import { AuthService } from '../../../services/auth.service';
import { MenuService } from '../../../services/menu.service';
import { MensajesStoreService } from '../../../services/mensajes-store.service';
import { SessionService } from '../../../services/session.service';
import { SidebarMenuItemComponent } from './sidebar-menu-item';
import { UserSolidIcon } from '../../../common/icons/user-solid.icon';
import { LogoutIcon } from '../../../common/icons/logout.icon';
import { SpinnerIcon } from '../../../common/icons/spinner.icon';

@Component({
  selector: 'app-sidebar',
  templateUrl: './sidebar.html',
  styleUrl: './sidebar.css',
  imports: [SidebarMenuItemComponent, UserSolidIcon, LogoutIcon, SpinnerIcon],
})
export class SidebarComponent {
  private auth = inject(AuthService);
  private router = inject(Router);
  private mensajesStore = inject(MensajesStoreService);
  private menuService = inject(MenuService);
  protected session = inject(SessionService);

  user = this.auth.user;
  menu = this.menuService.visibleMenu;

  isCollapsed = signal(false);
  isLoggingOut = signal(false);

  toggleSidebar(): void {
    this.isCollapsed.update((v) => !v);
  }

  logout(): void {
    this.mensajesStore.disconnect();
    this.session.clear();
    this.isLoggingOut.set(true);
    this.auth.logout().subscribe({
      next: () => {
        this.auth.user.set(null);
      },
      error: () => {
        this.isLoggingOut.set(false);
      },
      complete: () => {
        this.isLoggingOut.set(false);
        this.router.navigate(['/login']);
      },
    });
  }

}
