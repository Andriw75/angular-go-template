import { NgComponentOutlet } from '@angular/common';
import { Component, input, output } from '@angular/core';
import type { MenuItem } from '../../../../models/dashboard';

@Component({
  selector: 'app-module-card',
  templateUrl: './module-card.html',
  styleUrl: './module-card.css',
  imports: [NgComponentOutlet],
})
export class ModuleCardComponent {
  readonly item = input.required<MenuItem>();
  readonly select = output<MenuItem>();
}
