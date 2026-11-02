# Flujo de instalaciones y mapas

## Resumen

Mapa interactivo de instalaciones (cámaras, paneles, banners, sensores y
routers) usando **Leaflet** con tiles de OpenStreetMap.

- No requiere permiso específico: la ruta usa `AuthMiddleware()` y el menú
  **Mapas** no tiene `requiredPermission`. Cualquier usuario logueado puede
  entrar.
- `GET /instalaciones` devuelve **todo** el listado en una sola llamada; los
  filtros y la búsqueda se resuelven en cliente con un store.
- Se puede crear, editar y eliminar desde el mapa (clic derecho), desde la lista
  lateral o desde el modal.
- En modo base de datos **no hay seed**: el mapa arranca vacío.

## Modelo

`domain.Instalacion` (`back/domain/instalacion.go`):

| Campo               | Tipo      | Notas                                      |
| ------------------- | --------- | ------------------------------------------ |
| `id`                | int64     | PK                                         |
| `tipo`              | string    | `CAMARA`, `PANEL`, `BANNER`, `SENSOR`, `ROUTER` |
| `nombre`            | string    | Obligatorio                                |
| `estado`            | string    | Ver catálogo de estados                    |
| `lat`, `lng`        | float64   | Coordenadas                                |
| `direccion`         | string    |                                            |
| `zona`              | string    | `CENTRO`, `NORTE`, `SUR`, `ESTE`, `OESTE`  |
| `fecha_instalacion` | string    | `YYYY-MM-DD`                               |
| `serial`            | string    |                                            |
| `descripcion`       | *string   | Opcional                                   |
| `fabricante`        | *string   | Opcional                                   |
| `modelo`            | *string   | Opcional                                   |
| `instalador`        | *string   | Opcional                                   |
| `proveedor`         | *string   | Opcional                                   |
| `ultima_conexion`   | *string   | Opcional, ISO                              |
| `firmware`          | *string   | Opcional                                   |
| `ip`                | *string   | Opcional                                   |
| `mac`               | *string   | Opcional                                   |
| `resolucion`        | *string   | Opcional                                   |
| `contenido`         | *string   | Opcional                                   |
| `senal`             | *float64  | Opcional                                   |
| `bateria`           | *float64  | Opcional                                   |
| `potencia`          | *float64  | Opcional                                   |
| `contacto`          | *string   | Opcional                                   |
| `creado_en`         | time.Time |                                            |
| `actualizado_en`    | time.Time |                                            |

Catálogos en `front/src/app/models/instalacion.ts`:

- `INSTALACION_TIPOS`
- `INSTALACION_ESTADOS`: `ACTIVO`, `INACTIVO`, `MANTENIMIENTO`, `MALOGRADO`,
  `PENDIENTE_INSTALACION`, `PENDIENTE_REPARACION`.
- `INSTALACION_ZONAS`
- `ESTADO_CONFIG`: etiqueta, color y fondo por estado (usado por chips, badges y
  marcadores).

## Backend

### Rutas

Todas bajo `AuthMiddleware()` (`back/api/router.go:119`), sin permiso adicional:

| Método | Ruta                |
| ------ | ------------------- |
| GET    | `/instalaciones`    |
| POST   | `/instalaciones`    |
| PUT    | `/instalaciones/{id}`|
| DELETE | `/instalaciones/{id}`|

No hay `/count` ni paginación.

### Listar

`InstalacionHandler.List` devuelve todas las instalaciones ordenadas por `id`.
El frontend filtra en cliente.

### Crear

`POST /instalaciones`:

- Obligatorios: `tipo` y `nombre`.
- Defaults del backend si faltan:
  - `estado` → `PENDIENTE_INSTALACION`
  - `direccion` → `Sin dirección`
  - `zona` → `CENTRO`
  - `fecha_instalacion` → fecha actual
  - `serial` → `SN-<timestamp>`
- El resto son opcionales.
- Responde `201`.

### Actualizar

`PUT /instalaciones/{id}` con actualización parcial:

- Campos simples: se aplican si llegan.
- Strings opcionales: `applyOptionalString` los limpia si llega `""`.
- Numéricos opcionales (`senal`, `bateria`, `potencia`): `applyOptionalNumber`
  los limpia si llega `null`.
- `ultima_conexion` es especial: si llega, se asigna el puntero recibido; el
  frontend envía `""` al limpiarlo, por lo que queda como string vacío, no
  necesariamente `NULL`.
- Responde `200` con la instalación actualizada.

### Eliminar

`DELETE /instalaciones/{id}`: borrado físico; `404` si no existe.

### Validaciones

