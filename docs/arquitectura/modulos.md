# Anatomía de un módulo

Guía para agregar un módulo nuevo al template sin romper las convenciones de
capas, permisos, rutas, listados y UI. El ejemplo canónico es **Buses**:
`docs/flujos/buses.md` y `back/api/handlers/bus.go`.

> Los diagramas de `docs/diagramas/` son de flujo de negocio. Este documento es
> el mapa de implementación.

## Vertical slice

Un módulo atraviesa estas capas:

```
frontend (modelo, servicio, ruta, menú, list/modal)
        │  HTTP + cookie HttpOnly
        ▼
handler (validación HTTP, permisos, respuesta)
        │
        ▼
store (interfaz en back/store)
        │
        ▼
database (implementación SQL)
        │
        ▼
tabla(s) + seed de permisos + índice(s)
```

La regla es: el handler no escribe SQL y el store no conoce `http.ResponseWriter`.
Los errores de negocio se devuelven como `*domain.Error` y `writeError` los
traduce a `404`, `409`, `422`, etc.

## Backend

### 1. Modelo de dominio

Agrega la entidad en `back/domain/` o `back/domain/base.go`:

```go
type Ejemplo struct {
    ID        int64     `json:"id"`
    Nombre    string    `json:"nombre"`
    Activo    bool      `json:"activo"`
    CreadoEn  time.Time `json:"creado_en"`
    ActualizadoEn time.Time `json:"actualizado_en"`
}
```

- Usa `json:"-"` para campos sensibles.
- Si el campo es opcional, usa puntero y `omitempty`.
- No expongas el modelo directamente si necesitas recortar campos: crea un
  `outputs.EjemploResponse` y un `ToEjemploResponse`.

### 2. Inputs y outputs

- `back/domain/inputs/<entidad>.go`: `EjemploInput` para crear y
  `EjemploUpdateInput` con punteros para actualización parcial.
- `back/domain/outputs/<entidad>.go`: respuesta pública y mapper.
- Si hay listado paginado, reutiliza
  `outputs.PaginatedResponse[T]` (`data`, `offset`, `limit`, `has_more`).

### 3. Interfaz de store

Añade la interfaz en `back/store/store.go`:

```go
type EjemploStore interface {
    Count(filters EjemploFilters) int64
    List(offset, limit int, filters EjemploFilters) outputs.PaginatedResponse[outputs.EjemploResponse]
    FindByID(id int64) (*domain.Ejemplo, error)
    Create(input inputs.EjemploInput) (domain.Ejemplo, error)
    Update(id int64, input inputs.EjemploUpdateInput) (domain.Ejemplo, error)
    Delete(id int64) error
}
```

- `List` pide `limit+1` filas para calcular `has_more`; no ejecutes `COUNT(*)`
  dentro de `List`.
- `Count` es la única fuente del total.
- Devuelve `domain.NotFound(...)` / `domain.Conflict(...)` /
  `domain.Validation(...)`.
- Ordena de forma determinista (`ORDER BY id` o una columna única).

### 4. Implementación SQL

Crea `back/infrastructure/database/<entidad>.go`:

- Usa `s.db.rebind(...)` para portar `?` a PostgreSQL.
- Usa `translateDBError` para `sql.ErrNoRows` y violaciones de unicidad.
- Procesa listados y filtros con `LOWER(...) LIKE` si el filtro es textual.
- Si hay relación N:M, usa transacción y `replace...` como en usuarios.
- Si hay archivos, coordina con `Storage` y las carpetas pública/privada.

Añade la tabla en `back/infrastructure/database/schema.go` y, si corresponde,
el seed de permisos en `back/infrastructure/database/db.go`.

### 5. Handler

Crea `back/api/handlers/<entidad>.go`:

```go
type EjemploHandler struct{ deps *Dependencies }

func NewEjemploHandler(deps *Dependencies) *EjemploHandler { ... }

func (h *EjemploHandler) List(w http.ResponseWriter, r *http.Request) { ... }
```

Checklist del handler:

- [ ] Lee `offset`/`limit` con default y máximo (10/100 es la convención).
- [ ] Lee filtros con `strings.TrimSpace`.
- [ ] Valida `id` con `strconv.ParseInt` y responde `400 invalid id`.
- [ ] Usa `http.MaxBytesReader` para JSON (1 MiB es la convención).
- [ ] Devuelve `201` al crear y `200` al actualizar/leer.
- [ ] Usa `writeError(w, r, err)` para errores de negocio.
- [ ] No filtres permisos solo en el frontend: la ruta debe exigirlos.

### 6. Dependencias y router

- Añade el store a `handlers.Dependencies` (`back/api/handlers/deps.go`).
- Construye el store en `back/cmd/server/main.go` y pásalo a `Dependencies`.
- Registra el grupo en `back/api/router.go`:

```go
r.Route("/ejemplos", func(r chi.Router) {
    r.Use(middleware.Timeout(30 * time.Second))
    r.Use(authHandler.AuthMiddleware("ejemplos"))
    r.Get("/count", ejemploHandler.Count)
    r.Get("/", ejemploHandler.List)
    r.Post("/", ejemploHandler.Create)
    r.Get("/{id}", ejemploHandler.GetByID)
    r.Put("/{id}", ejemploHandler.Update)
    r.Delete("/{id}", ejemploHandler.Delete)
})
```

