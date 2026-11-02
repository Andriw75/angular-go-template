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
| `users`            | `id`, `username` (único), `email`, `password` (bcrypt), `activo` |
| `user_permissions` | N:M (`user_id`, `permission_id`), PK compuesta, FK con cascade   |

Permisos sembrados (`back/infrastructure/database/db.go`):

| Nombre                 | Descripción              |
| ---------------------- | ------------------------ |
| `buses`                | Acceso a buses           |
| `usuarios`             | Gestión de usuarios      |
| `mensajes_pendientes`  | Mensajes pendientes      |
| `shop`                 | Gestión de tienda        |
| `archivos`             | Gestión de archivos      |

El seed crea el usuario `admin` / `REDACTED_DEFAULT_ADMIN_PASSWORD` con todos los permisos **solo
si la tabla `users` está vacía**. Los permisos se aseguran siempre con
`ON CONFLICT (nombre) DO NOTHING`.

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

Todas se registran bajo `AuthMiddleware("usuarios")`
(`back/api/router.go:45` y `back/api/router.go:54`). No existe CRUD de permisos:
para agregar uno hay que modificar el seed y reiniciar.

### Handlers

`back/api/handlers/user.go`:

- **List**: devuelve `[]UserResponse` sin contraseña.
- **GetByID**: `404 user not found` si no existe.
- **Create**: exige `username`, `email` y `password`; el resto es opcional.
  Hashea la contraseña y asigna los permisos por nombre.
- **Update**: actualización parcial con punteros (`username`, `email`,
  `password`, `activo`, `permisos`). La contraseña solo se cambia si llega no
  vacía.
  - Si el update incluye `permisos` o `activo`, ejecuta
    `JWTStore.RemoveUserTokens(id)` para forzar la resincronización del token
    (ver [logout.md](./logout.md)).
  - Un `id` inexistente termina en `500 failed to update user` (el store
    devuelve `sql.ErrNoRows` y el handler lo trata como error interno).
- **Delete**: impide eliminarse a uno mismo (`claims.UserID == id` →
  `403 cannot delete yourself`). El resto devuelve `404` si no existe.

`back/api/handlers/permiso.go`:

- **List**: devuelve el catálogo completo (`id`, `nombre`, `descripcion`).

### Persistencia

`back/infrastructure/database/users.go`:

- `Create` (`users.go:23`): inserta el usuario, hashea la contraseña y llama a
  `replacePermisos`.
- `Update` (`users.go:62`): carga el usuario, aplica los campos enviados y
  reemplaza permisos si `permisos` no es `nil`.
- `replacePermisos` (`users.go:214`): borra las asignaciones previas e inserta
  cada permiso con `INSERT ... SELECT id FROM permissions WHERE nombre = ?`.
  Si un nombre no existe, la inserción afecta 0 filas y devuelve
  `permiso no encontrado`.
- `loadPermisos` (`users.go:194`): join `user_permissions` + `permissions`,
  ordenado por nombre. Ese es el orden que termina en el JWT.
- `IsValidPassword` (`users.go:152`): compara con bcrypt; lo usa el login.
- `FindAll` (`users.go:132`): carga cada usuario y sus permisos.

### Permisos en el JWT

Al hacer login, `JWTManager.Generate` incluye `user.Permisos` como claim. Es una
**foto del momento**: si luego cambian, el middleware debe refrescar el token
(ver [logout.md](./logout.md)).

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
- **Crear**: envía `username`, `email`, `password`, `activo` y `permisos`.
- **Editar**: envía solo los campos que cambiaron (diff contra los originales).
  `permisos` solo se envía si el array cambió; `activo` solo si cambió; la
  contraseña solo si se escribió.
- `isSelf`: si el usuario editado es uno mismo, deshabilita los checkboxes de
  permisos y muestra un aviso. Es una restricción **de UI**; el backend no la
  impone, así que una llamada directa a `PUT /users/{id}` sí podría cambiarlos.
- Al guardar, el padre recarga la lista.

### Sidebar y permisos

`front/src/app/components/dashboard/sidebar/menu.config.ts` define
`requiredPermission` por entrada. `SidebarComponent.filterMenu`
(`sidebar.ts:49`) filtra el menú con `auth.user()?.permisos`.

Importante: esto es solo **visibilidad de UI**. Las rutas hijas del dashboard no
tienen guards por permiso (`app.routes.ts` solo aplica `authGuard` al padre). Un
usuario sin permiso puede navegar directo a `/dashboard/usuarios`; el componente
se instancia, pero `GET /users` responde `403` y se muestra un toast de error.
La autorización real siempre está en el backend.

## Permiso → módulo

| Permiso               | Endpoints                                        | Menú          |
| --------------------- | ------------------------------------------------ | ------------- |
| `buses`               | `/buses`                                         | Buses         |
| `usuarios`            | `/users`, `/permisos`                            | Usuarios      |
| `mensajes_pendientes` | `/mensajes_pendientes`                           | Mensajes      |
| `shop`                | `/categorias`, `/productos`                      | Tienda        |
| `archivos`            | `/archivos` (subida, listado y administración)   | Archivos      |

`/archivos/{filename}` no exige permiso: usa `OptionalAuthMiddleware` y valida
la visibilidad del archivo.

## Errores esperados

| Caso                                  | Respuesta                          |
| ------------------------------------- | ---------------------------------- |
| Body inválido o faltan campos          | `400`                              |
| Sin sesión                             | `401 not authenticated`            |
| Sin permiso `usuarios`                 | `403 forbidden`                    |
| Intento de eliminarse a uno mismo      | `403 cannot delete yourself`       |
| Usuario inexistente en GET/DELETE      | `404 user not found`               |
| Usuario inexistente en UPDATE          | `500 failed to update user`        |
| `username` duplicado                   | `500 failed to create/update user` |
| Permiso inexistente al asignar         | `500 failed to create/update user` |

## Limitaciones conocidas

1. **Eliminar un usuario no revoca sus tokens**: `UserHandler.Delete` no llama a
   `RemoveUserTokens`. El JWT del usuario eliminado sigue siendo válido hasta
   expirar (≤ 15 min). Para corte inmediato habría que revocar sus tokens en el
   delete.
2. **Autoprotección parcial**: el backend impide eliminarse a uno mismo, pero
   **no** impide que un administrador cambie sus propios permisos o su `activo`.
   La UI deshabilita los checkboxes de permisos cuando `isSelf`, pero una
   llamada directa a la API sí puede hacerlo.
3. **Sin CRUD de permisos**: el catálogo es fijo (seed). Agregar un permiso
   requiere tocar código y reiniciar.
4. **Permisos de UI, no de ruta**: el sidebar oculta entradas sin permiso, pero
   las rutas hijas del dashboard no tienen guards por permiso. La autorización
   real está en el backend (`403`).
5. **Conflictos devuelven 500**: `username` duplicado o un permiso inexistente
   se reportan como `500`, no como `409`/`422`.

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

## Ver también

- [Login](./login.md): emisión del JWT y claims de permisos.
- [Logout y revocación](./logout.md): qué pasa cuando cambian los permisos de
  una sesión activa y cómo se resincroniza el token.
- [Mensajes pendientes y SSE](./mensajes-sse.md): usa el permiso
  `mensajes_pendientes`.
