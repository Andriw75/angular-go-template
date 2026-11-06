# Flujo de login

## Resumen

El frontend envía `username` y `password` a `POST /auth/token`. El backend valida
la contraseña contra el hash bcrypt almacenado y, si es correcta, responde con
los datos del usuario y establece una cookie **HttpOnly** con el JWT.

> Importante: la contraseña **no** se guarda en cookies ni en el frontend.
> Se envía en el cuerpo del request y se descarta. Lo que persiste es el token
> JWT en una cookie HttpOnly, inaccesible para JavaScript.

## Secuencia (login inicial)

```
Usuario            LoginComponent        AuthService             API Go
  |                     |                    |                     |
  |-- usuario/clave --->|                    |                     |
  |                     |-- login(u,p) ----->|                     |
  |                     |                    |-- POST /auth/token ->|
  |                     |                    |   (form-urlencoded) |
  |                     |                    |                     |-- bcrypt.Compare
  |                     |                    |                     |-- genera JWT
  |                     |                    |<-- 200 + Set-Cookie -|
  |                     |<-- UserResponse ---|                     |
  |                     |-- user.set(u)      |                     |
  |                     |-- navigate(/dashboard)                   |
  |                     |                    |                     |
  |          authGuard ve user != null y permite sin llamar a la API
```

## Backend

1. Ruta: `POST /auth/token` (`back/api/router.go`), sin autenticación previa.
2. Handler `AuthHandler.Token` (`back/api/handlers/auth.go`):
   - Lee `username` y `password` con `r.FormValue` (el frontend envía
     `application/x-www-form-urlencoded`).
   - Si falta alguno, responde `400`.
   - Valida credenciales con `UserStore.IsValidPassword`:
     - Busca el usuario por `username`.
     - Compara con `bcrypt.CompareHashAndPassword`.
     - Si falla, responde `401 {"error":"invalid credentials"}`.
   - Genera el JWT con `JWTManager.Generate`.
   - Escribe la cookie y responde `200` con `UserResponse`
     (`id`, `username`, `email`, `activo`, `permisos`); el hash nunca se expone.
3. JWT (`back/infrastructure/auth/jwt.go`):
   - Firma `HS256` con `JWT_SECRET`.
   - Claims: `user_id`, `username`, `email`, `activo`, `permisos` y `jti`.
   - Expiración: `JWT_EXPIRATION_MINUTES` (15 por defecto).
   - El `jti` se registra en `JWTStore` en memoria si `JWT_STORE_ENABLED=true`.
4. Cookie de sesión:
   - Nombre: `COOKIE_NAME` (por defecto `access_token_go`).
   - `HttpOnly: true`, `Path: /`, `SameSite: Strict`.
   - `Secure` según `COOKIE_SECURE` (`false` en local, `true` detrás de HTTPS).
   - `Expires = ahora + JWT_EXPIRATION_MINUTES`.

## Frontend

1. Ruta `/login` (`front/src/app/app.routes.ts`) protegida por `loginGuard`.
2. `LoginComponent` (`front/src/app/components/login/login.ts`):
   - Signals `username`, `password`, `error`, `isLoading`.
   - `handleSubmit` recorta espacios y valida que ambos campos existan.
   - Llama a `AuthService.login(user, pass)`.
   - Éxito: `auth.user.set(u)` y navega a `/dashboard`.
   - Error: muestra `Usuario o contraseña incorrectos`.
3. `AuthService` (`front/src/app/services/auth.service.ts`):
   - `login()` hace `POST ${API_URL}/auth/token` con `URLSearchParams` y
     `Content-Type: application/x-www-form-urlencoded`.
   - `me()` hace `GET /auth/me` para restaurar la sesión desde la cookie.
   - `logout()` hace `POST /auth/logout`.
   - `user` es un signal en memoria (se pierde al recargar).
4. Interceptores (`front/src/app/app.config.ts`):
   - `credentialsInterceptor`: añade `withCredentials: true` a **todas** las
     peticiones para que el navegador envíe la cookie.
   - `errorInterceptor`: ante `401` en endpoints protegidos (excepto
     `/auth/token` y `/auth/me`), limpia `auth.user` y redirige a `/login`.

## Navegación en la SPA vs recarga completa

Este es el punto clave: **los guards se ejecutan antes de instanciar el
componente** y consultan primero el signal `auth.user`. Si el usuario ya está
en memoria, no se llama a la API.

### Dentro de la SPA (sin recargar)

- Después del login exitoso, `LoginComponent` hace `auth.user.set(u)` y
  `router.navigate(['/dashboard'])`.
- `authGuard` (protege `/dashboard`) ve `auth.user()` no nulo y devuelve
  `of(true)` **sin llamar a `/auth/me`**. El dashboard se instancia directo.
- Si el usuario ya logueado navega manualmente a `/login`, `loginGuard` ve el
  signal y redirige a `/dashboard` **sin llamar a la API**. `LoginComponent`
  ni siquiera llega a crearse.

