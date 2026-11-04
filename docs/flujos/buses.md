# Flujo de buses

## Resumen

CRUD de buses con permisos, filtros y paginación de servidor. Es el flujo más
simple del dashboard y sirve de plantilla para los demás listados.

- Permiso requerido: `buses`.
- Entrada del menú: **Buses**.
- En modo base de datos **no hay seed de buses**: la tabla arranca vacía.

## Modelo

`domain.Bus` (`back/domain/base.go`):

| Campo                 | Tipo       | Notas                                   |
| --------------------- | ---------- | --------------------------------------- |
| `id`                  | int64      | PK                                      |
| `placa`               | string     |                                         |
| `nombre`              | string     |                                         |
| `marca`               | string     |                                         |
| `modelo`              | string     |                                         |
| `anio`                | int        |                                         |
| `capacidad`           | int        |                                         |
| `tipo`                | string     | `BUS`, `VAN`, `MINIBUS`, `MICROBUS`     |
| `activo`              | bool       |                                         |
| `fecha_compra`        | time.Time  | Se expone como `YYYY-MM-DD`             |
| `ultimo_mantenimiento`| *time.Time | Nullable; se expone como `YYYY-MM-DD`   |
| `precio`              | float64    |                                         |
| `peso`                | float64    |                                         |
| `color`               | string     |                                         |
| `descripcion`         | string     |                                         |
| `creado_en`           | time.Time  |                                         |
| `actualizado_en`      | time.Time  |                                         |

Tabla `buses` con las mismas columnas; `fecha_compra` y
`ultimo_mantenimiento` se guardan como texto de fecha.

## Backend

### Rutas

Todas bajo `AuthMiddleware("buses")` (`back/api/router.go:59`):

| Método | Ruta            | Descripción                  |
| ------ | --------------- | ---------------------------- |
| GET    | `/buses/count`  | Conteo con filtros           |
| GET    | `/buses`        | Listado paginado             |
| POST   | `/buses`        | Crear                        |
| GET    | `/buses/{id}`   | Obtener uno                  |
| PUT    | `/buses/{id}`   | Actualizar parcialmente      |
| DELETE | `/buses/{id}`   | Eliminar                     |

### Filtros y paginación

`BusHandler` (`back/api/handlers/bus.go`) y `database/buses.go`:

- `q`: busca en `placa`, `nombre`, `marca` y `modelo` con `LOWER(...) LIKE`.
- `tipo`: igualdad sin distinguir mayúsculas.
- `activo`: `true`/`false`.
- `count` devuelve un `int64` y es la única fuente del total.
- `list` acepta `offset` y `limit` (10 por defecto, máximo 100) y responde:

```json
{ "data": [ ... ], "offset": 0, "limit": 10, "has_more": true }
```

- `list` ya no ejecuta `COUNT(*)`: pide `limit+1` filas para calcular
  `has_more`. El front cachea el total de `/buses/count` y calcula las páginas.
- Orden fijo: `ORDER BY id` (determinista porque `id` es único).

### Crear

`POST /buses`:

- Obligatorios: `placa`, `nombre` y `fecha_compra` (formato `YYYY-MM-DD`).
- `ultimo_mantenimiento` es opcional y también usa `YYYY-MM-DD`.
- El handler construye el `domain.Bus` y el store asigna `creado_en` y
  `actualizado_en`.
- Responde `201` con `BusResponse`.

### Actualizar

`PUT /buses/{id}` con actualización parcial (punteros en `inputs.BusUpdateInput`):

- Solo se modifican los campos enviados.
- `ultimo_mantenimiento` acepta `null` para limpiarlo.
- `fecha_compra` y `ultimo_mantenimiento` se parsean como `YYYY-MM-DD`.
- Responde `200` con el bus actualizado.

### Eliminar

`DELETE /buses/{id}`: borrado físico. Si no existe → `404`.

### Errores

| Caso                                  | Respuesta                    |
| ------------------------------------- | ---------------------------- |
| Body inválido o fecha mal formada     | `400`                        |
| Sin sesión                            | `401`                        |
| Sin permiso `buses`                   | `403`                        |
| Bus inexistente en GET/DELETE         | `404 bus not found`          |
| Error interno                         | `500`                        |

## Frontend

### Lista

`BusesListComponent` (`front/src/app/components/dashboard/buses/list.ts`):

- `loadWithCount()` hace `forkJoin` de `GET /buses/count` y `GET /buses` para
  tener total y página en una sola pasada.
- `load()` solo pide la página (sin count) al cambiar de página.
- `search()` resetea a página 1 y vuelve a pedir count + página.
- `resetFilters()` limpia `q`, tipo y activo.
- Eliminar usa `ConfirmService` y recarga con `loadWithCount()`.

### Modal

`BusModalComponent`:

- Formulario con valores por defecto (`anio` actual, capacidad 20, tipo `BUS`,
  activo).
- Valida `placa`, `nombre` y `fecha_compra` en cliente.
- **Crear** envía el formulario completo.
- **Editar** envía solo los campos que cambiaron (`buildChanges`), incluyendo
  `ultimo_mantenimiento` como `null` si se limpió.

### Servicio

`BusesService` (`front/src/app/services/buses.service.ts`):

- `count`, `list`, `getByID`, `create`, `update`, `delete`.
- Construye query strings con `URLSearchParams`.

## Limitaciones

1. **Tabla vacía en modo BD**: el seed de 60 buses existía solo en el antiguo
   mock. Si clonas el template, el listado arranca vacío.
2. **Sin unicidad de `placa`**: el esquema no tiene `UNIQUE`; se permiten
   duplicados.
3. **Count y list separados**: son dos requests; si otro usuario crea/elimina
   entre ambas, el `total` y la página pueden no coincidir.
4. **Orden fijo por `id`**: no hay ordenamiento por columna.
5. **Borrado físico**: no hay soft delete ni papelera.
6. **Sin índices específicos** para los filtros; con muchos registros las
   búsquedas por `LIKE` pueden degradarse.

## Archivos involucrados

| Capa     | Archivo                                                     |
| -------- | ----------------------------------------------------------- |
| Backend  | `back/api/router.go`                                        |
| Backend  | `back/api/handlers/bus.go`                                  |
| Backend  | `back/domain/base.go`                                       |
| Backend  | `back/domain/inputs/bus.go`                                 |
| Backend  | `back/domain/outputs/bus.go`                                |
| Backend  | `back/infrastructure/database/buses.go`                     |
| Frontend | `front/src/app/components/dashboard/buses/list.ts`          |
| Frontend | `front/src/app/components/dashboard/buses/modal.ts`         |
| Frontend | `front/src/app/services/buses.service.ts`                   |
| Frontend | `front/src/app/models/bus.ts`                               |
| Frontend | `front/src/app/components/dashboard/sidebar/menu.config.ts` |

## Ver también

- [Usuarios y permisos](./usuarios-permisos.md): permiso `buses`.
- [Mensajes pendientes y SSE](./mensajes-sse.md): ejemplo del otro patrón de
  actualización (estado compartido en vivo vs consulta bajo demanda).
