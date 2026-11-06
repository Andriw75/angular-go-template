# Laboratorio de consultas largas

## Resumen

Herramienta de desarrollo para simular consultas largas contra PostgreSQL y
cancelarlas terminando la consulta en el servidor (`pg_cancel_backend`).

Es un recurso **compartido por diseño**: cualquier usuario autenticado puede
crear tareas, ver las de todos y cancelarlas. No se filtra por dueño ni requiere
un permiso adicional; los límites de concurrencia son la única protección.

Tiene dos modos:

| Modo        | Ciclo de vida                                             | Cancelación                         |
| ----------- | --------------------------------------------------------- | ----------------------------------- |
| `directa`   | Dentro de la petición HTTP (`r.Context()`).                | Angular aborta con `unsubscribe()`. |
| `background`| Tarea en goroutine + registro en memoria + SSE.            | Endpoint de cancelación.            |

No usa variables de entorno; los límites están fijos en código.

## Límites

| Límite                              | Valor       |
| ----------------------------------- | ----------- |
| Duración máxima (`pg_sleep`)        | `600` s     |
| Filas máximas (`sleep`/`cpu`)       | `2000000`   |
| Tareas asíncronas globales          | `10`        |
| Tareas asíncronas por usuario       | `5`         |
| TTL de tareas terminadas en memoria | `5` minutos |

Solo funciona con `DB_TYPE=postgres`; con SQLite la API responde `501`.

## Consultas

| Modo    | Consulta                                                                 |
| ------- | ------------------------------------------------------------------------ |
| `sleep` | `pg_sleep(n)` una vez y un `count(*)` acotado de `generate_series`.       |
| `cpu`   | `count(*)` sobre `generate_series` con `md5(g::text)`, acotado por filas. |

Ambas son lecturas. La conexión fija `default_transaction_read_only = on`,
`work_mem = '16MB'`, `application_name = 'template-lab'` y un
`statement_timeout` de seguridad; al terminar se restauran esos valores antes de
devolver la conexión al pool.

## API

| Método | Ruta                            | Descripción                                      |
| ------ | ------------------------------- | ------------------------------------------------ |
| `POST` | `/lab/run`                      | Modo directo. La petición dura hasta terminar o ser abortada. |
| `POST` | `/lab/slow-queries`             | Crea una tarea en segundo plano.                 |
| `GET`  | `/lab/slow-queries`             | Lista todas las tareas (todos los usuarios las ven). |
| `POST` | `/lab/slow-queries/{id}/cancel` | Detiene una tarea.                               |

Body común: `{ "modo": "sleep|cpu", "segundos": 30, "filas": 100000 }`.

Estados: `ejecutando → cancelando → cancelada | completada | error`.

## Modo directo

`POST /lab/run` ejecuta la consulta en la misma petición HTTP. Si Angular hace
`unsubscribe()` (botón Cancelar, navegación o refresh), la petición se aborta,
`r.Context()` se cancela y el backend ejecuta `pg_cancel_backend(pid)`. No queda
ninguna tarea huérfana ni nada que rastrear. El tradeoff es que no hay historial:
al refrescar, la operación simplemente se cancela.

## Modo en segundo plano

`POST /lab/slow-queries` registra una tarea con su propio `context` y devuelve
`201` de inmediato. El frontend puede iniciar varias tareas. Cada cambio de
estado se publica por el SSE global en el canal:

```json
{ "module": "laboratorio", "name": "tareas", "type": "create|update|current", "data": { ... } }
```

El canal `laboratorio/tareas` es **solo-autenticado**: cualquier usuario con
sesión lo recibe. Todos los usuarios ven todas las tareas y cualquiera puede
detenerlas, porque es un laboratorio de desarrollo.

Al montar o refrescar la página, la grilla se hidrata con
`GET /lab/slow-queries`; los cambios posteriores llegan por SSE. Así el
seguimiento no se pierde al refrescar y la tarea sigue siendo detenible.

