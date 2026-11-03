# AGENTS.md

Guía breve de trabajo para agentes y desarrolladores. El detalle funcional vive
en `docs/flujos/`; este archivo es solo el punto de entrada y el índice.

## Contexto rápido

Monorepo con tres partes:

| Carpeta   | Stack                           | Rol                                      |
| --------- | ------------------------------- | ---------------------------------------- |
| `back/`   | Go 1.26 + chi + JWT + bcrypt    | API REST en `:8080`                      |
| `front/`  | Angular 21 standalone + signals | SPA servida bajo `/template/`            |
| `deploy/` | nginx                           | SPA, estáticos y proxy `/template/api/`  |

## Comandos habituales

```powershell
cd back
air                          # API con recarga automática (reinicia al guardar)
go build ./...
go vet ./...
go test ./...
# alternativa sin recarga automática: go run ./cmd/server

cd front
ng serve                     # SPA en http://localhost:4200 (recarga al guardar)
ng build --base-href=/template/
```

Arranca `air` y `ng serve` una sola vez: al guardar cambios en Go o Angular se
recompilan y recargan automáticamente, sin reiniciar nada a mano. `air` también
vigila `.env`. `npm start` es un alias de `ng serve`.

## Base de datos

- `DB_TYPE=sqlite|postgres` (ya no existe `USE_MOCK`).
- SQLite por defecto en `back/data/template.db` (`DB_PATH`).
- PostgreSQL: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`.
- Al arrancar se crean las tablas y se siembran los permisos base. **No hay
  usuario por defecto.** El primer admin se define con
  `BOOTSTRAP_ADMIN_USERNAME` / `BOOTSTRAP_ADMIN_PASSWORD` en `.env` y debe
  cambiar la contraseña en el primer login.
- Público en `PUBLIC_DIR/public` (nginx): imágenes de entidades activas y
  archivos `publico`. Restringido en `PUBLIC_DIR/archivos` (bloqueado en nginx,
  servido por Go): imágenes de entidades inactivas y archivos `autenticado`/`propietario`.

## Seguridad

- Reglas y decisiones: [SECURITY.md](SECURITY.md).
- No hay credenciales por defecto. El primer admin se define con
  `BOOTSTRAP_ADMIN_USERNAME` / `BOOTSTRAP_ADMIN_PASSWORD` y debe cambiar la
  contraseña en el primer login.
- `JWT_SECRET` es obligatorio (32+ caracteres); el backend no arranca sin él.
- Login con rate limit en nginx y en Go (`LOGIN_MAX_ATTEMPTS`, etc.).
- Antes de cerrar un cambio en `back/`: `go build ./...`, `go vet ./...` y
  `govulncheck ./...`. En `front/`: `npm audit --omit=dev` y `npm run build`.

## Documentación por flujo

Regla: un archivo por flujo en `docs/flujos/<flujo>.md`. `AGENTS.md` solo
resume y enlaza. No numerar archivos; el orden se controla con esta tabla.

| Flujo                         | Estado      | Documento                                                                |
| ----------------------------- | ----------- | ------------------------------------------------------------------------ |
| Login                         | Documentado | [docs/flujos/login.md](docs/flujos/login.md)                             |
| Logout y revocación           | Documentado | [docs/flujos/logout.md](docs/flujos/logout.md)                           |
| Contador de sesión            | Documentado | [docs/flujos/sesion-contador.md](docs/flujos/sesion-contador.md)         |
| Usuarios y permisos           | Documentado | [docs/flujos/usuarios-permisos.md](docs/flujos/usuarios-permisos.md)     |
| Mensajes pendientes (SSE)     | Documentado | [docs/flujos/mensajes-sse.md](docs/flujos/mensajes-sse.md)               |
| Archivos e imágenes           | Documentado | [docs/flujos/archivos.md](docs/flujos/archivos.md)                       |
| Buses                         | Documentado | [docs/flujos/buses.md](docs/flujos/buses.md)                             |
| Tienda (categorías/productos) | Documentado | [docs/flujos/tienda.md](docs/flujos/tienda.md)                           |
| Ventas y reportes             | Documentado | [docs/flujos/reportes.md](docs/flujos/reportes.md)                       |
| Instalaciones / mapas         | Documentado | [docs/flujos/instalaciones-mapas.md](docs/flujos/instalaciones-mapas.md) |
| Laboratorio de consultas      | Documentado | [docs/flujos/laboratorio.md](docs/flujos/laboratorio.md)                 |

## Diagramas de flujo

Los diagramas visuales de negocio viven en `front/public/flujos/` y se abren
desde el sidebar en **Flujos** (`/dashboard/flujos`). El índice con HTML, PNG,
GIF y fuentes está en
[docs/diagramas/README.md](docs/diagramas/README.md).

## Convenciones de trabajo

- Antes de dar por terminado un cambio en `back/`, ejecutar `go build ./...` y
  `go vet ./...`.
- No hacer commit ni push salvo que se pida explícitamente.
- No guardar secretos en el repositorio (`.env` está ignorado).
