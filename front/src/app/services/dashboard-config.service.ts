import { Injectable, signal } from '@angular/core';
import { environment } from '../../environments/environment';
import type { DashboardConfig, DashboardVariant } from '../models/dashboard';

@Injectable({ providedIn: 'root' })
export class DashboardConfigService {
  readonly config = environment.DASHBOARD;
  readonly variant = signal<DashboardVariant>(environment.DASHBOARD.VARIANT);

  get horizontal(): DashboardConfig['HORIZONTAL'] {
    return this.config.HORIZONTAL;
  }

  get grid(): DashboardConfig['GRID'] {
    return this.config.GRID;
  }
}
