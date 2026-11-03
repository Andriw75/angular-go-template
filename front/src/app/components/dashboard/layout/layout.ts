import { Component, effect, inject } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { Router, RouterOutlet } from '@angular/router';
import { SidebarComponent } from '../sidebar/sidebar';
import { ConfirmComponent } from '../../../common/confirm/confirm';
import { ToastComponent } from '../../../common/toast/toast';
import { AuthService } from '../../../services/auth.service';
import { MensajesStoreService } from '../../../services/mensajes-store.service';
import { SessionService } from '../../../services/session.service';
import { hasAnySsePermission } from '../../../services/sse-channels';

@Component({
  selector: 'app-dashboard-layout',
  templateUrl: './layout.html',
  styleUrl: './layout.css',
  imports: [RouterOutlet, SidebarComponent, ConfirmComponent, ToastComponent],
})
export class DashboardLayoutComponent {
  constructor() {
    const mensajes = inject(MensajesStoreService);
    const session = inject(SessionService);
    const auth = inject(AuthService);
    const router = inject(Router);

    effect(() => {
      if (hasAnySsePermission(auth.user())) {
        mensajes.init();
      } else {
        mensajes.disconnect();
      }
    });

    session.expired$
      .pipe(takeUntilDestroyed())
      .subscribe(() => {
        mensajes.disconnect();
        auth.user.set(null);
        session.clear();
        router.navigate(['/login'], { queryParams: { expired: '1' } });
      });
  }
}
