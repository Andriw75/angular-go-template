# Flujo de usuarios y permisos

## Resumen

El sistema usa permisos planos (sin roles ni jerarquías). El catálogo de permisos
es fijo y se siembra al arrancar. Cada usuario tiene una lista de permisos
asignados (relación N:M). Esa lista viaja dentro del JWT y se usa para:

- Autorizar endpoints con `AuthMiddleware("permiso")`.
- Mostrar u ocultar entradas del sidebar.

Solo un usuario con el permiso `usuarios` puede gestionar usuarios y consultar
el catálogo de permisos.

## Modelo de datos

| Tabla              | Contenido                                                        |
| ------------------ | ---------------------------------------------------------------- |
| `permissions`      | Catálogo fijo: `id`, `nombre` (único), `descripcion`             |
| `users`            | `id`, `username` (único), `email`, `password` (bcrypt), `activo`, `must_change_password` |
| `user_permissions` | N:M (`user_id`, `permission_id`), PK compuesta, FK con cascade   |

Permisos sembrados (`back/infrastructure/database/db.go`):

| Nombre                 | Descripción                    |
| ---------------------- | ------------------------------ |
| `buses`                | Acceso a buses                 |
| `usuarios`             | Gestión de usuarios            |
| `mensajes_pendientes`  | Mensajes pendientes            |
| `shop`                 | Gestión de tienda              |
| `archivos`             | Gestión de archivos            |
| `ventas`               | Gestión de ventas              |
| `reportes`             | Generación de reportes         |
| `documentos`           | Gestión de documentos generados |

El seed **no crea usuarios por defecto**. Si la tabla `users` está vacía, el
primer administrador se crea solo cuando `BOOTSTRAP_ADMIN_PASSWORD` está
definido; recibe todos los permisos y debe cambiar la contraseña en el primer
login. Los permisos se aseguran siempre con `ON CONFLICT (nombre) DO NOTHING`.

## Backend

### Rutas

| Método | Ruta           | Permiso requerido |
| ------ | -------------- | ----------------- |
| GET    | `/users`       | `usuarios`        |
| POST   | `/users`       | `usuarios`        |
| GET    | `/users/{id}`  | `usuarios`        |
| PUT    | `/users/{id}`  | `usuarios`        |
| DELETE | `/users/{id}`  | `usuarios`        |
| GET    | `/permisos`    | `usuarios`        |

Todas se registran bajo `AuthMiddleware("usuarios")` en
`back/api/router.go`. No existe CRUD de permisos: para agregar uno hay que
modificar el seed y reiniciar.

### Handlers

`back/api/handlers/user.go`:

- **List**: devuelve `[]UserResponse` sin contraseña.
- **GetByID**: `404 user not found` si no existe.
- **Create**: exige `username`, `email` y `password`; el resto es opcional.
  Aplica la política de contraseñas (12+ caracteres, no común, distinta al
  username), hashea la contraseña, asigna los permisos por nombre y fuerza
  `must_change_password=true`.
- **Update**: actualización parcial con punteros (`username`, `email`,
  `password`, `activo`, `permisos`, `must_change_password`). La contraseña solo
  se cambia si llega no vacía; un reset de admin vuelve a activar
  `must_change_password`.
  - Si el update incluye `permisos`, `activo` o una contraseña nueva, ejecuta
    `JWTStore.RemoveUserTokens(id)`. La siguiente petición del afectado recibe
    `401 session_expired` y debe volver a iniciar sesión (ver
    [logout.md](./logout.md) y [cambio-password.md](./cambio-password.md)).
  - Un `id` inexistente responde `404 not_found`; el store devuelve un error de
    negocio y `writeError` lo traduce.
- **Delete**: impide eliminarse a uno mismo (`claims.UserID == id` →
  `403 cannot delete yourself`). El resto devuelve `404` si no existe y revoca
  los tokens del usuario eliminado.

`back/api/handlers/permiso.go`:

- **List**: devuelve el catálogo completo (`id`, `nombre`, `descripcion`).

