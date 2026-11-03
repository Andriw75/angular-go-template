import { Component, effect, inject, input, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { apiErrorMessage } from '../../../common/api-error';
import { ProductosService } from '../../../services/productos.service';
import { VentasService } from '../../../services/ventas.service';
import type { Producto } from '../../../models/producto';
import type { Venta, VentaInput, VentaItemInput } from '../../../models/venta';

@Component({
  selector: 'app-venta-modal',
  templateUrl: './modal.html',
  styleUrl: './modal.css',
  imports: [FormsModule],
})
export class VentaModalComponent {
  private service = inject(VentasService);
  private productosService = inject(ProductosService);

  venta = input<Venta | null>(null);
  close = output<void>();
  saved = output<void>();

  productos = signal<Producto[]>([]);
  saving = signal(false);
  error = signal('');

  serie = 'FAC';
  clienteNombre = '';
  clienteDocumento = '';
  clienteDireccion = '';
  fecha = new Date().toISOString().slice(0, 10);
  moneda = 'PEN';
  observaciones = '';
  items: VentaItemInput[] = [];

  constructor() {
    this.loadProductos();
    effect(() => {
      const v = this.venta();
      if (v) {
        this.serie = v.serie;
        this.clienteNombre = v.cliente_nombre;
        this.clienteDocumento = v.cliente_documento;
        this.clienteDireccion = v.cliente_direccion;
        this.fecha = v.fecha;
        this.moneda = v.moneda;
        this.observaciones = v.observaciones;
        this.items = v.items.map((it) => ({
          producto_id: it.producto_id,
          descripcion: it.descripcion,
          cantidad: it.cantidad,
          precio_unitario: it.precio_unitario,
          descuento: it.descuento,
          impuesto_porcentaje: it.impuesto_porcentaje,
        }));
      } else {
        this.items = [this.blankItem()];
      }
      this.error.set('');
    });
  }

  private loadProductos(): void {
    this.productosService.list({ limit: 100 }).subscribe({
      next: (res) => this.productos.set(res.data),
    });
  }

  private blankItem(): VentaItemInput {
    return {
      descripcion: '',
      cantidad: 1,
      precio_unitario: 0,
      descuento: 0,
      impuesto_porcentaje: 18,
    };
  }

  addItem(): void {
    this.items = [...this.items, this.blankItem()];
  }

  removeItem(index: number): void {
    this.items = this.items.filter((_, i) => i !== index);
    if (!this.items.length) this.items = [this.blankItem()];
  }

  addProduct(event: Event): void {
    const select = event.target as HTMLSelectElement;
    const id = Number(select.value);
    if (!id) return;
    const producto = this.productos().find((p) => p.id === id);
    if (producto) {
      this.items = [
        ...this.items,
        {
          producto_id: producto.id,
          descripcion: producto.nombre,
          cantidad: 1,
          precio_unitario: producto.precio,
          descuento: 0,
          impuesto_porcentaje: 18,
        },
      ];
    }
    select.value = '';
  }

  lineBase(it: VentaItemInput): number {
    return Math.max(0, it.cantidad * it.precio_unitario - it.descuento);
  }

  lineTax(it: VentaItemInput): number {
    return this.lineBase(it) * it.impuesto_porcentaje / 100;
  }

  lineTotal(it: VentaItemInput): number {
    return this.lineBase(it) + this.lineTax(it);
  }

  subtotal(): number {
    return this.items.reduce((sum, it) => sum + this.lineBase(it), 0);
  }

  impuesto(): number {
    return this.items.reduce((sum, it) => sum + this.lineTax(it), 0);
  }

  total(): number {
    return this.subtotal() + this.impuesto();
  }

  save(): void {
    if (!this.clienteNombre.trim()) {
      this.error.set('El cliente es obligatorio');
      return;
    }
    const invalid = this.items.some((it) => !it.descripcion.trim() || it.cantidad <= 0 || it.precio_unitario < 0);
    if (invalid) {
      this.error.set('Completa los ítems: descripción, cantidad y precio');
      return;
    }

    const input: VentaInput = {
      serie: this.serie,
      cliente_nombre: this.clienteNombre,
      cliente_documento: this.clienteDocumento,
      cliente_direccion: this.clienteDireccion,
      fecha: this.fecha,
      moneda: this.moneda,
      observaciones: this.observaciones,
      items: this.items,
    };

    this.saving.set(true);
    this.error.set('');
    const current = this.venta();
    const req = current ? this.service.update(current.id, input) : this.service.create(input);
    req.subscribe({
      next: () => {
        this.saving.set(false);
        this.saved.emit();
      },
      error: (err) => {
        this.saving.set(false);
        this.error.set(apiErrorMessage(err, 'Error al guardar la venta'));
      },
    });
  }

  cancel(): void {
    this.close.emit();
  }
}
