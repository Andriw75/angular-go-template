import { Component, effect, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CategoriasService } from '../../../../services/categorias.service';
import { ToastService } from '../../../../services/toast.service';
import { ImageManagerComponent } from '../../../../common/image-manager/image-manager';
import type { Categoria, CategoriaInput } from '../../../../models/categoria';

@Component({
  selector: 'app-categoria-modal',
  templateUrl: './modal.html',
  styleUrl: './modal.css',
  imports: [FormsModule, ImageManagerComponent],
})
export class CategoriaModalComponent {
  service = inject(CategoriasService);
  private toast = inject(ToastService);

  categoria = input<Categoria | null>(null);
  onClose = output<void>();
  onSaved = output<void>();

  isEditing = signal(false);
  saving = signal(false);
  error = signal('');

  form: CategoriaInput = {
    nombre: '',
    descripcion: '',
    activo: true,
  };

  constructor() {
    effect(() => {
      const c = this.categoria();
      if (c) {
        this.isEditing.set(true);
        this.form = {
          nombre: c.nombre,
          descripcion: c.descripcion,
          activo: c.activo,
        };
      } else {
        this.isEditing.set(false);
        this.form = { nombre: '', descripcion: '', activo: true };
      }
      this.error.set('');
    });
  }

  close(): void {
    this.onClose.emit();
  }

  onSubmit(): void {
    if (!this.form.nombre) {
      this.error.set('Nombre es obligatorio');
      return;
    }

    this.saving.set(true);
    this.error.set('');

    const obs = this.isEditing()
      ? this.service.update(this.categoria()!.id, this.buildChanges(this.categoria()!))
      : this.service.create(this.form);

    obs.subscribe({
      next: () => {
        this.toast.success(this.isEditing() ? 'Categoría actualizada' : 'Categoría creada');
        this.saving.set(false);
        this.onSaved.emit();
        this.close();
      },
      error: () => {
        this.saving.set(false);
        this.error.set('Error al guardar la categoría');
      },
    });
  }

  private buildChanges(original: Categoria): Partial<CategoriaInput> {
    const changes: Record<string, unknown> = {};
    if (this.form.nombre !== original.nombre) changes['nombre'] = this.form.nombre;
    if (this.form.descripcion !== original.descripcion) changes['descripcion'] = this.form.descripcion;
    if (this.form.activo !== original.activo) changes['activo'] = this.form.activo;
    return changes as Partial<CategoriaInput>;
  }
}