### Al refrescar (F5) o abrir en una pestaña nueva

- El signal `user` es solo memoria, por lo que se pierde.
- El guard correspondiente encuentra `auth.user() === null` y llama a
  `GET /auth/me` para intentar restaurar la sesión desde la cookie HttpOnly.
- Si la cookie es válida: repuebla `auth.user` y continúa (o redirige a
  `/dashboard` si estaba en `/login`).
- Si falla: `authGuard` redirige a `/login`; `loginGuard` deja mostrar el login.

### Tabla de escenarios

| Escenario                          | `auth.user()` | ¿Llama `/auth/me`? | Resultado                          |
| ---------------------------------- | ------------- | ------------------ | ---------------------------------- |
| Login exitoso → `/dashboard`       | asignado      | No                 | `authGuard` permite                |
| Ya logueado → `/login`             | asignado      | No                 | `loginGuard` redirige a dashboard  |
| F5 en `/dashboard`                 | `null`        | Sí                 | restaura sesión o redirige a login |
| F5 en `/login` con cookie válida   | `null`        | Sí                 | restaura y redirige a dashboard    |

> `errorInterceptor` excluye `/auth/token` y `/auth/me` para no duplicar
> redirecciones durante el login o la restauración de sesión.

## Renovación del token

En cada request autenticada, `AuthMiddleware`:

1. Lee y valida la cookie.
2. Si `JWT_STORE_ENABLED`, exige que el `jti` exista. Si no existe (logout,
   revocación, usuario eliminado o reinicio del backend), responde
   `401 session_expired` y borra la cookie: hay que iniciar sesión de nuevo.
3. Si pasaron `JWT_RENEW_MINUTES` (5 por defecto) desde `iat`, genera un token
   nuevo, **rota el `jti`** (elimina el anterior) y actualiza la cookie. Con
   `JWT_LOAD_USER_ON_RENEW=true` recarga permisos y estado desde la BD y rechaza
   cuentas inactivas.
4. Verifica el permiso requerido por la ruta (si aplica).
5. Inyecta los claims en el contexto (`WithClaims`).

## CORS y cookies

- El backend responde con `Access-Control-Allow-Origin` igual al origen de la
  petición **solo si está en la allowlist** de `CORS_ORIGIN` (lista separada por
  comas) y con `Access-Control-Allow-Credentials: true`. Añade `Vary: Origin`.
- En desarrollo, el frontend (`:4200`) consume la API vía nginx (`:5000`) o
  directo (`:8080`); el origen debe coincidir exactamente con una entrada de
  `CORS_ORIGIN`.
- `SameSite=Strict` funciona porque front y API comparten sitio (localhost o
  mismo dominio tras nginx).

## Errores esperados

| Caso                         | Respuesta                 |
| ---------------------------- | ------------------------- |
| Faltan usuario o contraseña  | `400`                     |
| Credenciales inválidas       | `401 invalid credentials` |
| Sin cookie en ruta protegida | `401 not authenticated`   |
| Token inválido/expirado      | `401 invalid token`       |
| Sesión revocada o jti ausente| `401 session_expired`     |
| Sin permiso para la ruta     | `403 forbidden`           |

## Archivos involucrados

| Capa     | Archivo                                                 |
| -------- | ------------------------------------------------------- |
| Backend  | `back/api/router.go`                                    |
| Backend  | `back/api/handlers/auth.go`                             |
| Backend  | `back/infrastructure/auth/jwt.go`                       |
| Backend  | `back/infrastructure/auth/jwt_store.go`                 |
| Backend  | `back/infrastructure/auth/crypt.go`                     |
| Backend  | `back/services/config.go`                               |
| Frontend | `front/src/app/components/login/login.ts`               |
| Frontend | `front/src/app/components/login/login.html`             |
| Frontend | `front/src/app/services/auth.service.ts`                |
| Frontend | `front/src/app/guards/auth.guard.ts`                    |
| Frontend | `front/src/app/guards/login.guard.ts`                   |
| Frontend | `front/src/app/interceptors/credentials.interceptor.ts` |
| Frontend | `front/src/app/interceptors/error.interceptor.ts`       |
| Frontend | `front/src/app/app.config.ts`                           |

## Variables de entorno relevantes

```
JWT_SECRET=...
JWT_EXPIRATION_MINUTES=15
JWT_RENEW_MINUTES=5
JWT_STORE_ENABLED=true
JWT_LOAD_USER_ON_RENEW=true
COOKIE_NAME=access_token_go
COOKIE_SECURE=false
CORS_ORIGIN=http://localhost:4200
```

## Ver también

- [Logout y revocación](./logout.md): revocación del token y sincronización de
  permisos con sesión activa.
- [Contador de sesión](./sesion-contador.md): tiempo restante y auto-logout.
- [Usuarios y permisos](./usuarios-permisos.md): origen del catálogo de
  permisos, CRUD de usuarios y hook `RemoveUserTokens`.
