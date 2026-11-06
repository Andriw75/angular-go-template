import { Component, input } from '@angular/core';
import { NavFlyoutComponent } from '../nav-flyout/nav-flyout';
import type { MenuItem } from '../../../models/dashboard';

@Component({
  selector: 'app-horizontal-subnav',
  templateUrl: './horizontal-subnav.html',
  styleUrl: './horizontal-subnav.css',
  imports: [NavFlyoutComponent],
})
export class HorizontalSubnavComponent {
  readonly items = input.required<MenuItem[]>();
  readonly activeUrl = input('');
}
