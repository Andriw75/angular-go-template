# Seguridad — angular-go-template

Documento vivo con las reglas y decisiones de seguridad del proyecto. Aplicar
con criterio: no todas las reglas son obligatorias para todos los entornos.

---

## 1. Modelo de acceso por endpoint

Cada ruta declara **explícitamente** uno de estos niveles. No se pone
`AuthMiddleware(permiso)` por reflejo: se elige el mínimo necesario.

| Nivel | Implementación | Ejemplos |
|---|---|---|
| `public` | Sin middleware | Landing, catálogo público sin PII |
| `authenticated` | `AuthMiddleware()` | `/auth/me`, `/instalaciones` |
| `permission` | `AuthMiddleware("x")` | `/users` → `usuarios` |
| `owner` | `AuthMiddleware()` + comparar `claims.UserID` | `/archivos/{id}` |

Decisiones actuales:
- `/instalaciones`: **solo autenticado** (decisión del equipo).
- `/documentos`: permiso `documentos`; el recurso es global para quien tenga ese
  permiso. Si en el futuro se requiere aislamiento por usuario, usar `user_id`
  (ya existe en la tabla `documentos`).
- `/documentos/jobs/{id}`: mismo criterio que `/documentos`. El job vive en
  memoria, dura `JOB_TTL_MINUTES` y es compartido para quien tenga el permiso.
- `/archivos`: `publico` sin login, `autenticado` con login y `propietario` solo
  para el dueño.

---

## 2. Secretos y configuración

- `JWT_SECRET` es **obligatorio** y debe tener **32+ caracteres**. El backend
  no arranca si falta o es corto (`services.Config.Validate`).
- Generar con: `go run ./cmd/generate_secrets.go`.
- No hay valores por defecto de contraseñas ni de JWT en código ni en
  `.env.example`.
- `.env` está en `.gitignore` y nunca se commitea.
- `DB_PASSWORD` no tiene valor por defecto (`postgres` fue eliminado).
- `DB_SSLMODE` es configurable: `disable` en local o servidores sin TLS;
  `require`/`verify-full` cuando el servidor lo soporte. **No se fuerza.**
- En producción: `COOKIE_SECURE=true` y HTTPS.
- No imprimir secretos en logs ni respuestas.

---

## 3. Primer administrador

No existe usuario por defecto. Para una base vacía:

```env
BOOTSTRAP_ADMIN_USERNAME=admin
BOOTSTRAP_ADMIN_PASSWORD=<mínimo 12 caracteres, no común>
BOOTSTRAP_ADMIN_EMAIL=admin@example.com
```

El usuario se crea con `must_change_password=1`; al iniciar sesión se le exige
cambiar la contraseña antes de acceder al resto de la API. Si
`BOOTSTRAP_ADMIN_PASSWORD` está vacío, no se crea ningún admin.

Para una base que ya tiene un admin creado con el seed anterior, rota su
contraseña sin necesitar la vieja:

```bash
cd back
NEW_PASSWORD='...' go run ./cmd/set_password -user USUARIO_ADMIN
```

---

## 4. Contraseñas

- Mínimo **12 caracteres**, máximo **72 bytes** (límite de bcrypt).
- Rechazo de contraseñas comunes (`password`, `admin123`, etc.) y de la
  contraseña igual al usuario (`services.ValidatePasswordForUser`).
- Al crear un usuario o resetear su contraseña, se marca
  `must_change_password` para que la cambie en el primer login.
- El cambio de contraseña revoca todos los tokens del usuario y limpia la cookie.

---

## 5. Login y fuerza bruta

Dos capas:

1. **nginx** (`deploy/nginx.conf`): `limit_req` 5 r/min con `burst=5` en
   `/template/api/auth/token` y 30 r/s en el resto de la API; `limit_conn` por IP.
2. **Go** (`auth.LoginLimiter`): 5 fallos por `IP + usuario` en 15 minutos →
   bloqueo 15 minutos, con `Retry-After`. Configurable con
   `LOGIN_MAX_ATTEMPTS`, `LOGIN_WINDOW_MINUTES`, `LOGIN_BLOCK_MINUTES`.

La clave incluye el usuario porque nginx no lo conoce. `X-Real-IP` /
`X-Forwarded-For` se usan detrás de nginx; si Go se expone directo, el límite
por usuario sigue protegiendo.

---

## 6. Subidas de archivos

- Límite global de body en `api/middleware.go`: **2 MiB** para JSON y **120 MiB**
  para subidas (`/archivos`, `/{categorias,productos}/{id}/imagenes`). Si se
  excede, la API responde `413 request_too_large`.
- Límite total por petición: 100 MB (`http.MaxBytesReader`) y 100 MB por archivo.
- Máximo 20 archivos por carga; imágenes 64 MB totales y 32 MB por archivo.
- Validación de **extensión + magic bytes** (`http.DetectContentType`).
- **Nombre generado por el servidor** (timestamp + 16 bytes aleatorios); el
  nombre original solo se guarda como metadato.
- `safeFileName` rechaza `..`, separadores y bytes nulos, incluso para filas
  antiguas de la BD.
- Documentos se sirven con `Content-Disposition: attachment` y
  `X-Content-Type-Options: nosniff`.
- nginx bloquea `/template/archivos`; solo Go sirve archivos privados.

---