El backend **no valida** contra los catálogos de `tipo`, `estado` o `zona`, ni
rangos de latitud/longitud. El frontend restringe por tipos TypeScript, pero una
llamada directa a la API podría insertar valores fuera del catálogo.

## Frontend: cómo se usa Leaflet

### Dependencias y configuración

- `leaflet` y `@types/leaflet` en `front/package.json`.
- El CSS se importa globalmente en `front/src/styles.css`:
  ```css
  @import 'leaflet/dist/leaflet.css';
  ```
- Configuración en `environment.MAPS` (`front/src/environments/environment.ts`):
  ```ts
  MAPS: {
    initialLat: -12.005424268674062,
    initialLng: -77.06810865375328,
    initialZoom: 15.5,
    tileUrl: 'https://tile.openstreetmap.org/{z}/{x}/{y}.png',
  },
  ```
- `MapsService` devuelve la configuración y añade la atribución por defecto de
  OpenStreetMap.
- No se usa API key. Para producción conviene revisar la política de uso de
  tiles de OSM o usar un proveedor con cuota/contrato.

### Inicialización

`MapasComponent` (`front/src/app/components/dashboard/mapas/mapa.ts`):

1. `ngAfterViewInit` obtiene la configuración con `MapsService`.
2. Crea el mapa:
   ```ts
   this.map = L.map(container, {
     center: [cfg.initialLat, cfg.initialLng],
     zoom: cfg.initialZoom,
   });
   ```
3. Añade la capa de tiles:
   ```ts
   L.tileLayer(cfg.tileUrl, { attribution: cfg.attribution ?? '' }).addTo(this.map);
   ```
4. Crea un `L.layerGroup()` para los marcadores.
5. Registra eventos:
   - `click` en el mapa → cierra el menú contextual.
   - `contextmenu` → abre el menú para crear en esas coordenadas.
6. Crea un `ResizeObserver` que llama `map.invalidateSize()` cuando cambia el
   contenedor.
7. Llama `store.load()` para traer las instalaciones.
8. En `ngOnDestroy`: desconecta el observer y ejecuta `map.remove()`.

El contenedor necesita altura definida; el layout del mapa lo resuelve con
`flex` y `height: 100%` en `mapa.css`.

### Store de instalaciones

`InstalacionesStoreService` (`front/src/app/services/instalaciones-store.service.ts`):

- Signals: `items`, `loading`, `selectedId`, `q`, `tipo`, `estado`, `zona`.
- Computed:
  - `total`: cantidad de items.
  - `filtered`: aplica `q` (nombre, dirección, serial, contenido, descripción),
    tipo, estado y zona.
  - `countsByEstado`: conteo por estado para los chips.
- Métodos:
  - `load()`: `GET /instalaciones` y reemplaza `items`.
  - `create()`, `update()`, `remove()`: llaman a la API y actualizan la lista
    local.
  - `select(id)`, `toggleEstado(estado)`, `resetFilters()`.

Es un **patrón de estado en cliente**: no hay SSE ni paginación; se cargan todas
las instalaciones y se filtran localmente.

### Marcadores

`renderMarkers(list, selected)`:

- Limpia el `layerGroup` y vuelve a crear un `L.marker` por instalación.
- Usa un `L.divIcon` propio (`pinIcon`) con SVG:
  - Color según `ESTADO_CONFIG[item.estado].color`.
  - Tamaño mayor y anillo si está seleccionado.
- `bindPopup(popupHtml(item))` con nombre, tipo, estado, zona, dirección y
  coordenadas. El HTML se escapa con `escapeHtml`.
- Click en marcador → `store.select(item.id)`.
- Clic derecho en marcador → menú contextual para editar/eliminar.

Se evita el problema clásico de los iconos por defecto de Leaflet (rutas de
assets) usando `DivIcon` con SVG embebido.

### Interacción con el mapa

- **Clic derecho en el mapa vacío** → menú “Crear instalación aquí” con las
  coordenadas del clic.
- **Clic derecho en un marcador** → “Editar” / “Eliminar”.
- **Clic izquierdo en el mapa** → cierra el menú contextual.
- **Selección desde la lista** → `flyTo` al marcador con zoom mínimo 15.
- **Cambio de filtros** → se vuelven a renderizar marcadores.
- **`fitBounds`**: cuando cambia el conjunto filtrado (y no es la primera vez),
  el mapa encuadra todos los resultados con padding y `maxZoom: 15`. La primera
  carga respeta el centro/zoom configurado.
- El menú contextual se posiciona relativo al viewport (`getBoundingClientRect`).

### Lista lateral

`ListaInstalacionesComponent`:

- Chips por estado con conteo (`countsByEstado`).
- Filtros: búsqueda `q`, tipo, estado y zona.
- Tarjetas con nombre, dirección, badges de tipo/estado/zona, serial, fecha y
  coordenadas.
