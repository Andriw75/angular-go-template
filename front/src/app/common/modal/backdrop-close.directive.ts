import { Directive, EventEmitter, HostListener, Output, inject } from '@angular/core';
import { ModalConfigService } from './modal-config.service';

// Centraliza el cierre al hacer clic fuera del diálogo. Por defecto no cierra;
// se habilita globalmente con ModalConfigService o environment.MODAL.
@Directive({
  selector: '[appBackdropClose]',
  standalone: true,
})
export class BackdropCloseDirective {
  private readonly config = inject(ModalConfigService);

  @Output() readonly appBackdropClose = new EventEmitter<void>();

  @HostListener('click', ['$event'])
  onClick(event: MouseEvent): void {
    if (event.target !== event.currentTarget) {
      return;
    }
    if (!this.config.closeOnBackdrop()) {
      return;
    }
    this.appBackdropClose.emit();
  }
}