## 7. Exportaciones

`report.safeCell` neutraliza valores que empiezan por `= + - @` y limpia
caracteres de control en Excel. Aunque excelize escribe strings como shared
strings (no fórmulas), la protección se mantiene para futuros exports CSV.

---

## 8. Sesiones JWT y revocación

- El JWT es stateless; la revocación se apoya en el claim `jti` y en
  `JWTStore` (en memoria). Cada token registra `jti -> userID` y su expiración.
- Con `JWT_STORE_ENABLED=true` (default), el `jti` debe existir en **cada**
  request. Si falta (logout, cambio de contraseña, permisos/estado cambiados,
  usuario eliminado o reinicio del backend) la API responde
  `401 session_expired` y borra la cookie: hay que iniciar sesión de nuevo.
- La renovación deslizante (`JWT_RENEW_MINUTES`) genera un token nuevo y **rota
  el jti**; el anterior se elimina para no acumular entradas.
- Las entradas vencidas se purgan al consultarlas y, en bloque, cuando el mapa
  supera 1024 tokens.
- `OptionalAuthMiddleware` (imágenes y archivos) también valida el `jti`, así
  que un token revocado no accede a recursos `autenticado`/`propietario`.
- Limitación: el store es por proceso. Con varias instancias hace falta un store
  compartido (BD/Redis) para revocación global. Reiniciar invalida las sesiones;
  es una decisión de seguridad deliberada.

---

## 9. Servidor Go

`cmd/server/main.go` configura:

```go
ReadHeaderTimeout: 10 * time.Second
ReadTimeout:       30 * time.Second
IdleTimeout:       120 * time.Second
MaxHeaderBytes:    1 << 20
```

`WriteTimeout` queda en 0 por el endpoint SSE de mensajes. Para otros handlers
de larga duración, usar `http.NewResponseController(w).SetWriteDeadline`.

`api/router.go` y `api/middleware.go` añaden:

- `securityHeaders`: `X-Content-Type-Options`, `X-Frame-Options`,
  `Referrer-Policy`, `Permissions-Policy` y CSP (`default-src 'none'`) para la API.
- `limitBody`: 2 MiB por defecto y 120 MiB en subidas; responde `413` con
  `request_too_large`.
- `middleware.Timeout`: 30 s en grupos JSON, 2 min en reportes/documentos y
  5 min en archivos. `/sse` y `/lab` quedan sin timeout por ser largos.
- CORS por allowlist (`CORS_ORIGIN` admite lista separada por comas), con
  `Vary: Origin` y sin reflejar orígenes no configurados.

---

## 10. Dependencias

- Go: toolchain parcheado (`go 1.26.10`) y `govulncheck ./...`.
- Node: `npm audit --omit=dev` sin `high`/`critical`.
- `exceljs 4.4.0` arrastra advisories moderados (`uuid`, `archiver`, `glob`,
  `minimatch`) sin parche upstream. Riesgo aceptado: se usa en el navegador para
  generar archivos, no procesa entradas no confiables en servidor. Revisar al
  actualizar exceljs.
- `vitest` se mantiene en la última 4.x (dev).
- CI: `.github/workflows/security.yml` ejecuta `go vet`, `go test -race ./...`,
  `govulncheck`, `npm audit --omit=dev` y `gitleaks`.

---

## 11. Despliegue (nginx)

- Producción: TLS 1.2+, redirect 80→443, HSTS.
- `client_max_body_size 120M`, timeouts de body/header/send.
- `limit_req` / `limit_conn` en auth y API.
- Bloquear archivos ocultos (`.env`, `.git`) y `/template/archivos`.
- No exponer el puerto de Go directamente a Internet.
- Cabeceras de seguridad activas en el `server`: `X-Content-Type-Options`,
  `X-Frame-Options`, `Referrer-Policy` y `Permissions-Policy`. HSTS vive en el
  bloque TLS. CSP se aplica solo en la API para no romper la SPA.

---

## 12. Si se expone un secreto

1. **Rotarlo primero** (el historial no des-expone nada).
2. Hacer backup: `git bundle create backup.bundle --all`.
3. Reescribir con `git filter-repo --replace-text replacements.txt`.
4. `git push --force --all` y avisar a colaboradores para re-clonar.
5. GitHub conserva objetos antiguos un tiempo; para purga total, borrar/recrear
   el repo o contactar soporte.

---

## 13. Checklist antes de desplegar

- [ ] `JWT_SECRET` generado y de 32+ caracteres.
- [ ] Sin `BOOTSTRAP_ADMIN_PASSWORD` en producción después del primer login.
- [ ] `COOKIE_SECURE=true` y HTTPS/HSTS activos.
- [ ] `npm audit --omit=dev` y `govulncheck ./...` limpios.
- [ ] `limit_req` de login probado (429 tras 5 intentos).
- [ ] `go test ./...` (y `-race` en CI) pasa.
- [ ] Reiniciar el backend invalida sesiones (`401 session_expired`).
- [ ] JSON grande devuelve `413` y una subida válida de 100 MB sigue funcionando.
- [ ] Cabeceras de seguridad presentes en SPA y API.
- [ ] Subida rechaza archivo con extensión válida pero contenido falso.
- [ ] `/instalaciones` y documentos revisados con su nivel de acceso.
- [ ] Logs sin secretos ni PII.
