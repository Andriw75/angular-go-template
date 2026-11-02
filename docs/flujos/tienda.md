# Flujo de tienda (categorías y productos)

## Resumen

Dos CRUD relacionados, ambos bajo el permiso `shop`:

- **Categorías**: agrupan productos y tienen imágenes.
- **Productos**: pertenecen a una categoría y tienen imágenes.

El menú **Tienda** agrupa `Categorias` y `Productos`. En modo base de datos no
hay seed: ambas tablas arrancan vacías.

## Modelo

### Categoría

`domain.Categoria`:

| Campo           | Tipo      |
| --------------- | --------- |
| `id`            | int64     |
| `nombre`        | string    |
| `descripcion`   | string    |
| `activo`        | bool      |
| `creado_en`     | time.Time |
| `actualizado_en`| time.Time |

### Producto

`domain.Producto`:

| Campo         | Tipo      | Notas                         |
| ------------- | --------- | ----------------------------- |
| `id`          | int64     |                               |
| `nombre`      | string    |                               |
| `descripcion` | string    |                               |
| `precio`      | float64   |                               |
| `stock`       | int       |                               |
| `categoria_id`| int64     | FK a `categorias(id)`         |
| `activo`      | bool      |                               |
| `creado_en`   | time.Time |                               |
| `actualizado_en` | time.Time |                            |

### Imágenes

Tabla `imagenes` con relación polimórfica:

- `tipo = "categoria"` + `entidad_id` = id de categoría.
- `tipo = "producto"` + `entidad_id` = id de producto.

Las imágenes dependen del estado de la entidad: si la categoría/producto está
activa son públicas y las sirve **nginx**; si está inactiva las sirve la API y
exigen permiso `shop`. El detalle de carpetas, subida, movimiento y borrado está
en [archivos.md](./archivos.md).

## Backend

### Rutas

Todas bajo `AuthMiddleware("shop")` (`back/api/router.go:81` y `:94`).

**Categorías**

| Método | Ruta                                  |
| ------ | ------------------------------------- |
| GET    | `/categorias/count`                   |
| GET    | `/categorias`                         |
| POST   | `/categorias`                         |
| GET    | `/categorias/{id}`                    |
| PUT    | `/categorias/{id}`                    |
| DELETE | `/categorias/{id}?cascade=true\|false`|
| POST   | `/categorias/{id}/imagenes`           |
| PUT    | `/categorias/{id}/imagenes/orden`     |
| DELETE | `/categorias/{id}/imagenes/{imagenID}`|

**Productos**

| Método | Ruta                                |
| ------ | ----------------------------------- |
| GET    | `/productos/count`                  |
| GET    | `/productos`                        |
| POST   | `/productos`                        |
| GET    | `/productos/{id}`                   |
| PUT    | `/productos/{id}`                   |
| DELETE | `/productos/{id}`                   |
| POST   | `/productos/{id}/imagenes`          |
| PUT    | `/productos/{id}/imagenes/orden`    |
| DELETE | `/productos/{id}/imagenes/{imagenID}`|

### Categorías

`CategoriaHandler` (`back/api/handlers/categoria.go`):

- **Count/List**: filtros `q` (nombre/descripción) y `activo`; paginación
  `offset`/`limit` (10 por defecto, máximo 100). Cada fila incluye sus imágenes
  (`ListByEntidad("categoria", id)`).
- **GetByID**: incluye imágenes.
- **Create**: exige `nombre`; `descripcion` y `activo` son opcionales.
- **Update**: parcial (`nombre`, `descripcion`, `activo`).
- **Delete**:
  - Busca los productos de la categoría.
  - Si hay productos y no llega `cascade=true` → `409`.
  - Con `cascade=true`: recolecta imágenes de productos y de la categoría,
    elimina registros de imágenes/productos/categoría y recién al final borra
    los archivos del disco.
  - Sin productos: elimina sus imágenes y la categoría.

### Productos

`ProductoHandler` (`back/api/handlers/producto.go`):

- **Count/List**: filtros `q` (nombre/descripción), `categoria_id` y `activo`;
  paginación igual que categorías. Cada fila incluye sus imágenes
  (`ListByEntidad("producto", id)`).
- **GetByID**: incluye imágenes.
- **Create**:
  - Exige `nombre`.
  - Exige `categoria_id > 0` y que la categoría exista → si no, `400`.
  - No valida precios/stock negativos en backend (sí lo hace el frontend).
- **Update**: parcial; si llega `categoria_id`, valida que exista.
- **Delete**: recolecta imágenes, elimina el producto, borra registros de
  imágenes y luego los archivos del disco.

### Errores

