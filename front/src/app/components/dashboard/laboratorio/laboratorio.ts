import { Component, OnDestroy, computed, inject, signal } from '@angular/core';
import { Subscription } from 'rxjs';
import { apiErrorMessage } from '../../../common/api-error';
import { DatabaseIcon } from '../../../common/icons/database.icon';
import type { SlowQuery, SlowQueryModo } from '../../../models/lab';
import { ConfirmService } from '../../../services/confirm.service';
import { LabService } from '../../../services/lab.service';
import { SSEService } from '../../../services/sse.service';
import { ToastService } from '../../../services/toast.service';

type ExecutionMode = 'sync' | 'async';

interface SyncState {
  estado: 'ejecutando' | 'completada' | 'cancelada' | 'error';
  modo: SlowQueryModo;
  segundos: number;
  filas: number;
  iniciadoEn: number;
  finalizadoEn?: number;
  resultado?: SlowQuery;
  error?: string;
}

@Component({
  selector: 'app-laboratorio',
  templateUrl: './laboratorio.html',
  styleUrl: './laboratorio.css',
  imports: [DatabaseIcon],
})
export class LaboratorioComponent implements OnDestroy {
  private lab = inject(LabService);
  private sse = inject(SSEService);
  private toast = inject(ToastService);
  private confirm = inject(ConfirmService);

  executionMode = signal<ExecutionMode>('sync');
  modo = signal<SlowQueryModo>('sleep');
  segundos = signal(30);
  filas = signal(500_000);

  syncState = signal<SyncState | null>(null);
  syncLoading = signal(false);
  tasks = signal<SlowQuery[]>([]);
  asyncLoading = signal(false);
  cancelling = signal<Record<string, boolean>>({});

  now = signal(Date.now());

  private syncSub: Subscription | null = null;
  private sseSub: Subscription | null = null;
  private tickTimer: ReturnType<typeof setInterval> | null = null;

  readonly syncRunning = computed(() => this.syncState()?.estado === 'ejecutando');
  readonly syncElapsed = computed(() => {
    const state = this.syncState();
    if (!state) return 0;
    return Math.max(0, Math.floor(((state.finalizadoEn ?? this.now()) - state.iniciadoEn) / 1000));
  });

  constructor() {
    this.loadTasks();
    this.sseSub = this.sse.events$.subscribe((event) => {
      if (event.module !== 'laboratorio' || event.name !== 'tareas') return;

      switch (event.type) {
        case 'current':
          this.tasks.set((event.data as SlowQuery[]) ?? []);
          break;
        case 'create':
        case 'update':
          this.upsertTask(event.data as SlowQuery);
          break;
        case 'delete': {
          const { id } = event.data as { id: string };
          this.tasks.update((list) => list.filter((task) => task.id !== id));
          break;
        }
      }
    });
    this.tickTimer = setInterval(() => this.now.set(Date.now()), 1000);
  }

  setExecutionMode(mode: ExecutionMode): void {
    this.executionMode.set(mode);
  }

  setModo(value: string): void {
    this.modo.set(value === 'cpu' ? 'cpu' : 'sleep');
  }

  setSegundos(value: string): void {
    this.segundos.set(this.clamp(Number(value) || 1, 1, 600));
  }

  setFilas(value: string): void {
    this.filas.set(this.clamp(Number(value) || 1, 1, 2_000_000));
  }

  start(): void {
    if (this.executionMode() === 'sync') this.startSync();
    else this.startAsync();
  }

  startSync(): void {
    if (this.syncRunning()) return;

    const state: SyncState = {
      estado: 'ejecutando',
      modo: this.modo(),
      segundos: this.segundos(),
      filas: this.filas(),
      iniciadoEn: Date.now(),
    };

    this.syncSub?.unsubscribe();
    this.syncLoading.set(true);
    this.syncState.set(state);

    this.syncSub = this.lab
      .runSync({ modo: state.modo, segundos: state.segundos, filas: state.filas })
      .subscribe({
        next: (res) => {
          this.syncLoading.set(false);
          this.syncState.set({
            estado: res.estado === 'completada' ? 'completada' : res.estado === 'cancelada' ? 'cancelada' : 'error',
            modo: res.modo,
            segundos: res.segundos,
            filas: Number(res.filas_solicitadas),
            iniciadoEn: new Date(res.iniciado_en).getTime(),
            finalizadoEn: res.finalizado_en ? new Date(res.finalizado_en).getTime() : Date.now(),
            resultado: res,
            error: res.error,
          });
          if (res.estado === 'completada') {
            this.toast.success(`Consulta completada: ${res.filas_procesadas} filas`);
          } else if (res.estado === 'error') {
            this.toast.error(res.error || 'La consulta falló');
          }
        },
        error: (err) => {
          this.syncLoading.set(false);
          const current = this.syncState();
          if (current?.estado === 'cancelada') return;
          const message = apiErrorMessage(err, 'No se pudo ejecutar la consulta');
          this.syncState.set({
            ...state,
            estado: 'error',
            finalizadoEn: Date.now(),
            error: message,
          });
          this.toast.error(message);
        },
      });
  }

