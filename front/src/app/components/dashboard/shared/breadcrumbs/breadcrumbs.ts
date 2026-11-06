import { Component, input } from '@angular/core';
import { RouterLink } from '@angular/router';
import type { MenuItem } from '../../../../models/dashboard';

@Component({
  selector: 'app-breadcrumbs',
  templateUrl: './breadcrumbs.html',
  styleUrl: './breadcrumbs.css',
  imports: [RouterLink],
})
export class BreadcrumbsComponent {
  readonly trail = input<MenuItem[]>([]);
}
