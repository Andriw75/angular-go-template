import { Component, effect, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import {
  INSTALACION_ESTADOS,
  INSTALACION_TIPOS,
  INSTALACION_ZONAS,
  type Instalacion,
  type InstalacionEstado,
  type InstalacionTipo,
  type InstalacionZona,
} from '../../../models/instalacion';

export interface InstalacionModalData {
  mode: 'create' | 'edit';
  lat: number;
  lng: number;
  instalacion?: Instalacion;
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

function randomSerial(): string {
  return `SN-${(10000 + Math.floor(Math.random() * 89999)).toString()}`;
}

@Component({
  selector: 'app-instalacion-modal',
  templateUrl: './instalacion-modal.html',
  styleUrl: './instalacion-modal.css',
  imports: [FormsModule],
})
export class InstalacionModalComponent {
  data = input<InstalacionModalData | null>(null);
  save = output<Instalacion>();
  close = output<void>();

  readonly tipos = INSTALACION_TIPOS;
  readonly estados = INSTALACION_ESTADOS;
  readonly zonas = INSTALACION_ZONAS;

  readonly nombre = signal('');
  readonly tipo = signal<InstalacionTipo>('CAMARA');
  readonly estado = signal<InstalacionEstado>('ACTIVO');
  readonly zona = signal<InstalacionZona>('CENTRO');
  readonly direccion = signal('');
  readonly serial = signal('');
  readonly lat = signal(0);
  readonly lng = signal(0);
  readonly fechaInstalacion = signal(todayISO());

  readonly descripcion = signal('');
  readonly fabricante = signal('');
  readonly modelo = signal('');
  readonly instalador = signal('');
  readonly proveedor = signal('');
  readonly ultimaConexion = signal('');
  readonly firmware = signal('');
  readonly ip = signal('');
  readonly mac = signal('');
  readonly resolucion = signal('');
  readonly contenido = signal('');
  readonly senal = signal<number | null>(null);
  readonly bateria = signal<number | null>(null);
  readonly potencia = signal<number | null>(null);
  readonly contacto = signal('');

  readonly error = signal('');

  constructor() {
    effect(() => {
      const d = this.data();
      if (!d) return;

      this.lat.set(d.lat);
      this.lng.set(d.lng);

      const i = d.instalacion;
      if (d.mode === 'edit' && i) {
        this.nombre.set(i.nombre);
        this.tipo.set(i.tipo);
        this.estado.set(i.estado);
        this.zona.set(i.zona);
        this.direccion.set(i.direccion);
        this.serial.set(i.serial);
        this.fechaInstalacion.set(i.fecha_instalacion);
        this.descripcion.set(i.descripcion ?? '');
        this.fabricante.set(i.fabricante ?? '');
        this.modelo.set(i.modelo ?? '');
        this.instalador.set(i.instalador ?? '');
        this.proveedor.set(i.proveedor ?? '');
        this.ultimaConexion.set(i.ultima_conexion ? i.ultima_conexion.slice(0, 16) : '');
        this.firmware.set(i.firmware ?? '');
        this.ip.set(i.ip ?? '');
        this.mac.set(i.mac ?? '');
        this.resolucion.set(i.resolucion ?? '');
        this.contenido.set(i.contenido ?? '');
        this.senal.set(i.senal ?? null);
        this.bateria.set(i.bateria ?? null);
        this.potencia.set(i.potencia ?? null);
        this.contacto.set(i.contacto ?? '');
      } else {
        this.serial.set(randomSerial());
        this.fechaInstalacion.set(todayISO());
        this.tipo.set('CAMARA');
        this.estado.set('PENDIENTE_INSTALACION');
        this.zona.set('CENTRO');
        this.nombre.set('');
        this.direccion.set('');
        this.descripcion.set('');
        this.fabricante.set('');
        this.modelo.set('');
        this.instalador.set('');
        this.proveedor.set('');
        this.ultimaConexion.set('');
        this.firmware.set('');
        this.ip.set('');
        this.mac.set('');
        this.resolucion.set('');
        this.contenido.set('');
        this.senal.set(null);
        this.bateria.set(null);
        this.potencia.set(null);
        this.contacto.set('');
      }

      this.error.set('');
    });
  }

  submit(): void {
    const name = this.nombre().trim();
    const lat = Number(this.lat());
    const lng = Number(this.lng());

    if (!name) {
      this.error.set('Nombre es obligatorio');
      return;
    }
    if (Number.isNaN(lat) || Number.isNaN(lng)) {
      this.error.set('Latitud y Longitud son obligatorias');
      return;
    }
    if (!this.fechaInstalacion()) {
      this.error.set('Fecha de instalación es obligatoria');
      return;
    }

    const item: Instalacion = {
      id: this.data()?.instalacion?.id ?? 0,
      tipo: this.tipo(),
      nombre: name,
      estado: this.estado(),
      lat,
      lng,
      direccion: this.direccion().trim() || 'Sin dirección',
      zona: this.zona(),
      fecha_instalacion: this.fechaInstalacion(),
      serial: this.serial().trim() || randomSerial(),
    };

    this.setStr(item, 'descripcion', this.descripcion());
    this.setStr(item, 'fabricante', this.fabricante());
    this.setStr(item, 'modelo', this.modelo());
    this.setStr(item, 'instalador', this.instalador());
    this.setStr(item, 'proveedor', this.proveedor());
    this.setStr(item, 'firmware', this.firmware());
    this.setStr(item, 'ip', this.ip());
    this.setStr(item, 'mac', this.mac());
    this.setStr(item, 'resolucion', this.resolucion());
    this.setStr(item, 'contenido', this.contenido());
    this.setStr(item, 'contacto', this.contacto());

    if (this.ultimaConexion()) item.ultima_conexion = new Date(this.ultimaConexion()).toISOString();
    else if (this.data()?.mode === 'edit') item.ultima_conexion = null;

    this.setNum(item, 'senal', this.senal());
    this.setNum(item, 'bateria', this.bateria());
    this.setNum(item, 'potencia', this.potencia());

    this.save.emit(item);
  }

  private setStr<K extends keyof Instalacion>(item: Instalacion, key: K, value: string): void {
    if (value && value.trim() !== '') {
      (item as unknown as Record<string, unknown>)[key as string] = value.trim();
    }
  }

  private setNum(item: Instalacion, key: 'senal' | 'bateria' | 'potencia', value: number | null | ''): void {
    if (value === null || value === '') return;
    const n = Number(value);
    if (!Number.isNaN(n)) {
      (item as unknown as Record<string, unknown>)[key] = n;
    }
  }
}