- Añade el permiso al seed si el módulo lo necesita.
- Si el módulo emite eventos, añade el canal en `sse_handler.go` y el broadcast
  después de cada mutación (ver `docs/flujos/mensajes-sse.md`).

### 7. Tests

- Handler: `httptest` + store falso; cubre `400`, `401`, `403`, `404`, `409`,
  `422`, `201`, `200`.
- Store: pruebas contra SQLite en memoria o PostgreSQL de test.
- Auth: si agregas claims o flags, actualiza `jwt_test.go`.
- Ejecuta `go build ./...`, `go vet ./...`, `go test ./...` y
  `govulncheck ./...`.

## Frontend

### 1. Modelo

Crea `front/src/app/models/<entidad>.ts` con la forma exacta de
`outputs.EjemploResponse` y los inputs de creación/edición.

### 2. Servicio

Crea `front/src/app/services/<entidad>.service.ts`:

```ts
count(filters): Observable<number>
list(offset, limit, filters): Observable<Paginated<Ejemplo>>
getByID(id): Observable<Ejemplo>
create(input): Observable<Ejemplo>
update(id, changes): Observable<Ejemplo>
delete(id): Observable<void>
```

- Construye query strings con `URLSearchParams`.
- No pongas `withCredentials`: lo añade `credentialsInterceptor`.
- No dupliques manejo de errores: `errorInterceptor` ya redirige en `401` y
  `403 password_change_required`.

### 3. Ruta y permiso

En `front/src/app/app.routes.ts`:

```ts
{
  path: 'ejemplos',
  canActivate: [permissionGuard],
  data: { permission: 'ejemplos' },
  loadComponent: () => import('./components/dashboard/ejemplos/list').then((m) => m.EjemplosListComponent),
}
```

- `authGuard` ya protege el padre `/dashboard`.
- `permissionGuard` redirige a `/dashboard` con toast si falta el permiso.
- La autorización real sigue en el backend.

### 4. Menú

Añade la entrada en
`front/src/app/components/dashboard/sidebar/menu.config.ts`:

```ts
{
  label: 'Ejemplos',
  key: 'ejemplos',
  path: '/dashboard/ejemplos',
  requiredPermission: 'ejemplos',
  group: 'Operación',
  description: '...',
  icon: { component: EjemploIcon, inputs: { size: '20' } },
}
```

El mismo `MenuService` filtra sidebar, horizontal y grid.

### 5. Listado

Copia el patrón de `buses/list.ts`:

- `loadWithCount()` con `forkJoin` de `count` + `list`.
- `load()` solo pide la página al cambiar de página.
- `search()` resetea a página 1 y vuelve a pedir count + página.
- `resetFilters()` limpia filtros.
- Eliminar con `ConfirmService` y recarga con `loadWithCount()`.
- Usa `PaginationComponent` compartido.

### 6. Modal

Copia el patrón de `buses/modal.ts`:

- Señales o campos para cada input.
- Defaults de creación.
- Validación de cliente para lo que el backend no valida (por ejemplo, precios
  negativos).
- **Crear** envía todo; **editar** envía solo el diff (`buildChanges`).
- Si hay imágenes, muestra `ImageManager` solo en edición y después de guardar.

### 7. Estados de UI

- `isLoading` para spinner.
- `error` local para errores de formulario.
- `ToastService` para éxito/error de listado.
- `ConfirmService` para borrados.
- No dupliques la redirección de `401`.

## Checklist de aceptación

- [ ] El permiso existe en el seed y en `menu.config.ts`.
- [ ] La ruta hija tiene `permissionGuard`.
- [ ] `GET /count` y `GET /` comparten filtros.
- [ ] El listado no ejecuta `COUNT(*)` y usa `has_more`.
- [ ] Create/Update devuelven la entidad actualizada.
- [ ] Los errores de negocio usan `domain.Error` + `writeError`.
- [ ] El frontend distingue crear de editar y envía diff.
- [ ] `go build`, `go vet`, `go test` y `npm run build` pasan.
- [ ] `node tools/check-docs.mjs` pasa si tocaste documentación.
- [ ] Si el módulo es un flujo nuevo, añade `docs/flujos/<flujo>.md` y su
      diagrama en `docs/diagramas/fuentes/`.

## Módulos que no son CRUD

- **SSE**: reutiliza `SSEHub`/`SSEService`; añade canal en `sseChannels` y
  `SSE_CHANNELS`, y un store que filtre `module/name`.
- **Archivos**: usa `Storage` y decide visibilidad/carpeta; revisa
  `docs/flujos/archivos.md`.
- **Reportes/documentos**: usa `services/reportes.go`,
  `services/documentos.go` y `JobStore`; revisa `docs/flujos/reportes.md`.
- **Mapas**: usa `InstalacionesStoreService` como patrón de estado en cliente;
  revisa `docs/flujos/instalaciones-mapas.md`.

## Referencias

- [Buses](../flujos/buses.md): CRUD paginado, filtros y diff.
- [Tienda](../flujos/tienda.md): relación categoría/producto e imágenes.
- [Usuarios y permisos](../flujos/usuarios-permisos.md): N:M, revocación y
  `permissionGuard`.
- [Mensajes y SSE](../flujos/mensajes-sse.md): estado compartido y canales.
- [Archivos](../flujos/archivos.md): público/privado y almacenamiento.
