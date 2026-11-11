# Cambio obligatorio de contraseña

## Resumen

El template exige cambiar la contraseña antes de usar el dashboard cuando la
cuenta tiene el flag `must_change_password`. El flag viaja dentro del JWT y se
activa en tres casos:

- el primer administrador creado con `BOOTSTRAP_ADMIN_*`;
- un usuario creado por un administrador;
- un reset de contraseña hecho por un administrador.

Mientras el flag esté activo, `AuthMiddleware` bloquea cualquier ruta protegida
que no empiece por `/auth/` con `403 password_change_required`. El frontend
redirige a `/change-password`; al cambiarla, se revocan todos los tokens del
usuario y se obliga a iniciar sesión de nuevo.

> La contraseña actual y la nueva viajan en el cuerpo del request y no se
> persisten en el frontend. Lo único que persiste es la cookie HttpOnly.

## Secuencia

```
Usuario        Login          Guards          Cambio          API Go           BD / JWT
  |              |               |               |               |                |
  |-- login ---->|               |               |               |                |
  |              |-- POST /auth/token --------------------------->|                |
  |              |               |               |               |-- bcrypt ------>|
  |              |<-- 200 + cookie + must_change_password --------|                |
  |              |-- navigate --->|               |               |                |
  |              |               |-- si flag ---->|               |                |
  |              |               |               |-- POST /auth/change-password ->|
  |              |               |               |               |-- valida ------>|
  |              |               |               |               |-- hash + flag=false
  |              |               |               |               |-- revoca tokens
  |              |               |               |<-- 200 --------|                |
  |              |<-- /login?changed=1 ----------|               |                |
  |-- login con la nueva contraseña ----------->|               |                |
```

## Backend

### Origen del flag

| Situación | Dónde se activa | Resultado |
| --- | --- | --- |
| Bootstrap del primer admin | `database.Seed` inserta `must_change_password=1` | El admin debe cambiarla en el primer acceso. |
| Alta de usuario | `UserHandler.Create` fuerza `input.MustChangePassword = true` | Todo usuario creado por un admin debe cambiarla. |
| Reset de contraseña | `UserStore.Update` pone `existing.MustChangePassword = true` cuando llega una contraseña nueva | El usuario afectado debe cambiarla al volver a entrar. |
| Update explícito | `PUT /users/{id}` acepta `must_change_password` | Permite forzarlo o limpiarlo desde API. |

El flag se guarda en la columna `users.must_change_password` y se copia al claim
`must_change_password` del JWT al iniciar sesión o al renovar el token.

### Bloqueo global

`AuthMiddleware` (`back/api/handlers/auth.go`) comprueba el claim después de
validar el token y antes de comprobar permisos:

- Si `must_change_password=true` y una ruta protegida por `AuthMiddleware` **no**
  empieza por `/auth/`, responde `403` con código `password_change_required`.
  Las rutas con `OptionalAuthMiddleware` (imágenes y archivos públicos) no
  aplican este bloqueo.
- Las rutas que empiezan por `/auth/` siguen accesibles: `/auth/me`,
  `/auth/logout` y `/auth/change-password` permiten consultar el estado, salir
  o completar el cambio.

### Cambio de contraseña

`POST /auth/change-password` (`AuthHandler.ChangePassword`) requiere sesión y
recibe:

```json
{
  "current_password": "...",
  "new_password": "..."
}
```

Pasos:

1. Comprueba que ambos campos lleguen.
2. Busca al usuario por `claims.UserID`.
3. Verifica `current_password` con bcrypt.
4. Rechaza que la nueva sea igual a la actual.
5. Aplica `services.ValidatePasswordForUser`:
   - mínimo 12 caracteres;
   - máximo 72 bytes (límite de bcrypt);
   - contraseña no común;
   - distinta al username.
6. Hashea la nueva contraseña.
7. `UserStore.UpdatePassword(user.ID, hash, false)` guarda el hash y limpia
   `must_change_password`.
8. Si `JWT_STORE_ENABLED=true`, ejecuta `JWTStore.RemoveUserTokens(user.ID)`.
9. Borra la cookie de sesión.
10. Responde `200 {"message":"contraseña actualizada; inicia sesión de nuevo"}`.

No se actualiza el JWT en caliente: el usuario debe volver a autenticarse. Eso
evita que un token viejo conserve el flag o permisos anteriores.

## Frontend

### Login

`LoginComponent` (`front/src/app/components/login/login.ts`) hace:

```ts
this.auth.login(user, pass).subscribe({
  next: (u) => {
    this.auth.user.set(u);
    this.router.navigate([u.must_change_password ? '/change-password' : '/dashboard']);
  },
});
```

Si el usuario ya está logueado y entra a `/login`, `loginGuard` decide entre
`/dashboard` y `/change-password` según el flag.

### Guards

