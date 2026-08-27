import { Component, ElementRef, QueryList, ViewChildren, effect, inject, output } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { InstalacionesStoreService } from '../../../services/instalaciones-store.service';
import { MapIcon } from '../../../common/icons/map.icon';
import {
  INSTALACION_ESTADOS,
  INSTALACION_TIPOS,
  INSTALACION_ZONAS,
  ESTADO_CONFIG,
  type Instalacion,
} from '../../../models/instalacion';

const OPTIONAL_LABELS: Array<[keyof Instalacion, string]> = [
  ['descripcion', 'Descripción'],
  ['fabricante', 'Fabricante'],
  ['modelo', 'Modelo'],
  ['instalador', 'Instalador'],
  ['proveedor', 'Proveedor'],
  ['ultima_conexion', 'Última conexión'],
  ['firmware', 'Firmware'],
  ['ip', 'IP'],
  ['mac', 'MAC'],
  ['resolucion', 'Resolución'],
  ['contenido', 'Contenido'],
  ['senal', 'Señal (%)'],
  ['bateria', 'Batería (%)'],
  ['potencia', 'Potencia (W)'],
  ['contacto', 'Contacto'],
];

@Component({
  selector: 'app-instalaciones-lista',
  imports: [FormsModule, MapIcon],
  templateUrl: './lista.html',
  styleUrl: './lista.css',
})
export class ListaInstalacionesComponent {
  readonly store = inject(InstalacionesStoreService);

  readonly tipos = INSTALACION_TIPOS;
  readonly estados = INSTALACION_ESTADOS;
  readonly zonas = INSTALACION_ZONAS;

  edit = output<Instalacion>();
  remove = output<Instalacion>();

  @ViewChildren('cardRef', { read: ElementRef }) cards!: QueryList<ElementRef<HTMLElement>>;

  constructor() {
    effect(() => {
      const id = this.store.selectedId();
      if (id == null) return;
      const idx = this.store.filtered().findIndex((x) => x.id === id);
      if (idx < 0 || idx >= this.cards.length) return;
      this.cards.get(idx)?.nativeElement.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    });
  }

  estadoColor(estado: Instalacion['estado']): string {
    return ESTADO_CONFIG[estado].color;
  }

  estadoBg(estado: Instalacion['estado']): string {
    return ESTADO_CONFIG[estado].background;
  }

  estadoLabel(estado: Instalacion['estado']): string {
    return ESTADO_CONFIG[estado].label;
  }

  trackById(_: number, item: Instalacion): number {
    return item.id;
  }

  optionals(item: Instalacion): Array<{ label: string; value: string }> {
    const out: Array<{ label: string; value: string }> = [];
    for (const [key, label] of OPTIONAL_LABELS) {
      const v = item[key];
      if (v !== undefined && v !== null && v !== '') out.push({ label, value: String(v) });
    }
    return out;
  }
}