import { Component, ElementRef, OnDestroy, ViewChild, inject, signal } from '@angular/core';
import { NgIf, DecimalPipe } from '@angular/common';
import * as L from 'leaflet';
import { MapsService } from '../../../services/maps.service';
import type { MapConfig } from '../../../models/mapa';

@Component({
  selector: 'app-mapas',
  templateUrl: './mapa.html',
  styleUrl: './mapa.css',
  imports: [NgIf, DecimalPipe],
})
export class MapasComponent implements OnDestroy {
  private maps = inject(MapsService);

  @ViewChild('mapContainer') private container!: ElementRef<HTMLDivElement>;

  config = signal<MapConfig | null>(null);

  private map?: L.Map;

  constructor() {
    this.config.set(this.maps.getConfig());
  }

  ngAfterViewInit(): void {
    const cfg = this.config();
    if (!cfg || !this.container) return;

    this.map = L.map(this.container.nativeElement, {
      center: [cfg.initialLat, cfg.initialLng],
      zoom: cfg.initialZoom,
    });

    L.tileLayer(cfg.tileUrl).addTo(this.map);
  }

  ngOnDestroy(): void {
    this.map?.remove();
  }
}