import { Component, Input } from '@angular/core';

@Component({
  selector: 'icon-map',
  template: `<svg xmlns="http://www.w3.org/2000/svg" [attr.width]="size" [attr.height]="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M9 20l-6 2V6l6-2 6 2 6-2v16l-6 2-6-2z"/><path d="M9 4v16"/><path d="M15 6v16"/><path d="M3 6h6"/><path d="M15 10h6"/></svg>`,
  styles: `:host { display: inline-flex; align-items: center; justify-content: center; }`,
})
export class MapIcon {
  @Input() size = '20';
}