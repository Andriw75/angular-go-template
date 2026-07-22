import { Component, inject } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';
import { AuthService } from '../../../services/auth.service';
import { MENU_ITEMS, type MenuItem } from './menu.config';

@Component({
  selector: 'app-sidebar',
  templateUrl: './sidebar.html',
  styleUrl: './sidebar.css',
  imports: [RouterLink, RouterLinkActive],
})
export class SidebarComponent {
  protected auth = inject(AuthService);

  getMenuItems(): MenuItem[] {
    return MENU_ITEMS.filter((item) => {
      if (!item.permission) return true;
      return this.auth.hasPermission(item.permission);
    });
  }

  onLogout(): void {
    this.auth.logout();
  }
}
