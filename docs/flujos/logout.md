# Flujo de logout y revocación

## Resumen

El usuario pulsa “Cerrar sesión” en el sidebar. El frontend cierra la conexión
SSE, llama a `POST /auth/logout`, limpia el usuario en memoria y navega a
`/login`. El backend elimina el `jti` del token del `JWTStore` (si está activo)
y borra la cookie de sesión.

## Secuencia

```
Usuario          SidebarComponent       AuthService              API Go
  |                    |                    |                      |
  |-- Cerrar sesión -->|                    |                      |
  |                    |-- disconnect SSE   |                      |
  |                    |-- logout() ------->|                      |
  |                    |                    |-- POST /auth/logout ->|
  |                    |                    |   (cookie HttpOnly)  |
  |                    |                    |                      |-- AuthMiddleware
  |                    |                    |                      |-- JWTStore.Remove(jti)
  |                    |                    |<-- 200 + cookie="" --|
  |                    |-- user.set(null)   |                      |
  |                    |-- navigate(/login) |                      |
```

## Backend

1. Ruta: `POST /auth/logout` dentro del grupo con `AuthMiddleware()`
   (`back/api/router.go:40`). Requiere una cookie válida.
2. Handler `AuthHandler.Logout` (`back/api/handlers/auth.go`):
   - Obtiene los claims del contexto.
   - Si `JWT_STORE_ENABLED` y el token tiene `jti`, ejecuta
     `JWTStore.Remove(claims.ID)`.
   - Borra la cookie con el mismo `Name`, `Path`, `HttpOnly`, `Secure` y
     `SameSite`, pero `Value: ""` y `MaxAge: -1`.
   - Responde `200 {"message":"logged out"}`.
3. `JWTStore` (`back/infrastructure/auth/jwt_store.go`):
   - Mantiene en memoria `jti -> userID` y `userID -> {jti...}`.
   - `Remove(jti)` (`jwt_store.go:30`): revoca una sesión concreta (logout).
   - `RemoveUserTokens(userID)` (`jwt_store.go:42`): revoca todas las sesiones.
   - `Exists(jti)` (`jwt_store.go:53`): lo consulta `AuthMiddleware` en cada
     request autenticada.

## Revocación por actualización de usuario

`UserHandler.Update` (`back/api/handlers/user.go:90`) llama a
`JWTStore.RemoveUserTokens(id)` cuando el update incluye `permisos` o `activo`.
El modal de usuarios solo envía `permisos` cuando el array cambió. El detalle
del CRUD y del catálogo está en [usuarios-permisos.md](./usuarios-permisos.md).

- Si se desactiva la cuenta vía API (`activo=false`), la siguiente request
  encuentra el `jti` ausente, valida `user.Activo` y responde `401`.
- Si solo cambian los permisos vía API, la sesión continúa pero el token se
  regenera con los permisos nuevos.
- Cambios de `username`, `email` o `password` **no** revocan tokens.

## Sincronización de permisos con sesión activa

Caso típico: el usuario tiene 3 permisos y un administrador le revoca 2.

Configuración por defecto: `JWT_STORE_ENABLED=true` y
`JWT_LOAD_USER_ON_RENEW=true`.

1. **Login (t=0)**: el JWT incluye `permisos=[buses, usuarios, shop]` y un
   `jti=J1`. El `JWTStore` registra `J1 -> usuario`.
2. **Admin revoca 2 permisos (t=1)**: el update llega con `permisos=[buses]`.
   `UserHandler.Update` actualiza la BD y ejecuta `RemoveUserTokens(id)`. El
   `J1` deja de existir en el store.
