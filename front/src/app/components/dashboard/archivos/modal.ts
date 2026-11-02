import { Component, inject, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { apiErrorMessage } from '../../../common/api-error';
import { ArchivosService } from '../../../services/archivos.service';
import { ToastService } from '../../../services/toast.service';
import type { Visibilidad } from '../../../models/archivo';

const ALLOWED_EXTENSIONS = new Set([
  '.jpg', '.jpeg', '.png', '.webp', '.gif',
  '.pdf', '.doc', '.docx', '.xls', '.xlsx',
  '.txt', '.csv', '.zip', '.rar',
  '.mp4', '.mp3', '.webm',
]);

@Component({
  selector: 'app-archivo-modal',
  templateUrl: './modal.html',
  styleUrl: './modal.css',
  imports: [FormsModule],
})
export class ArchivoModalComponent {
  private service = inject(ArchivosService);
  private toast = inject(ToastService);

  onClose = output<void>();
  onSaved = output<void>();

  visibilidad = signal<Visibilidad>('publico');
  selectedFiles = signal<File[]>([]);
  uploading = signal(false);
  error = signal('');
  dragging = signal(false);

  close(): void {
    this.onClose.emit();
  }

  onFilesSelected(event: Event): void {
    const input = event.target as HTMLInputElement;
    if (input.files) {
      this.selectedFiles.update(prev => [...prev, ...Array.from(input.files!)]);
    }
    input.value = '';
  }

  onDragOver(event: DragEvent): void {
    event.preventDefault();
    this.dragging.set(true);
  }

  onDragLeave(): void {
    this.dragging.set(false);
  }

  onDrop(event: DragEvent): void {
    event.preventDefault();
    this.dragging.set(false);
    if (event.dataTransfer?.files) {
      this.selectedFiles.update(prev => [...prev, ...Array.from(event.dataTransfer!.files)]);
    }
  }

  removeFile(file: File): void {
    this.selectedFiles.update(prev => prev.filter(f => f !== file));
  }

  upload(): void {
    if (!this.selectedFiles().length) return;

    const invalid = this.selectedFiles().filter(
      (file) => !ALLOWED_EXTENSIONS.has(this.extension(file.name)),
    );
    if (invalid.length > 0) {
      this.error.set(
        `Extensión no permitida: ${invalid.map((file) => file.name).join(', ')}`,
      );
      return;
    }

    this.uploading.set(true);
    this.error.set('');

    this.service.upload(this.selectedFiles(), this.visibilidad()).subscribe({
      next: () => {
        this.toast.success(`${this.selectedFiles().length} archivo(s) subido(s)`);
        this.uploading.set(false);
        this.onSaved.emit();
        this.close();
      },
      error: (err) => {
        this.uploading.set(false);
        this.error.set(apiErrorMessage(err, 'Error al subir archivos'));
      },
    });
  }

  private extension(name: string): string {
    const index = name.lastIndexOf('.');
    return index >= 0 ? name.slice(index).toLowerCase() : '';
  }

  formatSize(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  isImage(file: File): boolean {
    return file.type.startsWith('image/');
  }

  getPreviewUrl(file: File): string {
    return URL.createObjectURL(file);
  }
}