  cancelSync(): void {
    const state = this.syncState();
    if (!state || state.estado !== 'ejecutando') return;

    this.syncSub?.unsubscribe();
    this.syncSub = null;
    this.syncLoading.set(false);
    this.syncState.set({ ...state, estado: 'cancelada', finalizadoEn: Date.now() });
    this.toast.info('Petición cancelada; PostgreSQL abortará la consulta');
  }

  startAsync(): void {
    if (this.asyncLoading()) return;

    this.asyncLoading.set(true);
    this.lab.start({ modo: this.modo(), segundos: this.segundos(), filas: this.filas() }).subscribe({
      next: (task) => {
        this.asyncLoading.set(false);
        this.upsertTask(task);
        this.toast.success('Tarea iniciada');
      },
      error: (err) => {
        this.asyncLoading.set(false);
        this.toast.error(apiErrorMessage(err, 'No se pudo iniciar la tarea'));
      },
    });
  }

  async cancelTask(task: SlowQuery): Promise<void> {
    if (task.estado !== 'ejecutando' || this.cancelling()[task.id]) return;

    const ok = await this.confirm.confirm(
      'Detener tarea',
      `¿Detener la consulta ${task.modo} (${task.id.slice(0, 8)})? Se abortará en PostgreSQL.`,
      'danger',
    );
    if (!ok) return;

    this.cancelling.update((map) => ({ ...map, [task.id]: true }));
    this.lab.cancel(task.id).subscribe({
      next: (res) => {
        this.cancelling.update((map) => {
          const copy = { ...map };
          delete copy[task.id];
          return copy;
        });
        this.upsertTask(res);
        this.toast.info('Cancelación solicitada');
      },
      error: (err) => {
        this.cancelling.update((map) => {
          const copy = { ...map };
          delete copy[task.id];
          return copy;
        });
        this.toast.error(apiErrorMessage(err, 'No se pudo detener la tarea'));
      },
    });
  }

  loadTasks(): void {
    this.lab.list().subscribe({
      next: (items) => this.tasks.set(items),
      error: () => {},
    });
  }

  ngOnDestroy(): void {
    this.syncSub?.unsubscribe();
    this.syncSub = null;
    this.sseSub?.unsubscribe();
    this.sseSub = null;
    if (this.tickTimer) {
      clearInterval(this.tickTimer);
      this.tickTimer = null;
    }
  }

  estadoLabel(estado: string): string {
    switch (estado) {
      case 'ejecutando':
        return 'Ejecutando';
      case 'cancelando':
        return 'Cancelando';
      case 'cancelada':
        return 'Cancelada';
      case 'completada':
        return 'Completada';
      case 'error':
        return 'Error';
      default:
        return estado;
    }
  }

  modoLabel(modo: string): string {
    return modo === 'cpu' ? 'CPU' : 'Sleep';
  }

  shortTime(value: string): string {
    return value ? value.replace('T', ' ').slice(11, 19) : '';
  }

  segundosTranscurridos(task: SlowQuery): number {
    const start = new Date(task.iniciado_en).getTime();
    const end = task.finalizado_en ? new Date(task.finalizado_en).getTime() : this.now();
    return Math.max(0, Math.floor((end - start) / 1000));
  }

  isCancelling(id: string): boolean {
    return !!this.cancelling()[id];
  }

  private upsertTask(task: SlowQuery): void {
    this.tasks.update((list) => {
      const index = list.findIndex((x) => x.id === task.id);
      if (index === -1) return [task, ...list];
      const copy = [...list];
      copy[index] = task;
      return copy;
    });
  }

  private clamp(value: number, min: number, max: number): number {
    return Math.min(max, Math.max(min, Math.floor(value)));
  }
}
