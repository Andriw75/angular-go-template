import { Component, Input } from '@angular/core';

@Component({
  selector: 'icon-categories',
  template: `<svg xmlns="http://www.w3.org/2000/svg" [attr.width]="size" [attr.height]="size" viewBox="0 0 24 24" aria-hidden="true"><path fill="currentColor" d="M3 3h4v4H3zm7 0h4v4h-4zm7 0h4v4h-4zM3 17h4v4H3zm7 0h4v4h-4zm7 0h4v4h-4zM3 10h4v4H3zm7 0h4v4h-4zm7 0h4v4h-4z"/></svg>`,
  styles: `:host { display: inline-flex; align-items: center; justify-content: center; }`,
})
export class CategoriesIcon {
  @Input() size = '20';
}
