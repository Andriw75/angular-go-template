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
      ? this.service.update(this.bus()!.id, this.form)
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
}
