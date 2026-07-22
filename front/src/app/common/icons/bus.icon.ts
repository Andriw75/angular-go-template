import { Component, Input } from '@angular/core';

@Component({
  selector: 'icon-bus',
  template: `<svg xmlns="http://www.w3.org/2000/svg" [attr.width]="size" [attr.height]="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="14" rx="2"/><path d="M3 7h18"/><path d="M7 17v2"/><path d="M17 17v2"/><path d="M3 13h18"/><path d="M6 12v2"/><path d="M18 12v2"/><path d="M3 3h18"/></svg>`,
  styles: `:host { display: inline-flex; align-items: center; justify-content: center; }`,
})
export class BusIcon {
  @Input() size = '20';
}
