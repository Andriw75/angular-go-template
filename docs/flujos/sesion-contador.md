# Contador de sesión y auto-logout

## Resumen

El sidebar puede mostrar el tiempo restante del JWT actual y cerrar la sesión
automáticamente al llegar a cero.

El backend **solo expone el dato**: `X-Session-Expires-In` con los segundos que
le quedan al token. La decisión de mostrar el contador y auto-cerrar es
**exclusivamente del frontend**, mediante `environment.SESSION`.

Es opt-in: con `environment.SESSION.COUNTDOWN_ENABLED=false` no se muestra el
contador ni se aplica el auto-logout proactivo. El backend sigue rechazando
tokens expirados con `401` como siempre.

## Configuración

`front/src/environments/environment.ts`:

```ts
SESSION: {
  COUNTDOWN_ENABLED: true,
  WARN_SECONDS: 60,
},
```

No hay variables de sesión en `back/.env`. El backend no necesita saber si el
cliente muestra el contador.

## Backend

`AuthHandler.setSessionHeaders` (`back/api/handlers/auth.go`) agrega:

| Header                 | Contenido                           |
| ---------------------- | ----------------------------------- |
| `X-Session-Expires-In` | Segundos restantes del token actual |

Se envía en:

- `POST /auth/token` (login).
- `GET /auth/me`.
- Cualquier ruta con `AuthMiddleware`.
- `OptionalAuthMiddleware` (archivos) si hay claims.

El valor se calcula **después** de una posible renovación, así que refleja el
token nuevo. Para que el navegador pueda leerlo, `corsMiddleware`
(`back/api/router.go`) lo expone con `Access-Control-Expose-Headers`.

No hay cambios en la base de datos ni flags nuevos en el backend: la expiración
sale del claim `exp`.

## Frontend

### `SessionService`

`front/src/app/services/session.service.ts`:

- Signals `remaining`, `active`, `warning`, `critical` y `label`.
- `updateFromHeaders()` convierte `X-Session-Expires-In` en
  `expiresAt = Date.now() + segundos * 1000`.
- Si `environment.SESSION.COUNTDOWN_ENABLED` es `false`, ignora los headers.
- Un timer de 1 s **recalcula desde `expiresAt`** (no decrementa), para tolerar
  el throttling de pestañas en segundo plano.
- Al volver a la pestaña (`visibilitychange`) recalcula inmediatamente.
- Al llegar a 0 emite `expired$` una sola vez y detiene el timer.
- `clear()` detiene el timer y limpia el estado (logout manual o `401`).

### Interceptores

- `sessionInterceptor` lee `X-Session-Expires-In` de cada `HttpResponse` y llama
  a `updateFromHeaders`.
- `errorInterceptor` limpia el contador ante cualquier `401`.

### Sidebar

`front/src/app/components/dashboard/sidebar/sidebar.html` muestra:

- `Sesión mm:ss` cuando el sidebar está expandido.
- Un punto verde/ámbar/rojo cuando está colapsado.
- Ámbar dentro de `SESSION.WARN_SECONDS`; rojo en los últimos 10 s.

### Expiración

`DashboardLayoutComponent` se suscribe a `session.expired$` y:

1. Desconecta el SSE (`MensajesStoreService.disconnect()`).
2. Limpia `auth.user`.
3. Limpia el contador.
4. Navega a `/login?expired=1`.

`LoginComponent` lee `expired=1` y muestra
“La sesión ha caducado. Vuelve a iniciar sesión.”.

## Flujo

```
Login  ──► header X-Session-Expires-In ──► SessionService inicia el contador
  │                                              │
  │◄── cualquier request autenticada ────────────┤ (actualiza, incluye renovación)
  │                                              │
  │                                         llega a 0
  │                                              │
  └── logout local + cierre SSE + /login?expired=1
```

## Comportamiento con la renovación deslizante

El contador mide el token **actual**, no la sesión total:

- Si el usuario hace peticiones, el middleware renueva cada
  `JWT_RENEW_MINUTES` y el contador vuelve a subir.
- Solo llega a 0 si pasa `JWT_EXPIRATION_MINUTES` sin una renovación (por
  ejemplo, usuario inactivo).
- Si quieres un máximo absoluto desde el login, eso requiere otro claim
  (`session_started_at`) y no se extiende con la actividad.
- Cada renovación rota el `jti` en el servidor (el anterior se elimina). El
  contador no cambia por ello: sigue midiendo el token actual.

## Limitaciones

1. **El backend no puede desactivar el header** por configuración: siempre
   informa la expiración. Es un dato no sensible; el frontend decide si lo usa.
2. **Pestañas en segundo plano**: el navegador puede frenar el timer. Se
   recalcula al volver a la pestaña y, de todos modos, la siguiente petición
   devuelve `401` si ya expiró.
3. **Multi-pestaña**: no hay sincronización explícita todavía. Una pestaña puede
   hacer logout y las demás conservar el contador hasta su próximo `401`.
4. **SSE**: el servidor no cierra el stream al expirar; lo cierra el frontend al
   recibir `expired$`. Un cliente que ignore el evento seguiría conectado hasta
   que el servidor o el proxy corten.
5. **Reloj**: se usan segundos restantes calculados por el servidor, pero el
   `Date.now()` del cliente puede desviarse unos segundos.
6. **No llama a `/auth/logout` al expirar**: el token ya está expirado y la
   cookie también. El logout manual sí llama al endpoint y revoca el `jti`.

## Archivos involucrados

| Capa     | Archivo                                                          |
| -------- | ---------------------------------------------------------------- |
| Backend  | `back/api/handlers/auth.go`                                      |
| Backend  | `back/api/router.go`                                             |
| Frontend | `front/src/environments/environment.ts`                          |
| Frontend | `front/src/app/services/session.service.ts`                      |
| Frontend | `front/src/app/interceptors/session.interceptor.ts`              |
| Frontend | `front/src/app/interceptors/error.interceptor.ts`                |
| Frontend | `front/src/app/app.config.ts`                                    |
| Frontend | `front/src/app/components/dashboard/layout/layout.ts`            |
| Frontend | `front/src/app/components/dashboard/sidebar/sidebar.ts` / `.html` |
| Frontend | `front/src/app/components/login/login.ts`                        |

## Ver también

- [Login](./login.md): emisión del JWT y cookie HttpOnly.
- [Logout y revocación](./logout.md): cierre manual y revocación del `jti`.
- [Mensajes pendientes y SSE](./mensajes-sse.md): desconexión del stream.