| Caso                                      | Respuesta                              |
| ----------------------------------------- | -------------------------------------- |
| Body inválido o falta `nombre`            | `400`                                  |
| `categoria_id` inválido/inexistente       | `400 invalid categoria_id`             |
| Sin permiso `shop`                        | `403`                                  |
| Categoría/producto inexistente            | `404`                                  |
| Borrar categoría con productos sin cascade| `409`                                  |
| Error interno                             | `500`                                  |

## Frontend

### Categorías

`CategoriasListComponent`:

- `loadWithCount()` con `forkJoin` count + página.
- Filtros `q` y `activo`; `search()` resetea a página 1.
- Al eliminar, primero consulta `GET /productos/count?categoria_id=...`:
  - Si hay productos, pide confirmación y llama `delete(id, true)`.
  - Si no, elimina normal.
- Recarga el listado tras crear/editar/eliminar.

`CategoriaModalComponent`:

- Campos `nombre`, `descripcion`, `activo`.
- Crear envía todo; editar envía solo los cambios.
- El `ImageManager` se muestra **solo al editar**, porque necesita el id ya
  creado.

### Productos

`ProductosListComponent`:

- `loadWithCount()` con `forkJoin`.
- Filtros `q`, `categoria_id` y `activo`.
- Carga el catálogo de categorías con `limit=100` para el select.
- Eliminar con confirmación y recarga.

`ProductoModalComponent`:

- Campos `nombre`, `descripcion`, `precio`, `stock`, `categoria_id`, `activo`.
- Validaciones de cliente: nombre obligatorio, categoría seleccionada, precio y
  stock no negativos.
- Crear envía todo; editar envía solo los cambios.
- El `ImageManager` se muestra **solo al editar**.

### ImageManager

`front/src/app/common/image-manager/image-manager.ts`:

- Lista imágenes del servidor + pendientes de subir.
- Marca imágenes para eliminar, permite reordenar por drag & drop.
- `applyImages()`:
  1. Elimina las marcadas.
  2. Sube las pendientes.
  3. Reordena con los ids definitivos.
  4. Refresca desde el servidor.
- Usa `MEDIA_URL` si `img.publico` (nginx) y `API_URL` si es restringida.

## Limitaciones

1. **Tablas vacías en modo BD**: el seed de categorías/productos existía solo en
   el antiguo mock.
2. **Cascade no transaccional**: la eliminación de una categoría con productos
   hace varias operaciones separadas. Si falla una, puede quedar estado parcial.
3. **N+1 de imágenes**: el listado pide imágenes por cada categoría/producto.
4. **Sin unicidad de nombre de categoría**: se permiten duplicados.
5. **Validación de negocio en frontend**: precio/stock negativos se bloquean en
   el modal, no en el backend.
6. **Imágenes solo después de crear**: no se pueden subir imágenes en el mismo
   formulario de creación; primero se guarda la entidad.
7. **Imágenes dependientes del estado**: no tienen visibilidad propia. Activas →
   nginx; inactivas → API y solo usuarios con `shop` (ver
   [archivos.md](./archivos.md)).
8. **Borrado irreversible**: el cascade elimina productos e imágenes sin
   papelera.
9. **Count y list separados**: posible desfase ante cambios concurrentes.

## Archivos involucrados

| Capa     | Archivo                                                         |
| -------- | --------------------------------------------------------------- |
| Backend  | `back/api/router.go`                                            |
| Backend  | `back/api/handlers/categoria.go`                                |
| Backend  | `back/api/handlers/producto.go`                                 |
| Backend  | `back/api/handlers/imagen.go`                                   |
| Backend  | `back/domain/base.go`                                           |
| Backend  | `back/infrastructure/database/categorias.go`                    |
| Backend  | `back/infrastructure/database/productos.go`                     |
| Backend  | `back/infrastructure/database/imagenes.go`                      |
| Frontend | `front/src/app/components/dashboard/shop/categorias/list.ts`    |
| Frontend | `front/src/app/components/dashboard/shop/categorias/modal.ts`   |
| Frontend | `front/src/app/components/dashboard/shop/productos/list.ts`     |
| Frontend | `front/src/app/components/dashboard/shop/productos/modal.ts`    |
| Frontend | `front/src/app/common/image-manager/image-manager.ts`           |
| Frontend | `front/src/app/services/categorias.service.ts`                  |
| Frontend | `front/src/app/services/productos.service.ts`                   |

## Ver también

- [Archivos e imágenes](./archivos.md): carpetas, visibilidad y servido de
  imágenes.
- [Usuarios y permisos](./usuarios-permisos.md): permiso `shop`.
- [Buses](./buses.md): mismo patrón de listado, filtros y paginación.