## Cancelación y limpieza

1. La consulta usa una conexión dedicada (`db.Conn`).
2. `pg_cancel_backend(pid)` se ejecuta en otra conexión del pool para abortar la
   consulta en el servidor.
3. Se cancela el `context` y se espera a que la goroutine cierre la conexión.
4. Si no termina en 10 s, se intenta `pg_terminate_backend(pid)`.
5. `defer conn.Close()` y el registro se limpian con TTL (5 min).
6. Al apagar el servidor se ejecuta `CancelAll` antes de `srv.Shutdown`.

> Con `database/sql` + pgx v5, cancelar el `context` cierra la conexión por
> defecto, pero PostgreSQL puede tardar en notar el socket roto. Por eso la
> cancelación explícita con `pg_cancel_backend` es la que garantiza que la
> consulta deje de consumir recursos.

## SSE compartido

El laboratorio usa el mismo `GET /sse/events` que mensajes pendientes:

- `SSEService` mantiene una sola conexión `EventSource` y un `events$` compartido.
- `MensajesStoreService` filtra `mensajes_pendientes/activos`.
- `LaboratorioComponent` filtra `laboratorio/tareas`.
- El layout abre/cierra la conexión con `setEnabled(hasAnySseAccess(user))`.
- `SSE_CHANNELS` incluye canales con permiso y canales solo-autenticados
  (`permission: null`).
- La reconexión con backoff vive en `SSEService`.

## Frontend

`/dashboard/laboratorio` (sidebar **Laboratorio**, visible para cualquier usuario
autenticado):

- Pestañas **Petición directa** y **Tareas en segundo plano**.
- Modo directo: botón Ejecutar/Cancelar; el cancelar hace `unsubscribe()`.
  Al refrescar, la petición se aborta y la consulta se cancela.
- Modo segundo plano: botón Iniciar (permite varias), grilla con estado, PID,
  duración y botón **Detener** con confirmación.
- Sin polling: la grilla se actualiza por SSE.
- `ngOnDestroy`: cancela la petición directa y se desuscribe del SSE; no detiene
  las tareas en segundo plano, que siguen visibles al volver.

## Verificación

Con una consulta activa:

```sql
SELECT pid, application_name, state, now() - query_start AS elapsed, query
FROM pg_stat_activity
WHERE application_name = 'template-lab';
```

- Directo: iniciar 60 s, refrescar la página; la fila debe desaparecer y no debe
  quedar ningún `pg_sleep`.
- Segundo plano: iniciar 3 tareas, ver la grilla, detener una con confirmación y
  comprobar que el estado se actualiza por SSE sin polling.
- Refrescar en segundo plano: la grilla se repuebla y las tareas siguen siendo
  detenibles.
- Network: un solo `GET /sse/events` aunque estén abiertos mensajes y laboratorio.

## Archivos involucrados

| Capa     | Archivo                                                              |
| -------- | -------------------------------------------------------------------- |
| Backend  | `back/store/store.go`                                                |
| Backend  | `back/infrastructure/database/lab.go`                                |
| Backend  | `back/api/handlers/lab.go`                                           |
| Backend  | `back/api/handlers/sse_handler.go`                                   |
| Backend  | `back/api/router.go`                                                 |
| Backend  | `back/cmd/server/main.go`                                            |
| Frontend | `front/src/app/models/lab.ts`                                        |
| Frontend | `front/src/app/services/lab.service.ts`                              |
| Frontend | `front/src/app/services/sse.service.ts`                              |
| Frontend | `front/src/app/services/sse-channels.ts`                             |
| Frontend | `front/src/app/components/dashboard/laboratorio/laboratorio.ts`      |
| Frontend | `front/src/app/components/dashboard/layout/layout.ts`                |
| Frontend | `front/src/app/app.routes.ts`                                        |
| Frontend | `front/src/app/components/dashboard/sidebar/menu.config.ts`          |
