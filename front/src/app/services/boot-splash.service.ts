import { Injectable } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class BootSplashService {
  private hidden = false;

  hide(): void {
    if (this.hidden) return;
    this.hidden = true;

    const splash = document.getElementById('boot-splash');
    if (!splash) return;

    splash.classList.add('boot-splash--leaving');
    window.setTimeout(() => splash.remove(), 450);
  }
}