- Al seleccionar una tarjeta, se marca en el mapa y se hace scroll hacia ella.
- Al expandirse muestra campos opcionales presentes y botones Editar/Eliminar.
- Emite `edit` y `remove` al componente padre.

### Modal de instalación

`InstalacionModalComponent`:

- Recibe `{ mode, lat, lng, instalacion? }`.
- Usa signals para cada campo.
- **Crear**: defaults actuales (serial aleatorio, fecha de hoy, tipo `CAMARA`,
  estado `PENDIENTE_INSTALACION`, zona `CENTRO`).
- **Editar**: precarga todos los campos.
- Valida nombre, latitud/longitud y fecha.
- Emite un `Instalacion` completo; el padre decide crear o actualizar.
- `ultima_conexion` se envía como ISO si se completó.

### Crear/editar desde el mapa

- El modal emite el objeto.
- **Crear**: `store.create(fields)` sin `id`.
- **Editar**: `diffInstalacion(current, item)` envía solo los campos cambiados:
  - Strings opcionales limpiados → `""`.
  - Numéricos opcionales limpiados → `null`.
- El store actualiza la lista local con la respuesta de la API.
- Eliminar pide confirmación y llama `store.remove(id)`.

## Cómo reutilizar Leaflet en otro proyecto

1. Instalar `leaflet` y `@types/leaflet`; importar
   `leaflet/dist/leaflet.css`.
2. Definir centro, zoom y `tileUrl` en `environment`.
3. Crear un contenedor con altura y `position: relative`.
4. En `ngAfterViewInit`, crear `L.map`, añadir `tileLayer` y un `layerGroup`.
5. Registrar `ResizeObserver` para `invalidateSize()` y limpiar en
   `ngOnDestroy` con `map.remove()`.
6. Usar `L.divIcon` con SVG si quieres iconos propios sin depender de los assets
   de Leaflet.
7. Escapar siempre el HTML de los popups.
8. Para producción, elegir un proveedor de tiles con cuota/contrato y mantener
   la atribución.

## Limitaciones

1. **Sin paginación**: se cargan todas las instalaciones; con miles de registros
   el render de marcadores y el DOM de la lista se degradan.
2. **Filtros en cliente**: no hay búsqueda ni filtros de servidor.
3. **Sin clustering**: muchos marcadores cercanos se solapan.
4. **Sin geocodificación**: no se busca por dirección; se usan coordenadas o
   clic en el mapa.
5. **Sin permiso específico**: cualquier usuario logueado puede ver y modificar
   instalaciones.
6. **Catálogos no validados en backend**: `tipo`, `estado` y `zona` aceptan
   cualquier string; un valor desconocido puede romper `ESTADO_CONFIG` en el
   mapa.
7. **Sin seed**: el mapa arranca vacío en modo BD.
8. **`fitBounds` inicial omitido**: la primera carga mantiene el centro
   configurado; el encuadre automático ocurre al cambiar filtros.
9. **Limpieza de `ultima_conexion`**: el frontend envía `""` y el backend lo
   guarda como string vacío, no como `NULL`.
10. **Sin control de concurrencia**: dos usuarios pueden sobrescribir cambios.

## Archivos involucrados

| Capa     | Archivo                                                            |
| -------- | ------------------------------------------------------------------ |
| Backend  | `back/api/router.go`                                               |
| Backend  | `back/api/handlers/instalacion.go`                                 |
| Backend  | `back/domain/instalacion.go`                                       |
| Backend  | `back/domain/inputs/instalacion.go`                                |
| Backend  | `back/infrastructure/database/instalaciones.go`                    |
| Frontend | `front/src/app/components/dashboard/mapas/mapa.ts` / `.html` / `.css` |
| Frontend | `front/src/app/components/dashboard/mapas/lista.ts` / `.html`      |
| Frontend | `front/src/app/components/dashboard/mapas/instalacion-modal.ts`    |
| Frontend | `front/src/app/services/instalaciones.service.ts`                  |
| Frontend | `front/src/app/services/instalaciones-store.service.ts`            |
| Frontend | `front/src/app/services/maps.service.ts`                           |
| Frontend | `front/src/app/models/instalacion.ts` / `mapa.ts`                  |
| Frontend | `front/src/environments/environment.ts`                            |
| Frontend | `front/src/styles.css`                                             |

## Ver también

- [Usuarios y permisos](./usuarios-permisos.md): el resto de módulos usa
  permisos; mapas no.
- [Mensajes pendientes y SSE](./mensajes-sse.md): comparación entre estado en
  vivo y consulta bajo demanda.
- [Buses](./buses.md): patrón de listado con filtros de servidor.