- `authGuard` protege `/dashboard` y `/change-password`. Si el usuario tiene
  `must_change_password` y el destino no es `/change-password`, redirige.
- `loginGuard` protege `/login`. Si ya hay sesión, redirige al destino que
  corresponda.
- Ambos guards restauran la sesión con `GET /auth/me` cuando `auth.user` está
  vacío (F5 o pestaña nueva).

### Interceptor de errores

`errorInterceptor` (`front/src/app/interceptors/error.interceptor.ts`) escucha
el `403` con código `password_change_required` y navega a `/change-password`
si todavía no está allí. Esto cubre llamadas directas a la API o una sesión
restaurada con el flag activo.

### Pantalla de cambio

`ChangePasswordComponent`
(`front/src/app/components/change-password/change-password.ts`):

- Signals `currentPassword`, `newPassword`, `confirmPassword`, `error` e
  `isLoading`.
- Valida en cliente: campos completos, mínimo 12 caracteres, confirmación igual
  y nueva distinta a la actual.
- Llama a `AuthService.changePassword(current, new)`.
- En éxito limpia `auth.user` y navega a `/login?changed=1`.
- `LoginComponent` lee `changed=1` y muestra “Contraseña actualizada. Inicia
  sesión con la nueva.”.

## Escenarios

| Escenario | Resultado |
| --- | --- |
| Primer login del admin bootstrap | `must_change_password=true`; dashboard bloqueado hasta cambiarla. |
| Admin crea un usuario | El usuario nace con el flag activo. |
| Admin resetea una contraseña | El flag se activa y se revocan los tokens del usuario. |
| Usuario cambia su contraseña | Se guarda el hash, se limpia el flag y se revocan todas sus sesiones. |
| Usuario intenta usar el dashboard con el flag activo | `403 password_change_required`; el frontend redirige. |
| Usuario cancela y vuelve a `/login` | Puede iniciar sesión con la contraseña anterior; el flag sigue activo. |
| Contraseña nueva débil | `400` con el mensaje de la política. |
| Contraseña actual incorrecta | `400` “la contraseña actual es incorrecta”. |

## Errores esperados

| Caso | Respuesta |
| --- | --- |
| Body inválido | `400 invalid request body` |
| Faltan `current_password` o `new_password` | `400` |
| Contraseña actual incorrecta | `400` |
| Nueva igual a la actual | `400` |
| Nueva demasiado corta, común o igual al username | `400` |
| Sin sesión | `401 not authenticated` |
| Token inválido o revocado | `401 invalid token` / `401 session_expired` |
| Ruta protegida con flag activo | `403 password_change_required` |
| Usuario inexistente | `404 user not found` |
| Error al hashear o guardar | `500 internal error` |

## Limitaciones

1. **No hay historial de contraseñas**: solo se impide repetir la actual en el
   cambio, no las anteriores.
2. **El reset de admin revoca todas las sesiones**: el usuario debe volver a
   entrar y cambiar la contraseña.
3. **El flag solo se limpia con el cambio propio**: un admin no puede “aprobar”
   el flag sin que el usuario cambie la contraseña, salvo que use
   `must_change_password=false` por API.
4. **Sin MFA ni recuperación por email**: el template no incluye flujo de
   “olvidé mi contraseña”; el reset lo hace un administrador o
   `cmd/set_password`.
5. **La cookie se borra al cambiar**: si el usuario tenía varias pestañas,
   las demás reciben `401` en su siguiente request.

## Archivos involucrados

| Capa | Archivo |
| --- | --- |
| Backend | `back/api/handlers/auth.go` |
| Backend | `back/api/handlers/user.go` |
| Backend | `back/infrastructure/database/users.go` |
| Backend | `back/infrastructure/database/db.go` |
| Backend | `back/infrastructure/auth/jwt.go` |
| Backend | `back/services/password.go` |
| Frontend | `front/src/app/components/login/login.ts` |
| Frontend | `front/src/app/components/change-password/change-password.ts` |
| Frontend | `front/src/app/guards/auth.guard.ts` |
| Frontend | `front/src/app/guards/login.guard.ts` |
| Frontend | `front/src/app/interceptors/error.interceptor.ts` |
| Frontend | `front/src/app/services/auth.service.ts` |
| Frontend | `front/src/app/app.routes.ts` |

## Variables de entorno relevantes

```
BOOTSTRAP_ADMIN_USERNAME=admin
BOOTSTRAP_ADMIN_PASSWORD=<mínimo 12 caracteres, no común>
BOOTSTRAP_ADMIN_EMAIL=admin@example.com
JWT_STORE_ENABLED=true
```

## Ver también

- [Login](./login.md): emisión del JWT y navegación inicial.
- [Logout y revocación](./logout.md): qué pasa cuando se revocan tokens.
- [Usuarios y permisos](./usuarios-permisos.md): alta, reset y catálogo.
- [SECURITY.md](../../SECURITY.md): política de contraseñas y primer admin.
