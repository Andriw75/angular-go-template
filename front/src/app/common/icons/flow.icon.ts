import { Component, Input } from '@angular/core';

@Component({
  selector: 'icon-flow',
  template: `<svg xmlns="http://www.w3.org/2000/svg" [attr.width]="size" [attr.height]="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="5" cy="6" r="2"/><circle cx="19" cy="6" r="2"/><circle cx="12" cy="18" r="2"/><path d="M7 6h10"/><path d="M6.2 7.7 10.8 16"/><path d="m17.8 7.7-4.6 8.3"/></svg>`,
  styles: `:host { display: inline-flex; align-items: center; justify-content: center; }`,
})
export class FlowIcon {
  @Input() size = '20';
}
