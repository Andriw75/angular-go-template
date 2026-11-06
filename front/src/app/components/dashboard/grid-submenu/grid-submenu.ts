import {
  Component,
  HostListener,
  computed,
  effect,
  inject,
  input,
  output,
  signal,
} from '@angular/core';
import { Router } from '@angular/router';
import type { MenuItem } from '../../../models/dashboard';

interface SubmenuLevel {
  title: string;
  items: MenuItem[];
}

@Component({
  selector: 'app-grid-submenu',
  templateUrl: './grid-submenu.html',
  styleUrl: './grid-submenu.css',
})
export class GridSubmenuComponent {
  private readonly router = inject(Router);

  readonly item = input<MenuItem | null>(null);
  readonly close = output<void>();

  readonly levels = signal<SubmenuLevel[]>([]);
  readonly current = computed<SubmenuLevel>(
    () => this.levels().at(-1) ?? { title: '', items: [] },
  );

  constructor() {
    effect(() => {
      const root = this.item();
      if (!root?.submenu?.length) {
        this.levels.set([]);
        return;
      }
      this.levels.set([{ title: root.label, items: root.submenu }]);
    });
  }

  select(item: MenuItem): void {
    if (item.submenu?.length) {
      this.levels.update((levels) => [
        ...levels,
        { title: item.label, items: item.submenu ?? [] },
      ]);
      return;
    }
    if (item.path) {
      void this.router.navigateByUrl(item.path);
    }
    this.dismiss();
  }

  back(): void {
    this.levels.update((levels) => levels.slice(0, -1));
  }

  dismiss(): void {
    this.close.emit();
  }

  @HostListener('document:keydown.escape')
  onEscape(): void {
    this.dismiss();
  }
}
