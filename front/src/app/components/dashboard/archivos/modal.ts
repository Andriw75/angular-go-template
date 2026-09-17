import { Component, inject, output, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ArchivosService } from '../../../services/archivos.service';
import { ToastService } from '../../../services/toast.service';
import type { Visibilidad } from '../../../models/archivo';

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

    this.uploading.set(true);
    this.error.set('');

    this.service.upload(this.selectedFiles(), this.visibilidad()).subscribe({
      next: () => {
        this.toast.success(`${this.selectedFiles().length} archivo(s) subido(s)`);
        this.uploading.set(false);
        this.onSaved.emit();
        this.close();
      },
      error: () => {
        this.uploading.set(false);
        this.error.set('Error al subir archivos');
      },
    });
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