3. **El usuario refresca (t=2)**: `authGuard` llama a `GET /auth/me`. Antes de
   ejecutar el handler, `AuthMiddleware`:
   - Valida firma y expiración del JWT (sigue siendo válido y todavía contiene
     3 permisos).
   - Comprueba que `J1` existe: ya no está (`back/api/handlers/auth.go:115`).
   - Como `JWT_LOAD_USER_ON_RENEW=true`, recarga el usuario desde la BD y genera
     un **JWT nuevo** con los permisos actuales (`[buses]`), actualiza la cookie
     y reemplaza los claims del contexto.
   - Recién entonces `/auth/me` responde con `[buses]`.
4. **Resto de endpoints**: todos los protegidos pasan por el mismo middleware
   antes de `tienePermiso`. Con el JWT viejo, `GET /usuarios` primero se
   refresca a `[buses]` y luego el chequeo devuelve `403`. `GET /buses` sigue
   funcionando. **La revocación aplica en la siguiente request, no solo al
   refrescar la página.**

> `/auth/me` no consulta la BD por sí mismo: solo devuelve los claims que el
> middleware ya refrescó. La sincronización ocurre en `AuthMiddleware`.

Matiz de frontend: el backend bloquea de inmediato, pero el signal
`auth.user` puede seguir mostrando los menús revocados hasta un refresh o login,
porque no se actualiza con cada respuesta. Al hacer clic, el backend responde
`403`; al refrescar, `/me` actualiza el menú.

### Comportamiento según configuración

| Configuración               | Revocación vía API (UI admin)                       | Cambio directo en BD            |
| --------------------------- | --------------------------------------------------- | ------------------------------- |
| `STORE=true`, `LOAD=true`   | Inmediata: `jti` ausente → recarga BD → token nuevo | ≤ 5 min (ventana de renovación) |
| `STORE=true`, `LOAD=false`  | Inmediata: `jti` ausente → `401 session expired`    | ≤ 15 min (expiración)           |
| `STORE=false`, `LOAD=true`  | No aplica; se sincroniza en la renovación (≤ 5 min) | ≤ 5 min                         |
| `STORE=false`, `LOAD=false` | Stale hasta expirar (≤ 15 min)                      | ≤ 15 min                        |

## Reinicios del backend

`JWTStore` es en memoria. Con `air`, cada guardado reinicia el proceso y se
pierden todos los `jti`:

- `STORE=true` + `LOAD=true` (default): la siguiente request recarga el usuario
  y genera un token nuevo. **La sesión sobrevive al reinicio** y los permisos se
  resincronizan.
- `STORE=true` + `LOAD=false`: tras cada reinicio, `jti` ausente → `401`. Se
  debe volver a iniciar sesión.
- `STORE=false`: no hay `jti`; la sesión sobrevive hasta expirar y los permisos
  se sincronizan solo en la renovación si `LOAD=true`.
- Cambiar `JWT_SECRET` invalida todos los tokens al instante,
  independientemente del store.

## Limitaciones conocidas

1. **Cambios directos en BD**: si los permisos se actualizan por SQL sin pasar
   por `PUT /users/{id}`, no se ejecuta `RemoveUserTokens`. El JWT sigue válido
   hasta la renovación (≤ 5 min con `LOAD=true`) o la expiración (≤ 15 min).
2. **Múltiples instancias**: el store es por proceso. La revocación solo afecta
   a la instancia que atendió al admin; las demás no recargan hasta su propia
   renovación (≤ 5 min) o expiración. Para producción con varias instancias hace
   falta un store compartido (BD/Redis).
3. **Desactivación directa en BD**: el branch de renovación por tiempo
   (`back/api/handlers/auth.go:150`) recarga al usuario pero **no valida
   `user.Activo`**, y el middleware tampoco revisa `claims.Activo`. Un usuario
   desactivado por SQL podría seguir renovando hasta expirar. Vía API/UI sí se
   corta, porque el branch de `jti` ausente sí valida `Activo`.
4. **Replay de token revocado**: con `LOAD=true`, un token todavía vigente cuyo
   `jti` fue eliminado puede renovarse si se reenvía. El logout normal funciona
   porque el navegador borra la cookie. Para revocación estricta hace falta
   `LOAD=false` o una lista negra persistente.
