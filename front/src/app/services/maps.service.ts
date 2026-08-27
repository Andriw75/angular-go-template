import { Injectable } from '@angular/core';
import { environment } from '../../environments/environment';
import type { MapConfig } from '../models/mapa';

@Injectable({ providedIn: 'root' })
export class MapsService {
  getConfig(): MapConfig {
    return { ...environment.MAPS };
  }
}