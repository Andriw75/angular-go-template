import {
  AfterViewInit,
  Component,
  ElementRef,
  OnDestroy,
  ViewChild,
  effect,
  inject,
  signal,
} from '@angular/core';
import * as L from 'leaflet';
import { MapsService } from '../../../services/maps.service';
import { InstalacionesStoreService } from '../../../services/instalaciones-store.service';
import { ConfirmService } from '../../../services/confirm.service';
import { ToastService } from '../../../services/toast.service';
import { ListaInstalacionesComponent } from './lista';
import {
  InstalacionModalComponent,
  type InstalacionModalData,
} from './instalacion-modal';
import type { InstalacionUpdateInput } from '../../../services/instalaciones.service';
import { ESTADO_CONFIG, type Instalacion } from '../../../models/instalacion';

function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) => {
    const map: Record<string, string> = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };
    return map[c] ?? c;
  });
}

// Devuelve solo los campos que cambiaron (no enviamos todo el objeto).
function diffInstalacion(prev: Instalacion, next: Instalacion): InstalacionUpdateInput {
  const patch: Record<string, unknown> = {};
  const keys: Array<keyof Instalacion> = [
    'tipo', 'nombre', 'estado', 'lat', 'lng', 'direccion', 'zona', 'fecha_instalacion', 'serial',
    'descripcion', 'fabricante', 'modelo', 'instalador', 'proveedor', 'ultima_conexion', 'firmware',
    'ip', 'mac', 'resolucion', 'contenido', 'senal', 'bateria', 'potencia', 'contacto',
  ];
  const numericKeys = new Set<string>(['senal', 'bateria', 'potencia']);

  for (const k of keys) {
    const a = prev[k];
    const b = next[k];
    if (a === undefined && b === undefined) continue;
    if (Object.is(a, b)) continue;

    // Limpiar un opcional: strings se envían como '' y numéricos como null.
    if (a !== undefined && b === undefined) {
      patch[k] = numericKeys.has(k as string) ? null : '';
      continue;
    }

    patch[k] = b;
  }
  return patch as InstalacionUpdateInput;
}

interface CtxMenuState {
  x: number;
  y: number;
  lat: number;
  lng: number;
  target: Instalacion | null;
}

@Component({
  selector: 'app-mapas',
  imports: [ListaInstalacionesComponent, InstalacionModalComponent],
  templateUrl: './mapa.html',
  styleUrl: './mapa.css',
})
export class MapasComponent implements AfterViewInit, OnDestroy {
  readonly store = inject(InstalacionesStoreService);

  private maps = inject(MapsService);
  private confirm = inject(ConfirmService);
  private toast = inject(ToastService);

  @ViewChild('mapContainer') private container!: ElementRef<HTMLDivElement>;
  @ViewChild('mapViewport') private viewport!: ElementRef<HTMLDivElement>;

  private map?: L.Map;
  private markersLayer?: L.LayerGroup;
  private fitKey = '';
  private initialFitDone = false;
  private resizeObserver?: ResizeObserver;

  readonly contextMenu = signal<CtxMenuState | null>(null);
  readonly formData = signal<InstalacionModalData | null>(null);

  constructor() {
    effect(() => {
      const list = this.store.filtered();
      const selected = this.store.selectedId();
      if (!this.map) return;
      this.renderMarkers(list, selected);
    });

    effect(() => {
      const list = this.store.filtered();
      if (!this.map || list.length === 0) return;
      const key = list.map((x) => x.id).join(',');
      if (key === this.fitKey) return;
      this.fitKey = key;
      if (!this.initialFitDone) {
        this.initialFitDone = true;
        return;
      }
      this.map.fitBounds(
        L.latLngBounds(list.map((x) => [x.lat, x.lng] as [number, number])),
        { padding: [48, 48], maxZoom: 15 },
      );
    });

    effect(() => {
      const id = this.store.selectedId();
      if (id == null || !this.map) return;
      const item = this.store.items().find((x) => x.id === id);
      if (!item) return;
      this.map.flyTo([item.lat, item.lng], Math.max(this.map.getZoom(), 15), { duration: 0.6 });
    });
  }

  ngAfterViewInit(): void {
    const cfg = this.maps.getConfig();
    this.map = L.map(this.container.nativeElement, {
      center: [cfg.initialLat, cfg.initialLng],
      zoom: cfg.initialZoom,
    });

    L.tileLayer(cfg.tileUrl, { attribution: cfg.attribution ?? '' }).addTo(this.map);
    this.markersLayer = L.layerGroup().addTo(this.map);

    this.map.on('click', () => this.contextMenu.set(null));
    this.map.on('contextmenu', (e: L.LeafletMouseEvent) => {
      L.DomEvent.preventDefault(e.originalEvent);
      this.openContextMenu(e.originalEvent.clientX, e.originalEvent.clientY, e.latlng.lat, e.latlng.lng, null);
    });

    this.resizeObserver = new ResizeObserver(() => this.map?.invalidateSize());
    this.resizeObserver.observe(this.container.nativeElement);

    this.store.load();
  }

