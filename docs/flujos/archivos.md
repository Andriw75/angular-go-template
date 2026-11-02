# Flujo de archivos e imágenes

## Resumen

El proyecto separa **público** y **restringido**:

| Recurso                         | Público                              | Restringido                                  |
| ------------------------------- | ------------------------------------ | -------------------------------------------- |
| Imágenes de entidades           | Entidad activa → **nginx**           | Entidad inactiva → **API** (permiso `shop`)  |
| Archivos genéricos              | `visibilidad=publico` → **nginx**    | `autenticado`/`propietario` → **API**        |

La regla es: **si es público, lo sirve nginx sin pasar por la API**; si tiene
cualquier restricción, lo sirve la API. Para lograrlo, el backend mueve los
archivos entre carpetas públicas y privadas cuando cambia la visibilidad o el
estado de la entidad.

## Mapa de carpetas

Con `PUBLIC_DIR=C:\Andriw 2\nginx-1.28.0\html\template`:

```
...\html\
└── template\                         ← raíz de nginx (SPA)
    ├── index.html, assets...
    ├── public\                       ← nginx sirve /template/public/
    │   ├── imagenes\                 ← imágenes de entidades activas
    │   └── archivos\                 ← archivos con visibilidad=publico
    └── archivos\                     ← nginx bloquea /template/archivos
        ├── imagenes\                 ← imágenes de entidades inactivas
        └── ...                       ← archivos autenticado/propietario
```

Rutas calculadas en `back/services/config.go`:

- `PublicPath()` = `PUBLIC_DIR/public`
- `PublicImagesDir()` = `PUBLIC_DIR/public/imagenes`
- `PrivateImagesDir()` = `PUBLIC_DIR/archivos/imagenes`
- `PublicFilesDir()` = `PUBLIC_DIR/public/archivos`
- `PrivateFilesDir()` = `PUBLIC_DIR/archivos`

nginx (`deploy/nginx.conf`):

- `/template/public/` → alias a `html/template/public/` (imágenes y archivos
  públicos).
- `/template/archivos` → `return 404` (bloquea la carpeta privada; solo Go la
  sirve).
- `/template/api/` → proxy a Go en `127.0.0.1:8080`.
- Rutas públicas: `MEDIA_URL + /imagenes/...` y `MEDIA_URL + /archivos/...`.
- Rutas restringidas: `API_URL + /imagenes/...` y `API_URL + /archivos/...`.

## Respuestas con `publico`

La API incluye un booleano `publico` en las respuestas:

- `ImagenResponse.publico`
- `ArchivoResponse.publico`

El frontend usa ese flag para decidir la base de la URL:

```ts
const base = publico ? environment.MEDIA_URL : environment.API_URL;
```

Así el navegador pide las imágenes/archivos públicos directamente a nginx y
solo los restringidos pasan por la API.

## Imágenes de entidades

### Regla de publicación

`ImageIsPublic` (`back/api/handlers/imagen.go`):

- Imagen de **producto**: producto activo **y** categoría activa.
- Imagen de **categoría**: categoría activa.

Si no se cumple, la imagen es restringida y requiere sesión con permiso `shop`.

### Rutas

- `GET /imagenes/{filename}` con `OptionalAuthMiddleware`:
  - Pública → se sirve sin login (nginx es la vía normal; la API queda como
    respaldo).
  - Restringida → exige `shop`.
- Gestión bajo `AuthMiddleware("shop")`:
  - `POST /categorias/{id}/imagenes`
  - `PUT /categorias/{id}/imagenes/orden`
  - `DELETE /categorias/{id}/imagenes/{imagenID}`
  - Equivalentes para `/productos`.

### Subida

`uploadImagesForEntity`:

1. Determina `publico` con `ImageIsPublic`.
2. Elige `PublicImagesDir` o `PrivateImagesDir`.
3. Guarda con `Storage.SaveFilesIn`.
4. Inserta en `imagenes` y responde con `publico`.

### Servido

