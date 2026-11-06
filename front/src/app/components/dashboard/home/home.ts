import { Component, computed, inject } from '@angular/core';
import { AuthService } from '../../../services/auth.service';
import { DashboardConfigService } from '../../../services/dashboard-config.service';
import { GridHomeComponent } from '../grid-home/grid-home';

@Component({
  selector: 'app-home',
  templateUrl: './home.html',
  styleUrl: './home.css',
  imports: [GridHomeComponent],
})
export class HomeComponent {
  protected readonly auth = inject(AuthService);
  private readonly config = inject(DashboardConfigService);

  readonly isGrid = computed(() => this.config.variant() === 'grid');
}
