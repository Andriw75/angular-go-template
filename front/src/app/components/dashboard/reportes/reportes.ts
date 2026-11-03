import { Component, computed, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { apiErrorMessage } from '../../../common/api-error';
import { ConfirmService } from '../../../services/confirm.service';
import { ReportExportService } from '../../../services/report-export.service';
import { ReportesService, type ReporteFilters } from '../../../services/reportes.service';
import { ToastService } from '../../../services/toast.service';
import { VentasService } from '../../../services/ventas.service';
import type { Documento, DocumentoResult } from '../../../models/reporte';
import type { Venta } from '../../../models/venta';

type Source = 'buses' | 'ventas';
type Engine = 'frontend' | 'backend';
type Format = 'pdf' | 'xlsx';
type Mode = 'report' | 'document';

@Component({
  selector: 'app-reportes',
  templateUrl: './reportes.html',
  styleUrl: './reportes.css',
  imports: [FormsModule],
})
export class ReportesComponent {
  private reportes = inject(ReportesService);
  private exporter = inject(ReportExportService);
  private ventasService = inject(VentasService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmService);

  source = signal<Source>('buses');
  engine = signal<Engine>('backend');
  format = signal<Format>('pdf');
  mode = signal<Mode>('report');
  politica = signal<'inmutable' | 'actualizable'>('inmutable');
  dataMode = signal<'actual' | 'generado'>('actual');
  generatedCount = signal(50);
  readonly maxGenerated = 1000;

  q = signal('');
  tipo = signal('');
  activo = signal('');
  desde = signal('');
  hasta = signal('');
  serie = signal('');
  estado = signal('');

  template = signal('buses-a4-p');
  count = signal<number | null>(null);
  loading = signal(false);
  generating = signal(false);
  progress = signal(0);
  result = signal<Documento | null>(null);

  ventas = signal<Venta[]>([]);
  ventaId = signal<number | null>(null);

  documents = signal<Documento[]>([]);
  docsLoading = signal(false);

  readonly templates = computed(() => {
    if (this.source() === 'ventas') {
      return [
        { value: 'venta-factura', label: 'Factura A4 vertical' },
        { value: 'venta-ticket', label: 'Ticket 80 mm dinámico' },
        { value: 'venta-nota', label: 'Nota de venta A4' },
      ];
    }
    return [
      { value: 'buses-a4-p', label: 'A4 vertical con paginación' },
      { value: 'buses-a4-l', label: 'A4 horizontal' },
      { value: 'buses-mixto', label: 'Mixto (vertical + horizontal)' },
      { value: 'buses-dinamico', label: 'Alto dinámico según contenido' },
    ];
  });

  constructor() {
    this.loadVentas();
    this.loadDocuments();
    this.setSource('buses');
  }

  setSource(source: Source): void {
    this.source.set(source);
    this.template.set(source === 'ventas' ? 'venta-factura' : 'buses-a4-p');
    this.count.set(null);
    this.result.set(null);
    if (source === 'ventas') {
      this.dataMode.set('actual');
      this.loadVentas();
    }
  }

  setDataMode(mode: 'actual' | 'generado'): void {
    if (this.source() === 'ventas') return;
    this.dataMode.set(mode);
    this.count.set(null);
  }

  setGeneratedCount(value: number): void {
    const count = Math.floor(Number(value) || 1);
    this.generatedCount.set(Math.min(this.maxGenerated, Math.max(1, count)));
    this.count.set(null);
  }

  setMode(mode: Mode): void {
    this.mode.set(mode);
  }

  filters(): ReporteFilters {
    const base: ReporteFilters = {
      q: this.q() || undefined,
      desde: this.desde() || undefined,
      hasta: this.hasta() || undefined,
    };
    if (this.source() === 'buses') {
      base.tipo = this.tipo() || undefined;
      base.activo = this.activo() ? this.activo() === 'true' : undefined;
      base.modo = this.dataMode();
      if (this.dataMode() === 'generado') base.cantidad = this.generatedCount();
    } else {
      base.serie = this.serie() || undefined;
      base.estado = this.estado() || undefined;
    }
    return base;
  }

  countNow(): void {
    this.loading.set(true);
    const request = this.source() === 'buses'
      ? this.reportes.busesCount(this.filters())
      : this.reportes.ventasCount(this.filters());
    request.subscribe({
      next: (res) => { this.count.set(res.total); this.loading.set(false); },
      error: (err) => { this.loading.set(false); this.toast.error(apiErrorMessage(err, 'Error al contar')); },
    });
  }

  generate(): void {
    if (this.source() === 'ventas' && !this.ventaId()) {
      this.toast.error('Selecciona una venta emitida');
      return;
    }
    if (this.mode() === 'document') {
      this.generateDocument();
      return;
    }
    if (this.engine() === 'frontend') {
      this.generateFrontend();
      return;
    }
    this.downloadBackendReport();
  }

  private generateDocument(): void {
    this.generating.set(true);
    this.progress.set(0);
    this.result.set(null);

    const request = this.source() === 'buses'
      ? this.reportes.generateBusesDocument({
          serie: this.dataMode() === 'generado' ? 'GEN' : 'BUS',
          plantilla: this.template(),
          politica: this.politica(),
          q: this.q(),
          tipo: this.tipo(),
          activo: this.activo() ? this.activo() === 'true' : undefined,
          desde: this.desde(),
          hasta: this.hasta(),
          modo: this.dataMode(),
          cantidad: this.dataMode() === 'generado' ? this.generatedCount() : undefined,
        })
      : this.reportes.generateVentaDocument(this.ventaId()!, {
          serie: this.serie(),
          plantilla: this.template(),
          politica: this.politica(),
        });

    request.subscribe({
      next: (res) => this.handleDocumentResult(res),
      error: (err) => { this.generating.set(false); this.toast.error(apiErrorMessage(err, 'Error al generar documento')); },
    });
  }

  private handleDocumentResult(res: DocumentoResult): void {
    if (res.documento) {
      this.finishDocument(res.documento, res.exists === true);
      return;
    }
    if (res.job_id) {
      this.pollJob(res.job_id);
      return;
    }
    this.generating.set(false);
    this.toast.error('Respuesta inesperada del servidor');
  }

  private pollJob(id: string): void {
    this.reportes.job(id).subscribe({
      next: (res) => {
        this.progress.set(res.progreso ?? 0);
        if (res.estado === 'listo' && res.documento) {
          this.finishDocument(res.documento, res.exists === true);
        } else if (res.estado === 'error') {
          this.generating.set(false);
          this.toast.error(res.error || 'Error al generar documento');
        } else {
          setTimeout(() => this.pollJob(id), 2000);
        }
      },
      error: (err) => { this.generating.set(false); this.toast.error(apiErrorMessage(err, 'Error consultando el trabajo')); },
    });
  }

  private finishDocument(doc: Documento, exists: boolean): void {
    this.generating.set(false);
    this.progress.set(100);
    this.result.set(doc);
    this.toast.success(exists ? `El documento ya existía: ${doc.codigo}` : `Documento generado: ${doc.codigo}`);
    this.loadDocuments();
  }

  private generateFrontend(): void {
    this.generating.set(true);
    if (this.source() === 'buses') {
      this.reportes.buses(this.filters()).subscribe({
        next: async (report) => {
          try {
            if (this.format() === 'pdf') await this.exporter.exportBusesPdf(report, this.template());
            else await this.exporter.exportBusesExcel(report);
          } finally {
            this.generating.set(false);
          }
        },
        error: (err) => { this.generating.set(false); this.toast.error(apiErrorMessage(err, 'Error al obtener datos')); },
      });
      return;
    }

    const id = this.ventaId();
    if (!id) { this.generating.set(false); return; }
    this.ventasService.getByID(id).subscribe({
      next: async (venta) => {
        try {
          if (this.format() === 'pdf') await this.exporter.exportVentaPdf(venta, this.template());
          else await this.exporter.exportVentaExcel(venta);
        } finally {
          this.generating.set(false);
        }
      },
      error: (err) => { this.generating.set(false); this.toast.error(apiErrorMessage(err, 'Error al obtener la venta')); },
    });
  }

  private downloadBackendReport(): void {
    this.generating.set(true);
    const request = this.source() === 'buses'
      ? (this.format() === 'pdf'
          ? this.reportes.downloadBusesPdf(this.filters(), this.template())
          : this.reportes.downloadBusesXlsx(this.filters()))
      : (this.format() === 'pdf'
          ? this.reportes.downloadVentaPdf(this.ventaId()!, this.template())
          : this.reportes.downloadVentaXlsx(this.ventaId()!));

    request.subscribe({
      next: (blob) => {
        this.saveBlob(blob, `reporte-${this.source()}-${Date.now()}.${this.format()}`);
        this.generating.set(false);
      },
      error: (err) => { this.generating.set(false); this.toast.error(apiErrorMessage(err, 'Error al descargar reporte')); },
    });
  }

  typeLabel(tipo: string): string {
    if (tipo === 'reporte_buses') return 'Reporte de buses';
    if (tipo === 'venta') return 'Venta';
    return tipo;
  }

  shortDate(value: string): string {
    return value ? value.replace('T', ' ').slice(0, 16) : '';
  }

  downloadDocument(doc: Documento, formato: 'pdf' | 'xlsx'): void {
    this.reportes.documentFile(doc.id, formato).subscribe({
      next: (blob) => this.saveBlob(blob, `${doc.codigo}.${formato}`),
      error: (err) => this.toast.error(apiErrorMessage(err, 'Error al descargar documento')),
    });
  }

  async annulDocument(doc: Documento): Promise<void> {
    const ok = await this.confirm.confirm('Anular documento', `¿Anular ${doc.codigo}?`, 'danger');
    if (!ok) return;
    this.reportes.annulDocument(doc.id, 'Anulado desde Reportes').subscribe({
      next: () => { this.toast.success('Documento anulado'); this.loadDocuments(); },
      error: (err) => this.toast.error(apiErrorMessage(err, 'Error al anular documento')),
    });
  }

  loadDocuments(): void {
    this.docsLoading.set(true);
    this.reportes.documents({ limit: 20 }).subscribe({
      next: (res) => { this.documents.set(res.data); this.docsLoading.set(false); },
      error: (err) => { this.docsLoading.set(false); this.toast.error(apiErrorMessage(err, 'Error al cargar documentos')); },
    });
  }

  private loadVentas(): void {
    this.ventasService.list({ limit: 100, estado: 'emitido' }).subscribe({
      next: (res) => {
        this.ventas.set(res.data);
        if (!this.ventaId() && res.data.length) this.ventaId.set(res.data[0].id);
      },
    });
  }

  private saveBlob(blob: Blob, filename: string): void {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
}
