import { Injectable } from '@angular/core';
import { environment } from '../../environments/environment';
import type { MapConfig } from '../models/mapa';

const DEFAULT_ATTRIBUTION = '&copy; OpenStreetMap contributors';

@Injectable({ providedIn: 'root' })
export class MapsService {
  getConfig(): MapConfig {
    const m = environment.MAPS as MapConfig;
    return { ...m, attribution: m.attribution ?? DEFAULT_ATTRIBUTION };
  }
}