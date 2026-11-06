import { Component, inject, input } from '@angular/core';
import { SessionService } from '../../../../services/session.service';

@Component({
  selector: 'app-session-badge',
  templateUrl: './session-badge.html',
  styleUrl: './session-badge.css',
})
export class SessionBadgeComponent {
  protected readonly session = inject(SessionService);
  readonly compact = input(false);
}
