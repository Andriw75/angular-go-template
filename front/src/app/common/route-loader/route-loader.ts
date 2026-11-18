import { Component, inject } from '@angular/core';
import { environment } from '../../../environments/environment';
import { RouteLoadingService } from '../../services/route-loading.service';

@Component({
  selector: 'app-route-loader',
  templateUrl: './route-loader.html',
  styleUrl: './route-loader.css',
})
export class RouteLoaderComponent {
  private readonly loading = inject(RouteLoadingService);

  readonly name = environment.APP_NAME;
  readonly barVisible = this.loading.barVisible;
  readonly overlayVisible = this.loading.overlayVisible;

  constructor() {
    this.loading.init();
  }
}
