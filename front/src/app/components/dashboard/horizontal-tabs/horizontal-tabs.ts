import {
  Component,
  ElementRef,
  HostListener,
  input,
  output,
  signal,
  viewChild,
} from '@angular/core';
import type { MenuItem } from '../../../models/dashboard';

@Component({
  selector: 'app-horizontal-tabs',
  templateUrl: './horizontal-tabs.html',
  styleUrl: './horizontal-tabs.css',
})
export class HorizontalTabsComponent {
  readonly items = input.required<MenuItem[]>();
  readonly activeKey = input<string | undefined>(undefined);
  readonly select = output<MenuItem>();

  readonly moreOpen = signal(false);
  private readonly tabsContainer = viewChild<ElementRef<HTMLElement>>('tabsContainer');

  scroll(direction: number): void {
    this.tabsContainer()?.nativeElement.scrollBy({
      left: direction * 240,
      behavior: 'smooth',
    });
  }

  choose(item: MenuItem): void {
    this.moreOpen.set(false);
    this.select.emit(item);
  }

  toggleMore(): void {
    this.moreOpen.update((value) => !value);
  }

  @HostListener('document:click', ['$event'])
  onDocumentClick(event: MouseEvent): void {
    const host = (event.target as HTMLElement).closest('app-horizontal-tabs');
    if (!host) this.moreOpen.set(false);
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.moreOpen.set(false);
  }
}