  private renderMarkers(list: Instalacion[], selected: number | null): void {
    this.markersLayer?.clearLayers();
    for (const item of list) {
      const marker = L.marker([item.lat, item.lng], {
        icon: this.pinIcon(item, item.id === selected),
      });
      marker.bindPopup(this.popupHtml(item));
      marker.on('click', () => {
        this.contextMenu.set(null);
        this.store.select(item.id);
      });
      marker.on('contextmenu', (e: L.LeafletMouseEvent) => {
        L.DomEvent.stopPropagation(e.originalEvent);
        L.DomEvent.preventDefault(e.originalEvent);
        this.openContextMenu(e.originalEvent.clientX, e.originalEvent.clientY, item.lat, item.lng, item);
      });
      this.markersLayer?.addLayer(marker);
      if (item.id === selected) marker.openPopup();
    }
  }

  private openContextMenu(clientX: number, clientY: number, lat: number, lng: number, target: Instalacion | null): void {
    const rect = this.viewport.nativeElement.getBoundingClientRect();
    this.contextMenu.set({
      x: clientX - rect.left,
      y: clientY - rect.top,
      lat,
      lng,
      target,
    });
  }

  private pinIcon(item: Instalacion, selected: boolean): L.DivIcon {
    const color = ESTADO_CONFIG[item.estado].color;
    const size = selected ? 40 : 30;
    const ring = selected
      ? `<circle cx="14" cy="14" r="22" fill="none" stroke="${color}" stroke-width="5" stroke-opacity="0.35"/>`
      : '';
    const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${size}" height="${size}" viewBox="0 0 28 28">${ring}<path d="M14 2C10 2 7 5 7 9c0 6 7 15 7 15s7-9 7-15c0-4-3-7-7-7z" fill="${color}" stroke="#fff" stroke-width="1.6"/><circle cx="14" cy="9" r="3" fill="#fff"/></svg>`;
    return L.divIcon({
      className: 'map-pin',
      html: svg,
      iconSize: L.point(size, size),
      iconAnchor: L.point(size / 2, size - 4),
      popupAnchor: L.point(0, -(size - 4)),
    });
  }

  private popupHtml(item: Instalacion): string {
    const st = ESTADO_CONFIG[item.estado];
    const esc = escapeHtml;
    return `
      <div class="map-popup">
        <div class="map-popup-title">${esc(item.nombre)}</div>
        <div class="map-popup-tags">
          <span class="map-popup-badge map-popup-badge--tipo">${esc(item.tipo)}</span>
          <span class="map-popup-badge" style="background:${st.background};color:${st.color}">${esc(st.label)}</span>
          <span class="map-popup-badge map-popup-badge--zona">${esc(item.zona)}</span>
        </div>
        <div class="map-popup-row">${esc(item.direccion)}</div>
        <div class="map-popup-row map-popup-muted">${item.lat.toFixed(5)}, ${item.lng.toFixed(5)}</div>
        <div class="map-popup-action">Clic derecho o lista para editar</div>
      </div>`;
  }

  createAt(menu: CtxMenuState): void {
    this.contextMenu.set(null);
    this.formData.set({ mode: 'create', lat: menu.lat, lng: menu.lng });
  }

  editTarget(menu: CtxMenuState): void {
    this.contextMenu.set(null);
    if (!menu.target) return;
    this.formData.set({ mode: 'edit', lat: menu.lat, lng: menu.lng, instalacion: menu.target });
  }

  async deleteTarget(menu: CtxMenuState): Promise<void> {
    this.contextMenu.set(null);
    if (!menu.target) return;
    const ok = await this.confirm.confirm(
      'Eliminar instalación',
      `¿Seguro de eliminar "${menu.target.nombre}"?`,
      'danger',
    );
    if (!ok) return;
    this.store.remove(menu.target.id).subscribe({
      next: () => this.toast.success('Instalación eliminada'),
      error: () => this.toast.error('Error al eliminar instalación'),
    });
  }

  editFromList(item: Instalacion): void {
    this.formData.set({ mode: 'edit', lat: item.lat, lng: item.lng, instalacion: item });
  }

  async deleteFromList(item: Instalacion): Promise<void> {
    const ok = await this.confirm.confirm(
      'Eliminar instalación',
      `¿Seguro de eliminar "${item.nombre}"?`,
      'danger',
    );
    if (!ok) return;
    this.store.remove(item.id).subscribe({
      next: () => this.toast.success('Instalación eliminada'),
      error: () => this.toast.error('Error al eliminar instalación'),
    });
  }

  onModalSaved(item: Instalacion): void {
    const data = this.formData();
    if (data?.mode === 'create') {
      const { id: _id, ...fields } = item;
      this.store.create(fields).subscribe({
        next: () => this.toast.success('Instalación creada'),
        error: () => this.toast.error('Error al crear instalación'),
      });
    } else if (data?.mode === 'edit') {
      const current = this.store.items().find((x) => x.id === item.id);
      const patch = current ? diffInstalacion(current, item) : {};
      this.store.update(item.id, patch).subscribe({
        next: () => this.toast.success('Instalación actualizada'),
        error: () => this.toast.error('Error al actualizar instalación'),
      });
    }
    this.formData.set(null);
  }

  onModalClosed(): void {
    this.formData.set(null);
  }

  ngOnDestroy(): void {
    this.resizeObserver?.disconnect();
    this.map?.remove();
  }
}