### Persistencia

`back/infrastructure/database/users.go`:

- `Create`: inserta el usuario, hashea la contraseña y llama a
  `replacePermisos`.
- `Update`: carga el usuario, aplica los campos enviados, activa
  `must_change_password` si llega una contraseña nueva y reemplaza permisos si
  `permisos` no es `nil`.
- `replacePermisos`: borra las asignaciones previas e inserta cada permiso con
  `INSERT ... SELECT id FROM permissions WHERE nombre = ?`. Si un nombre no
  existe, la inserción afecta 0 filas y devuelve `permiso no encontrado`.
- `loadPermisos`: join `user_permissions` + `permissions`, ordenado por nombre.
  Ese es el orden que termina en el JWT.
- `IsValidPassword`: compara con bcrypt; lo usa el login.
- `FindAll`: carga cada usuario y sus permisos.

### Permisos en el JWT

Al hacer login, `JWTManager.Generate` incluye `user.Permisos` como claim. Es una
**foto del momento**: si luego cambian por API, se revocan los tokens del
usuario y el siguiente request exige re-login. En la renovación deslizante,
`JWT_LOAD_USER_ON_RENEW=true` recarga permisos desde la BD (ver
[logout.md](./logout.md)).

`AuthMiddleware(requiredPermission...)` (`back/api/handlers/auth.go`) valida el
token y, si la ruta exige permiso, usa `tienePermiso(claims.Permisos, required)`.
Sin el permiso responde `403 forbidden`.

## Frontend

### Lista de usuarios

`front/src/app/components/dashboard/usuarios/list.ts`:

- Carga `GET /users` en el constructor.
- Muestra usuario, email, badges de permisos y estado activo.
- Abre el modal para crear o editar.
- Elimina con confirmación y bloquea la eliminación de uno mismo en cliente.

### Modal

`front/src/app/components/dashboard/usuarios/modal.ts`:

- Carga el catálogo con `GET /permisos`.
- **Crear**: envía `username`, `email`, `password`, `activo` y `permisos`. El
  backend fuerza el cambio de contraseña en el primer acceso.
- **Editar**: envía solo los campos que cambiaron (diff contra los originales).
  `permisos` solo se envía si el array cambió; `activo` solo si cambió; la
  contraseña solo si se escribió. Un reset de contraseña vuelve a activar el
  cambio obligatorio.
- `isSelf`: si el usuario editado es uno mismo, deshabilita los checkboxes de
  permisos y muestra un aviso. Es una restricción **de UI**; el backend no la
  impone, así que una llamada directa a `PUT /users/{id}` sí podría cambiarlos.
- Al guardar, el padre recarga la lista.

### Sidebar y permisos

`front/src/app/components/dashboard/sidebar/menu.config.ts` define
`requiredPermission` por entrada. `MenuService`
(`front/src/app/services/menu.service.ts`) filtra recursivamente el menú con
`auth.user()?.permisos`; el sidebar, el layout horizontal y la grilla comparten
ese servicio.

Las rutas hijas del dashboard aplican `permissionGuard` con
`route.data.permission` (`front/src/app/app.routes.ts`). Si falta el permiso, el
guard redirige a `/dashboard` y muestra un toast. El sidebar además oculta la
entrada. Aun así, la autorización real sigue estando en el backend (`403`).

## Permiso → módulo

| Permiso               | Endpoints                                        | Menú                    |
| --------------------- | ------------------------------------------------ | ----------------------- |
| `buses`               | `/buses`                                         | Buses                   |
| `usuarios`            | `/users`, `/permisos`                            | Usuarios                |
| `mensajes_pendientes` | `/mensajes_pendientes`, `/sse/events` (canal)     | Mensajes                |
| `shop`                | `/categorias`, `/productos`, imágenes            | Tienda → Catálogo       |
| `archivos`            | `/archivos` (subida, listado y administración)   | Archivos                |
| `ventas`              | `/ventas`, `/reportes/ventas*`                    | Tienda → Operaciones    |
| `reportes`            | `/reportes/*`                                    | Reportes                |
| `documentos`          | `/documentos/*`                                  | Reportes (historial)    |

