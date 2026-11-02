import { Component, Input } from '@angular/core';

@Component({
  selector: 'icon-products',
  template: `<svg xmlns="http://www.w3.org/2000/svg" [attr.width]="size" [attr.height]="size" viewBox="0 0 24 24" aria-hidden="true"><g fill="none" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"><rect width="8" height="8" x="2.75" y="2.75" rx="2"/><circle cx="17.25" cy="17.25" r="4"/><path d="M16.5 2.75h3.25a1.5 1.5 0 0 1 1.5 1.5V7.5m-18.5 9v3.25a1.5 1.5 0 0 0 1.5 1.5H7.5"/></g></svg>`,
  styles: `:host { display: inline-flex; align-items: center; justify-content: center; }`,
})
export class ProductsIcon {
  @Input() size = '20';
}
