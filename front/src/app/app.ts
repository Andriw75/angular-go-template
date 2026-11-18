import { Component, inject } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { NavigationEnd, Router, RouterOutlet } from '@angular/router';
import { filter, take } from 'rxjs';
import { RouteLoaderComponent } from './common/route-loader/route-loader';
import { BootSplashService } from './services/boot-splash.service';

@Component({
  selector: 'app-root',
  templateUrl: './app.html',
  styleUrl: './app.css',
  imports: [RouterOutlet, RouteLoaderComponent],
})
export class App {
  private readonly router = inject(Router);
  private readonly splash = inject(BootSplashService);

  constructor() {
    this.router.events
      .pipe(
        filter((event): event is NavigationEnd => event instanceof NavigationEnd),
        take(1),
        takeUntilDestroyed(),
      )
      .subscribe(() => this.splash.hide());

    window.setTimeout(() => this.splash.hide(), 8000);
  }
}
