import { Component, Input } from '@angular/core';

@Component({
  selector: 'icon-report',
  template: `<svg xmlns="http://www.w3.org/2000/svg" [attr.width]="size" [attr.height]="size" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 20V10"/><path d="M10 20V4"/><path d="M16 20v-7"/><path d="M22 20H2"/></svg>`,
  styles: `:host { display: inline-flex; align-items: center; justify-content: center; }`,
})
export class ReportIcon {
  @Input() size = '20';
}
