import { Component, effect, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { CategoriasService } from '../../../../services/categorias.service';
import { ToastService } from '../../../../services/toast.service';
import { environment } from '../../../../../environments/environment';
import type { Categoria, CategoriaInput } from '../../../../models/categoria';
import type { Imagen } from '../../../../models/imagen';

@Component({
  selector: 'app-categoria-modal',
  templateUrl: './modal.html',
  styleUrl: './modal.css',
  imports: [FormsModule],
})
export class CategoriaModalComponent {
  private service = inject(CategoriasService);
  private toast = inject(ToastService);

  categoria = input<Categoria | null>(null);
  onClose = output<void>();
  onSaved = output<void>();

  isEditing = signal(false);
  saving = signal(false);
  error = signal('');

  imagenes = signal<Imagen[]>([]);
  uploading = signal(false);

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
        this.imagenes.set(c.imagenes ?? []);
      } else {
        this.isEditing.set(false);
        this.form = { nombre: '', descripcion: '', activo: true };
        this.imagenes.set([]);
      }
      this.error.set('');
    });
  }

  close(): void {
    this.onClose.emit();
  }

  imageUrl(img: Imagen): string {
    return environment.MEDIA_URL + img.url;
  }

  onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    const files = input.files ? Array.from(input.files) : [];
    if (!files.length) return;

    this.uploading.set(true);
    this.service.uploadImagenes(this.categoria()!.id, files).subscribe({
      next: (imgs) => {
        this.imagenes.set(imgs);
        this.uploading.set(false);
        input.value = '';
        this.toast.success('Imágenes subidas');
      },
      error: () => {
        this.uploading.set(false);
        input.value = '';
        this.toast.error('Error al subir imágenes');
      },
    });
  }

  removeImagen(img: Imagen): void {
    this.service.deleteImagen(this.categoria()!.id, img.id).subscribe({
      next: () => {
        this.imagenes.update((list) => list.filter((i) => i.id !== img.id));
        this.toast.success('Imagen eliminada');
      },
      error: () => this.toast.error('Error al eliminar imagen'),
    });
  }

  onSubmit(): void {
    if (!this.form.nombre) {
      this.error.set('Nombre es obligatorio');
      return;
    }

    this.saving.set(true);
    this.error.set('');

    const obs = this.isEditing()
      ? this.service.update(this.categoria()!.id, this.form)
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
}