5. **Archivos**: `GET /archivos/{filename}` usa `OptionalAuthMiddleware`
   (`back/api/router.go:108`), que no consulta el store ni renueva. Un JWT
   todavía vigente podría acceder a archivos `autenticado`/`propietario` hasta
   expirar. Ver [archivos.md](./archivos.md).
6. **Eliminar un usuario no revoca sus tokens**: `UserHandler.Delete` no llama a
   `RemoveUserTokens`; su JWT sigue válido hasta expirar (≤ 15 min). Ver
   [usuarios-permisos.md](./usuarios-permisos.md).

## Frontend

`SidebarComponent.logout()` (`front/src/app/components/dashboard/sidebar/sidebar.ts`):

1. `mensajesStore.disconnect()` cierra la conexión SSE (ver
   [mensajes-sse.md](./mensajes-sse.md)).
2. `session.clear()` detiene el contador de sesión (ver
   [sesion-contador.md](./sesion-contador.md)).
3. `isLoggingOut.set(true)` deshabilita el botón y muestra el spinner.
4. `auth.logout()` → `POST /auth/logout` con `withCredentials` (interceptor).
5. En `next`: `auth.user.set(null)`.
6. En `complete`: `isLoggingOut.set(false)` y `router.navigate(['/login'])`.
7. En `error`: `isLoggingOut.set(false)`; si el error es `401`, el
   `errorInterceptor` limpia el usuario y redirige a `/login`.

Al llegar a `/login`, `loginGuard` encuentra `auth.user() === null`, llama a
`GET /auth/me`, recibe `401` porque la cookie ya no existe y deja mostrar el
formulario.

## Escenarios

| Acción                              | JWTStore             | Cookie                         | Resultado                                                        |
| ----------------------------------- | -------------------- | ------------------------------ | ---------------------------------------------------------------- |
| Logout normal                       | elimina el `jti`     | borrada                        | `user=null`, redirige a `/login`                                 |
| Permisos revocados vía API (default) | `RemoveUserTokens`   | se reemplaza en la sig. request | siguiente request recarga BD y usa los permisos nuevos           |
| Usuario desactivado vía API         | `RemoveUserTokens`   | se borra                       | `401 session expired` (no renueva por `Activo=false`)            |
| Cambio directo en BD                | sin cambios          | intacta                        | sync recién en la renovación (≤ 5 min con `LOAD=true`)           |
| Reinicio del backend (`air`)        | store vacío          | intacta                        | con `LOAD=true` renueva y continúa; con `false` → `401`          |
| `JWT_STORE_ENABLED=false`           | no aplica            | borrada en logout              | sin revocación server-side; sync solo por renovación/expiración  |

## Archivos involucrados

| Capa     | Archivo                                                |
| -------- | ------------------------------------------------------ |
| Backend  | `back/api/router.go`                                   |
| Backend  | `back/api/handlers/auth.go`                            |
| Backend  | `back/api/handlers/user.go`                            |
| Backend  | `back/infrastructure/auth/jwt_store.go`                |
| Backend  | `back/infrastructure/auth/jwt.go`                      |
| Frontend | `front/src/app/components/dashboard/sidebar/sidebar.ts` |
| Frontend | `front/src/app/components/dashboard/sidebar/sidebar.html` |
| Frontend | `front/src/app/services/auth.service.ts`               |
| Frontend | `front/src/app/guards/login.guard.ts`                  |
| Frontend | `front/src/app/interceptors/error.interceptor.ts`      |

## Variables de entorno relevantes

```
JWT_STORE_ENABLED=true
JWT_LOAD_USER_ON_RENEW=true
JWT_EXPIRATION_MINUTES=15
JWT_RENEW_MINUTES=5
COOKIE_NAME=access_token_go
COOKIE_SECURE=false
```
