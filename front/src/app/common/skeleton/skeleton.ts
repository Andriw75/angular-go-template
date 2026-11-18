import { Component, computed, input } from '@angular/core';

const CELL_WIDTHS = ['72%', '58%', '84%', '46%', '64%', '52%', '68%', '56%'];

@Component({
  selector: 'app-skeleton',
  templateUrl: './skeleton.html',
  styleUrl: './skeleton.css',
})
export class SkeletonComponent {
  rows = input(5);
  columns = input(6);
  label = input('Cargando...');

  readonly rowItems = computed(() => Array.from({ length: Math.max(1, this.rows()) }));
  readonly columnItems = computed(() => Array.from({ length: Math.max(1, this.columns()) }));

  cellWidth(index: number): string {
    return CELL_WIDTHS[index % CELL_WIDTHS.length];
  }
}
