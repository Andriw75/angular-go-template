import { Component, effect, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { BusesService } from '../../../services/buses.service';
import { ToastService } from '../../../services/toast.service';
import type { Bus, BusInput } from '../../../models/bus';

@Component({
  selector: 'app-bus-modal',
  templateUrl: './modal.html',
  styleUrl: './modal.css',
  imports: [FormsModule],
})
export class BusModalComponent {
  private service = inject(BusesService);
  private toast = inject(ToastService);

  bus = input<Bus | null>(null);
  onClose = output<void>();
  onSaved = output<void>();

  isEditing = signal(false);
  saving = signal(false);
  error = signal('');

  form: BusInput = {
    placa: '',
    nombre: '',
    marca: '',
    modelo: '',
    anio: new Date().getFullYear(),
    capacidad: 20,
    tipo: 'BUS',
    activo: true,
    fecha_compra: '',
    ultimo_mantenimiento: null,
    precio: 0,
    peso: 0,
    color: '',
    descripcion: '',
  };

  tipos = ['BUS', 'VAN', 'MINIBUS', 'MICROBUS'];

  constructor() {
    effect(() => {
      const b = this.bus();
      if (b) {
        this.isEditing.set(true);
        this.form = {
          placa: b.placa,
          nombre: b.nombre,
          marca: b.marca,
          modelo: b.modelo,
          anio: b.anio,
          capacidad: b.capacidad,
          tipo: b.tipo,
          activo: b.activo,
          fecha_compra: b.fecha_compra,
          ultimo_mantenimiento: b.ultimo_mantenimiento,
          precio: b.precio,
          peso: b.peso,
          color: b.color,
          descripcion: b.descripcion,
        };
      } else {
        this.isEditing.set(false);
        this.form = {
          placa: '', nombre: '', marca: '', modelo: '',
          anio: new Date().getFullYear(), capacidad: 20, tipo: 'BUS',
          activo: true, fecha_compra: '', ultimo_mantenimiento: null,
          precio: 0, peso: 0, color: '', descripcion: '',
        };
      }
      this.error.set('');
    });
  }

  close(): void {
    this.onClose.emit();
  }

  onSubmit(): void {
    if (!this.form.placa || !this.form.nombre) {
      this.error.set('Placa y Nombre son obligatorios');
      return;
    }
    if (!this.form.fecha_compra) {
      this.error.set('Fecha de compra es obligatoria');
      return;
    }

    this.saving.set(true);
    this.error.set('');

    const obs = this.isEditing()
      ? this.service.update(this.bus()!.id, this.buildChanges(this.bus()!))
      : this.service.create(this.form);

    obs.subscribe({
      next: () => {
        this.toast.success(this.isEditing() ? 'Bus actualizado' : 'Bus creado');
        this.saving.set(false);
        this.onSaved.emit();
        this.close();
      },
      error: () => {
        this.saving.set(false);
        this.error.set('Error al guardar el bus');
      },
    });
  }

  private buildChanges(original: Bus): Partial<BusInput> {
    const changes: Record<string, unknown> = {};
    const f = this.form;

    if (f.placa !== original.placa) changes['placa'] = f.placa;
    if (f.nombre !== original.nombre) changes['nombre'] = f.nombre;
    if (f.marca !== original.marca) changes['marca'] = f.marca;
    if (f.modelo !== original.modelo) changes['modelo'] = f.modelo;
    if (f.anio !== original.anio) changes['anio'] = f.anio;
    if (f.capacidad !== original.capacidad) changes['capacidad'] = f.capacidad;
    if (f.tipo !== original.tipo) changes['tipo'] = f.tipo;
    if (f.activo !== original.activo) changes['activo'] = f.activo;
    if (f.fecha_compra !== original.fecha_compra) changes['fecha_compra'] = f.fecha_compra;
    if (f.ultimo_mantenimiento !== original.ultimo_mantenimiento) {
      changes['ultimo_mantenimiento'] = f.ultimo_mantenimiento;
    }
    if (f.precio !== original.precio) changes['precio'] = f.precio;
    if (f.peso !== original.peso) changes['peso'] = f.peso;
    if (f.color !== original.color) changes['color'] = f.color;
    if (f.descripcion !== original.descripcion) changes['descripcion'] = f.descripcion;

    return changes as Partial<BusInput>;
  }
}
