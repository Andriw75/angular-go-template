import { Component, input, output, signal } from '@angular/core';

@Component({
  selector: 'app-pagination',
  templateUrl: './pagination.html',
  styleUrl: './pagination.css',
})
export class PaginationComponent {
  page = input.required<number>();
  totalPages = input.required<number>();
  total = input<number>(0);
  pageChange = output<number>();

  inputValue = signal('');

  get from(): number {
    const p = this.page();
    const limit = 10;
    return (p - 1) * limit + 1;
  }

  get to(): number {
    const p = this.page();
    const limit = 10;
    const total = this.total();
    return Math.min(p * limit, total);
  }

  onInputKeydown(e: Event): void {
    const k = e as KeyboardEvent;
    if (k.key !== 'Enter') return;
    const val = parseInt(this.inputValue(), 10);
    if (!isNaN(val) && val >= 1 && val <= this.totalPages() && val !== this.page()) {
      this.pageChange.emit(val);
    }
    this.inputValue.set('');
  }

  onInputBlur(): void {
    this.inputValue.set('');
  }

  goTo(p: number): void {
    if (p >= 1 && p <= this.totalPages() && p !== this.page()) {
      this.pageChange.emit(p);
    }
  }

  jump(offset: number): void {
    this.goTo(this.page() + offset);
  }
}
