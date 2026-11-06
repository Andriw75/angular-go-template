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
   - `Exists(jti)` (`jwt_store.go`): lo consulta `AuthMiddleware` en cada
     request autenticada y descarta entradas vencidas.
   - `Rotate(old, new, ...)`: se usa en la renovación deslizante para eliminar
     el `jti` anterior y no acumular entradas.
   - Cada entrada guarda `expiresAt`; al superar 1024 tokens se purgan las
     vencidas.

## Revocación por actualización de usuario

`UserHandler.Update` (`back/api/handlers/user.go`) llama a
`JWTStore.RemoveUserTokens(id)` cuando el update incluye `permisos`, `activo` o
una contraseña nueva. `UserHandler.Delete` hace lo mismo tras borrar la fila.
El modal de usuarios solo envía `permisos` cuando el array cambió. El detalle
del CRUD y del catálogo está en [usuarios-permisos.md](./usuarios-permisos.md).

- Permisos, contraseña o `activo` cambiados vía API: la siguiente request
  encuentra el `jti` ausente y responde `401 session_expired`; el usuario debe
  iniciar sesión de nuevo con los datos actuales.
- Usuario eliminado: sus tokens se revocan y la siguiente request responde
  `401`.
- Cambios de `username` o `email` **no** revocan tokens.

## Sincronización de permisos con sesión activa

Caso típico: el usuario tiene 3 permisos y un administrador le revoca 2.

Configuración por defecto: `JWT_STORE_ENABLED=true`.

1. **Login (t=0)**: el JWT incluye `permisos=[buses, usuarios, shop]` y un
   `jti=J1`. El `JWTStore` registra `J1 -> usuario`.
2. **Admin revoca 2 permisos (t=1)**: el update llega con `permisos=[buses]`.
   `UserHandler.Update` actualiza la BD y ejecuta `RemoveUserTokens(id)`. El
   `J1` deja de existir en el store.
3. **Siguiente request (t=2)**: `AuthMiddleware` valida la firma, comprueba que
   `J1` exista y, al no encontrarlo, limpia la cookie y responde
   `401 session_expired`. **No reemite el token.**
4. **El usuario vuelve a iniciar sesión** y recibe un token con los permisos
   actuales (`[buses]`).

La revocación es inmediata y determinista. El costo es que cualquier cambio de
permisos, `activo`, contraseña o eliminación de usuario obliga a re-login. Lo
mismo aplica a los cambios hechos por `PUT /users/{id}` y a `DELETE /users/{id}`.

> `/auth/me` solo devuelve los claims del token vigente. Si el `jti` fue
> revocado, el middleware responde `401` antes de llegar al handler.

### Comportamiento según configuración

| Configuración               | Revocación vía API (UI admin)      | Cambio directo en BD            |
| --------------------------- | ---------------------------------- | ------------------------------- |
| `STORE=true`, `LOAD=true`   | Inmediata: `jti` ausente → `401`   | Recarga en la renovación (≤ 5 min) |
| `STORE=true`, `LOAD=false`  | Inmediata: `jti` ausente → `401`   | Hasta expirar (≤ 15 min)        |
| `STORE=false`, `LOAD=true`  | Sin revocación server-side         | Recarga en la renovación (≤ 5 min) |
| `STORE=false`, `LOAD=false` | Sin revocación server-side         | Hasta expirar (≤ 15 min)        |

## Reinicios del backend

`JWTStore` es en memoria. Con `air`, cada guardado reinicia el proceso y se
pierden todos los `jti`:

- `STORE=true` (default): la siguiente request no encuentra el `jti` y responde
  `401 session_expired`. **Hay que iniciar sesión de nuevo.** Es el precio de
  que la revocación sobreviva a un reinicio.
- `STORE=false`: no hay lista de revocación; el token sigue válido hasta
  expirar y los permisos se sincronizan solo en la renovación si `LOAD=true`.
- Cambiar `JWT_SECRET` invalida todos los tokens al instante,
  independientemente del store.

## Limitaciones conocidas

1. **Cambios directos en BD**: si los permisos se actualizan por SQL sin pasar
   por `PUT /users/{id}`, no se ejecuta `RemoveUserTokens`. El JWT sigue válido
   hasta la renovación (≤ 5 min con `LOAD=true`) o la expiración (≤ 15 min).
2. **Múltiples instancias**: el store es por proceso. Tras revocar, las demás
   instancias no ven el `jti` ausente y el token sigue válido allí hasta
   expirar. Para producción con varias instancias hace falta un store
   compartido (BD/Redis).
3. **Desactivación directa en BD**: con `LOAD=true`, la renovación recarga al
   usuario y rechaza cuentas inactivas (`401`). Con `LOAD=false`, el token sigue
   válido hasta expirar. Vía API/UI se revoca de inmediato.
4. **Replay**: con `STORE=true`, un token revocado no sirve para ninguna ruta:
   el `jti` ausente produce `401`, incluso si el token aún no expiró. Con
   `STORE=false` no hay revocación server-side.
5. **Archivos**: `OptionalAuthMiddleware` ahora también valida el `jti`. Un
   token revocado se trata como anónimo: los archivos `publico` siguen
   sirviéndose y los `autenticado`/`propietario` responden `401`. Ver
   [archivos.md](./archivos.md).
6. **Eliminar un usuario**: `UserHandler.Delete` revoca sus tokens con
   `RemoveUserTokens`; si el store está deshabilitado, el JWT sigue válido hasta
   expirar. Ver [usuarios-permisos.md](./usuarios-permisos.md).

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
| Permisos/contraseña/activo vía API  | `RemoveUserTokens`   | borrada en la sig. request     | `401 session_expired`; el usuario vuelve a iniciar sesión         |
| Usuario eliminado                   | `RemoveUserTokens`   | borrada en la sig. request     | `401 session_expired`                                            |
| Cambio directo en BD                | sin cambios          | intacta                        | sync en la renovación (≤ 5 min con `LOAD=true`) o al expirar      |
| Reinicio del backend (`air`)        | store vacío          | intacta                        | `STORE=true` → `401`; `STORE=false` → sobrevive hasta expirar     |
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