- Pública: nginx desde `/template/public/imagenes/{filename}`.
- Restringida: API `GET /imagenes/{filename}` con cookie y permiso `shop`.
- Caché:
  - nginx: `expires 1h`.
  - API pública (respaldo): `public, max-age=60`.
  - API restringida: `private, no-store`.

### Cambios de estado

Cuando cambia el estado de una entidad, sus imágenes se mueven de carpeta:

- `ProductoHandler.Update` → `EnsureImagePlacement("producto", id)`.
- `CategoriaHandler.Update` → `EnsureCategoryImagesPlacement(id)`, que mueve la
  categoría y todos sus productos (porque la categoría afecta la publicación de
  los productos).
- Al arrancar, `syncPlacement` recorre la BD y ubica cada imagen en la carpeta
  correcta.

### Borrado

- `deleteImageForEntity` elimina el registro y borra el archivo de **ambas**
  carpetas (defensivo).
- Al borrar categoría/producto se eliminan sus imágenes de ambas carpetas.

## Archivos genéricos

### Visibilidad

| Visibilidad   | Quién puede leer                    | Servidor |
| ------------- | ----------------------------------- | -------- |
| `publico`     | Cualquiera con la URL               | nginx    |
| `autenticado` | Cualquier usuario con JWT válido    | API      |
| `propietario` | Solo el usuario que lo subió        | API      |

> No existe lectura única ni enlaces de un solo uso. `propietario` significa
> solo el dueño, con lecturas ilimitadas.

### Rutas y permisos

| Método | Ruta                         | Permiso     | Regla extra          |
| ------ | ---------------------------- | ----------- | -------------------- |
| GET    | `/archivos`                  | `archivos`  | Lista metadata       |
| POST   | `/archivos`                  | `archivos`  | Sube uno o varios    |
| GET    | `/archivos/{filename}`       | —           | `OptionalAuth`       |
| PUT    | `/archivos/{id}/visibilidad` | `archivos`  | Solo el propietario  |
| DELETE | `/archivos/{id}`             | `archivos`  | Solo el propietario  |

### Subida

`ArchivoHandler.Upload`:

1. `ParseMultipartForm(100 MB)`.
2. Lee `files` y `visibilidad`.
3. Valida extensiones permitidas.
4. Elige `PublicFilesDir` si `publico`; si no, `PrivateFilesDir`.
5. Inserta en `archivos` con `user_id`.
6. Responde con `url` y `publico`.

### Listado

- Paginación `offset`/`limit` y búsqueda `q`.
- Devuelve metadata de todos los archivos con `publico`.
- `ArchivoResponse` no incluye `user_id`.

### Servido

`GET /archivos/{filename}`:

1. Busca el registro.
2. Aplica la regla de visibilidad (`publico`, `autenticado`, `propietario`).
3. Público: nginx desde `/template/public/archivos/{filename}`.
4. Restringido: API desde `PrivateFilesDir`.
5. Si por una transición fallida el archivo está en la otra carpeta, se mueve a
   la correcta antes de servirlo.
6. Cache:
   - Público (API de respaldo): `public, max-age=86400`.
   - Restringido: `private, no-store`.

### Cambio de visibilidad

`PUT /archivos/{id}/visibilidad`:

- Detecta el cambio público ↔ restringido.
- Mueve el archivo entre `PrivateFilesDir` y `PublicFilesDir`.
- Si la actualización en BD falla, intenta moverlo de vuelta.

### Borrado

- Elimina el registro y el archivo de **ambas** carpetas.

## Vista unificada (`/dashboard/archivos`)

La página **Archivos e imágenes** tiene dos pestañas:

- **Archivos**: gestor genérico (subida, visibilidad, descarga, borrado).
- **Imágenes**: galería de imágenes de categorías/productos. Muestra el nombre
  de la entidad, si la imagen es pública o restringida, y permite copiar el
  link, abrir o eliminar. Solo aparece con permiso `shop`.

La subida y el reordenamiento de imágenes de una entidad siguen en el modal de
categoría/producto (`ImageManager`).

## Frontend

- `ImageManager` usa `MEDIA_URL` si `img.publico`, y `API_URL` si no.
- `ArchivosService.getURL(archivo)` elige la base con `archivo.publico`.
- `ArchivosService.getImagenURL(imagen)` elige la base con `imagen.publico`.
- La galería de la pestaña Imágenes usa `getImagenURL`.

