import { Injectable, signal } from '@angular/core';
import { environment } from '../../../environments/environment';

export interface ModalConfig {
  closeOnBackdrop: boolean;
}

@Injectable({ providedIn: 'root' })
export class ModalConfigService {
  private readonly closeOnBackdropSignal = signal<boolean>(
    environment.MODAL?.closeOnBackdrop ?? false,
  );

  readonly closeOnBackdrop = this.closeOnBackdropSignal.asReadonly();

  setCloseOnBackdrop(value: boolean): void {
    this.closeOnBackdropSignal.set(value);
  }
}