`/archivos/{filename}` no exige permiso: usa `OptionalAuthMiddleware` y valida
la visibilidad del archivo.

## Errores esperados

| Caso                                  | Respuesta                          |
| ------------------------------------- | ---------------------------------- |
| JSON inválido o `id` no numérico        | `400`                              |
| Faltan campos o contraseña inválida    | `422 validation_error`             |
| Sin sesión                             | `401 not authenticated`            |
| Sin permiso `usuarios`                 | `403 forbidden`                    |
| Intento de eliminarse a uno mismo      | `403 cannot delete yourself`       |
| Usuario inexistente en GET/DELETE      | `404 user not found`               |
| Usuario inexistente en UPDATE          | `404 not_found`                    |
| `username` duplicado                   | `409 conflict`                     |
| Permiso inexistente al asignar         | `422 validation_error`             |

## Limitaciones conocidas

1. **Eliminar un usuario revoca sus tokens**: `UserHandler.Delete` llama a
   `RemoveUserTokens` tras borrar la fila. Con `JWT_STORE_ENABLED=false` no hay
   revocación y el JWT sigue válido hasta expirar (≤ 15 min).
2. **Autoprotección parcial**: el backend impide eliminarse a uno mismo, pero
   **no** impide que un administrador cambie sus propios permisos o su `activo`.
   La UI deshabilita los checkboxes de permisos cuando `isSelf`, pero una
   llamada directa a la API sí puede hacerlo.
3. **Sin CRUD de permisos**: el catálogo es fijo (seed). Agregar un permiso
   requiere tocar código y reiniciar.
4. **Permisos de UI y de ruta**: el sidebar oculta entradas sin permiso y las
   rutas hijas aplican `permissionGuard` (`route.data.permission`). Aun así, la
   autorización real sigue estando en el backend (`403`).
5. **Conflictos y validación tipados**: `username` duplicado responde `409`
   (`conflict`); un permiso inexistente al asignar responde `422`
   (`validation_error`); un `id` inexistente en update/delete responde `404`.

## Archivos involucrados

| Capa     | Archivo                                                     |
| -------- | ----------------------------------------------------------- |
| Backend  | `back/api/router.go`                                        |
| Backend  | `back/api/handlers/user.go`                                 |
| Backend  | `back/api/handlers/permiso.go`                              |
| Backend  | `back/domain/base.go`                                       |
| Backend  | `back/domain/inputs/user.go`                                |
| Backend  | `back/domain/outputs/auth.go`                               |
| Backend  | `back/infrastructure/database/users.go`                     |
| Backend  | `back/infrastructure/database/permisos.go`                  |
| Backend  | `back/infrastructure/database/db.go`                        |
| Backend  | `back/infrastructure/auth/jwt_store.go`                     |
| Frontend | `front/src/app/components/dashboard/usuarios/list.ts`       |
| Frontend | `front/src/app/components/dashboard/usuarios/modal.ts`      |
| Frontend | `front/src/app/services/users.service.ts`                   |
| Frontend | `front/src/app/models/auth.ts`                              |
| Frontend | `front/src/app/components/dashboard/sidebar/menu.config.ts` |
| Frontend | `front/src/app/components/dashboard/sidebar/sidebar.ts`     |
| Frontend | `front/src/app/services/menu.service.ts`                    |
| Frontend | `front/src/app/guards/permission.guard.ts`                  |
| Frontend | `front/src/app/app.routes.ts`                               |

## Ver también

- [Login](./login.md): emisión del JWT y claims de permisos.
- [Cambio obligatorio de contraseña](./cambio-password.md): alta, reset y
  bloqueo por `must_change_password`.
- [Logout y revocación](./logout.md): qué pasa cuando cambian los permisos de
  una sesión activa (revocación y re-login).
- [Mensajes pendientes y SSE](./mensajes-sse.md): usa el permiso
  `mensajes_pendientes`.