## Seguridad y cache

1. **Archivos/imágenes públicas**: nginx, sin coste de API.
2. **Restringidos**: API, con validación de visibilidad/estado/permiso.
3. **Movimiento de archivos**: al cambiar visibilidad o estado, y al arrancar
   (`syncPlacement`). Si un movimiento falla, el handler de servido intenta
   corregirlo en el siguiente acceso.
4. **Caché de públicos**: nginx cachea 1 h; los archivos públicos servidos por
   la API usan 24 h. Cambiar a restringido no invalida copias ya cacheadas en el
   navegador.
5. **`OptionalAuthMiddleware`** solo valida firma/expiración; no consulta el
   `JWTStore` ni `activo`. Un token vigente pero revocado puede leer archivos
   restringidos hasta expirar.
6. **`user_id` sin FK**: al borrar un usuario, sus archivos quedan huérfanos.

## Limitaciones

1. **Movimiento de archivos**: depende de que la transición de estado/visibilidad
   se haga por la API. Cambios directos en BD no mueven los archivos; el arranque
   (`syncPlacement`) los corrige.
2. **Caché pública**: una imagen/archivo público ya cacheado puede seguir
   viéndose hasta que expire la cache aunque pase a restringido.
3. **Multi-instancia**: el almacenamiento es local. Con varias instancias hace
   falta disco compartido.
4. **Sin lectura única ni enlaces firmados**.
5. **Colisión de nombres en archivos**: `archivo_{i}_{nombre}` puede repetirse.
6. **Rollback incompleto** en subidas múltiples de archivos genéricos.
7. **Límite de tamaño**: 100 MB es memoria de formulario, no límite duro de
   request.
8. **Sin validación de contenido**: solo extensión.
9. **Lista global de archivos**: cualquier usuario con `archivos` ve metadata de
   todos.
10. **Sin override de admin**: solo el dueño cambia visibilidad o borra archivos.

## Archivos involucrados

| Capa     | Archivo                                                     |
| -------- | ----------------------------------------------------------- |
| Backend  | `back/services/config.go`                                   |
| Backend  | `back/api/router.go`                                        |
| Backend  | `back/api/handlers/archivo.go`                              |
| Backend  | `back/api/handlers/imagen.go`                               |
| Backend  | `back/api/handlers/categoria.go`                            |
| Backend  | `back/api/handlers/producto.go`                             |
| Backend  | `back/store/store.go`                                       |
| Backend  | `back/infrastructure/storage/storage.go`                    |
| Backend  | `back/infrastructure/database/archivos.go`                  |
| Backend  | `back/infrastructure/database/imagenes.go`                  |
| Backend  | `back/cmd/server/main.go`                                   |
| Frontend | `front/src/app/services/archivos.service.ts`                |
| Frontend | `front/src/app/common/image-manager/image-manager.ts`       |
| Frontend | `front/src/app/components/dashboard/archivos/list.ts`       |
| Frontend | `front/src/app/components/dashboard/archivos/modal.ts`      |
| Deploy   | `deploy/nginx.conf`                                         |

## Variables de entorno

```
PUBLIC_DIR=C:\Andriw 2\nginx-1.28.0\html\template
X_ACCEL_REDIRECT_PREFIX=
```

- `PUBLIC_DIR` llega hasta `template`; el código deriva `public/imagenes`,
  `public/archivos`, `archivos/imagenes` y `archivos`.
- `MEDIA_URL` se usa para públicos; `API_URL` para restringidos.
- `X_ACCEL_REDIRECT_PREFIX` vacío = Go sirve con `http.ServeFile`.

## Ver también

- [Login](./login.md): cookie HttpOnly y autenticación de recursos restringidos.
- [Logout y revocación](./logout.md): `OptionalAuthMiddleware` y revocación.
- [Tienda (categorías/productos)](./tienda.md): estado activo de las entidades.
- [Usuarios y permisos](./usuarios-permisos.md): permisos `archivos` y `shop`.
