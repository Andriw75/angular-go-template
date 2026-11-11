import { Component, inject } from '@angular/core';
import { BackdropCloseDirective } from '../modal/backdrop-close.directive';
import { ConfirmService } from '../../services/confirm.service';

@Component({
  selector: 'app-confirm',
  templateUrl: './confirm.html',
  styleUrl: './confirm.css',
  imports: [BackdropCloseDirective],
})
export class ConfirmComponent {
  protected confirm = inject(ConfirmService);
}
