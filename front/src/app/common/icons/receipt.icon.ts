import { Component, Input } from '@angular/core';

@Component({
  selector: 'icon-receipt',
  template: `<svg xmlns="http://www.w3.org/2000/svg" [attr.width]="size" [attr.height]="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 3v18l2-1 2 1 2-1 2 1 2-1 2 1V3l-2 1-2-1-2 1-2-1-2 1z"/><path d="M8 8h8"/><path d="M8 12h8"/><path d="M8 16h5"/></svg>`,
  styles: `:host { display: inline-flex; align-items: center; justify-content: center; }`,
})
export class ReceiptIcon {
  @Input() size = '20';
}
