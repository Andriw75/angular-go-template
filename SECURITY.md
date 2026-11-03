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

## 8. Servidor Go

`cmd/server/main.go` configura:

```go
ReadHeaderTimeout: 10 * time.Second
ReadTimeout:       30 * time.Second
IdleTimeout:       120 * time.Second
MaxHeaderBytes:    1 << 20
```

`WriteTimeout` queda en 0 por el endpoint SSE de mensajes. Para otros handlers
de larga duración, usar `http.NewResponseController(w).SetWriteDeadline`.

---

## 9. Dependencias

- Go: toolchain parcheado (`go 1.26.10`) y `govulncheck ./...`.
- Node: `npm audit --omit=dev` sin `high`/`critical`.
- `exceljs 4.4.0` arrastra advisories moderados (`uuid`, `archiver`, `glob`,
  `minimatch`) sin parche upstream. Riesgo aceptado: se usa en el navegador para
  generar archivos, no procesa entradas no confiables en servidor. Revisar al
  actualizar exceljs.
- `vitest` se mantiene en la última 4.x (dev).
- CI: `.github/workflows/security.yml` ejecuta `go vet`, `govulncheck`,
  `npm audit --omit=dev` y `gitleaks`.

---

## 10. Despliegue (nginx)

- Producción: TLS 1.2+, redirect 80→443, HSTS.
- `client_max_body_size 120M`, timeouts de body/header/send.
- `limit_req` / `limit_conn` en auth y API.
- Bloquear archivos ocultos (`.env`, `.git`) y `/template/archivos`.
- No exponer el puerto de Go directamente a Internet.
- Cabeceras de seguridad (CSP, `X-Frame-Options`, `Referrer-Policy`): pendientes
  por decisión del equipo; el bloque TLS comentado en `nginx.conf` las incluye.

---

## 11. Si se expone un secreto

1. **Rotarlo primero** (el historial no des-expone nada).
2. Hacer backup: `git bundle create backup.bundle --all`.
3. Reescribir con `git filter-repo --replace-text replacements.txt`.
4. `git push --force --all` y avisar a colaboradores para re-clonar.
5. GitHub conserva objetos antiguos un tiempo; para purga total, borrar/recrear
   el repo o contactar soporte.

---

## 12. Checklist antes de desplegar

- [ ] `JWT_SECRET` generado y de 32+ caracteres.
- [ ] Sin `BOOTSTRAP_ADMIN_PASSWORD` en producción después del primer login.
- [ ] `COOKIE_SECURE=true` y HTTPS/HSTS activos.
- [ ] `npm audit --omit=dev` y `govulncheck ./...` limpios.
- [ ] `limit_req` de login probado (429 tras 5 intentos).
- [ ] Subida rechaza archivo con extensión válida pero contenido falso.
- [ ] `/instalaciones` y documentos revisados con su nivel de acceso.
- [ ] Logs sin secretos ni PII.
