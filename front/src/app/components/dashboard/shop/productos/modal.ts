import { Component, effect, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ProductosService } from '../../../../services/productos.service';
import { CategoriasService } from '../../../../services/categorias.service';
import { ToastService } from '../../../../services/toast.service';
import { ImageManagerComponent } from '../../../../common/image-manager/image-manager';
import type { Producto, ProductoInput } from '../../../../models/producto';
import type { Categoria } from '../../../../models/categoria';

@Component({
  selector: 'app-producto-modal',
  templateUrl: './modal.html',
  styleUrl: './modal.css',
  imports: [FormsModule, ImageManagerComponent],
})
export class ProductoModalComponent {
  service = inject(ProductosService);
  private categoriasService = inject(CategoriasService);
  private toast = inject(ToastService);

  producto = input<Producto | null>(null);
  onClose = output<void>();
  onSaved = output<void>();

  isEditing = signal(false);
  saving = signal(false);
  error = signal('');
  categorias = signal<Categoria[]>([]);

  form: ProductoInput = {
    nombre: '',
    descripcion: '',
    precio: 0,
    stock: 0,
    categoria_id: 0,
    activo: true,
  };

  constructor() {
    this.loadCategorias();

    effect(() => {
      const p = this.producto();
      if (p) {
        this.isEditing.set(true);
        this.form = {
          nombre: p.nombre,
          descripcion: p.descripcion,
          precio: p.precio,
          stock: p.stock,
          categoria_id: p.categoria_id,
          activo: p.activo,
        };
      } else {
        this.isEditing.set(false);
        this.form = {
          nombre: '',
          descripcion: '',
          precio: 0,
          stock: 0,
          categoria_id: 0,
          activo: true,
        };
      }
      this.error.set('');
    });
  }

  private loadCategorias(): void {
    this.categoriasService.list({ limit: 100 }).subscribe({
      next: (res) => this.categorias.set(res.data),
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
    if (!this.form.categoria_id) {
      this.error.set('Debe seleccionar una categoría');
      return;
    }
    if (this.form.precio < 0 || this.form.stock < 0) {
      this.error.set('Precio y stock no pueden ser negativos');
      return;
    }

    this.saving.set(true);
    this.error.set('');

    const obs = this.isEditing()
      ? this.service.update(this.producto()!.id, this.form)
      : this.service.create(this.form);

    obs.subscribe({
      next: () => {
        this.toast.success(this.isEditing() ? 'Producto actualizado' : 'Producto creado');
        this.saving.set(false);
        this.onSaved.emit();
        this.close();
      },
      error: () => {
        this.saving.set(false);
        this.error.set('Error al guardar el producto');
      },
    });
  }
}
