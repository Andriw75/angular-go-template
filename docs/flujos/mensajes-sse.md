# Mensajes pendientes y SSE

Este documento tiene dos partes:

1. **SSE como concepto reutilizable** (el núcleo del módulo).
2. **Su implementación concreta** en mensajes pendientes.

La idea es que, al clonar el template, puedas eliminar o reutilizar el módulo sin
arrastrar dependencias raras.

> **Pendiente de actualización visual:** el diagrama HTML/GIF/PNG de este flujo
> todavía muestra la conexión directa anterior (`/mensajes_pendientes/events`).
> Debe regenerarse para reflejar el SSE global filtrado por permisos descrito
> aquí.

---

# Parte 1 — SSE como concepto reutilizable

## Qué es SSE

Server-Sent Events (SSE) es un canal **unidireccional servidor → cliente** sobre
una conexión HTTP normal. El servidor responde con
`Content-Type: text/event-stream` y va escribiendo líneas con este formato:

```
data: {"type":"...","data":...}

```

El navegador las consume con `EventSource`. Las escrituras del cliente siguen
haciéndose por REST normal: **SSE solo notifica, no reemplaza a la API**.

| Mecanismo    | Dirección        | Uso típico                                  |
| ------------ | ---------------- | ------------------------------------------- |
| HTTP normal  | Cliente → Servidor | CRUD, búsquedas, paginación                |
| SSE          | Servidor → Cliente | Avisar cambios en vivo a todos los clientes |
| WebSocket    | Bidireccional    | Chat, colaboración en tiempo real           |
| Polling      | Cliente → Servidor | Alternativa simple si SSE no es viable     |

## Contrato de este template

- Un único sobre de evento:
  ```json
  { "module": "...", "name": "...", "type": "current|create|update|delete", "data": ... }
  ```
  - `module`: módulo de negocio (`mensajes_pendientes`).
  - `name`: recurso o canal dentro del módulo (`activos`). Un módulo puede
    emitir varios `name`.
  - `type`: qué ocurrió con ese recurso.
  - `data`: payload.
- Todo viaja **dentro** del JSON, no como campo `event:` de SSE.
- Hay **un solo** `GET /sse/events` para toda la aplicación.
- Autenticación por cookie HttpOnly: el frontend usa
  `new EventSource(url, { withCredentials: true })`.
- La ruta vive bajo `AuthMiddleware()` (solo autenticación). El handler calcula
  los canales permitidos:
  - Si el canal no tiene permiso (`Permission == ""`), se concede a cualquier
    usuario autenticado.
  - Si tiene permiso, se valida contra `claims.Permisos`.
  - Si no queda ningún canal, responde `403`.
- Canales actuales:
  - `mensajes_pendientes/activos` → permiso `mensajes_pendientes`.
  - `laboratorio/tareas` → solo autenticado (cualquier usuario logueado).

## Piezas reutilizables vs específicas

| Pieza                              | Dónde vive                                | ¿Reutilizable? |
| ---------------------------------- | ----------------------------------------- | -------------- |
| `SSEHub`                           | `back/api/handlers/sse.go`                | Sí, genérico   |
| `SSEHandler.Events`                | `back/api/handlers/sse_handler.go`        | Sí, SSE global |
| `sseChannels`                      | `back/api/handlers/sse_handler.go`        | Configurable por módulo |
| `SSEService`                       | `front/src/app/services/sse.service.ts`   | Sí, genérico   |
| `SSE_CHANNELS`                     | `front/src/app/services/sse-channels.ts`  | Configurable por módulo |
| `ActivosStore`                     | `back/infrastructure/cache/activos.go`    | No, es de mensajes |
| `MensajesStoreService`             | `front/src/app/services/mensajes-store.service.ts` | No, estado de mensajes |
| Eventos `current/create/update/delete` | Backend + store de mensajes           | No, contrato del módulo |

Recomendación para un proyecto real: si SSE se va a usar en más de un módulo,
mover `SSEHub` a `back/infrastructure/sse` (o similar) para que no dependa del
paquete `handlers`. Si solo se usa en mensajes, dejarlo donde está.

## Cómo clonar/adaptar este template

- **El proyecto no necesita mensajes ni tiempo real**: eliminar el módulo
  completo (frontend de mensajes, `mensajes-store.service.ts`,
  `MensajeHandler`, `SSEHub`, `ActivosStore`, ruta `/mensajes_pendientes`,
  entrada del menú y permiso). No queda nada acoplado al resto.
- **Otro módulo necesita tiempo real**: reutilizar `SSEHub` y `SSEService`,
  crear un store/endpoint propio y definir sus propios `type`. No reutilizar
  `ActivosStore` (es específico de mensajes).
- **No hay bus global de eventos de negocio**: el hub solo transporta; cada
  módulo decide qué emite y qué hace el cliente al recibirlo.

---

# Parte 2 — Implementación en mensajes pendientes

## Resumen del módulo

- **Activas**: mensajes en estado `pendiente` o `asignado`. Viven en una caché en
  memoria (`ActivosStore`) y se sincronizan por SSE a todos los clientes con
  permiso sobre el canal.
- **Pasadas**: mensajes `finalizado`. Se consultan a la BD con paginación normal.
- Cuando alguien crea, asigna o finaliza un mensaje, la BD se actualiza y el
  servidor emite un evento a **todos** los clientes conectados.

## Backend

### Ruta

`GET /sse/events` bajo `AuthMiddleware()` (`back/api/router.go`). Es la única
ruta SSE de la aplicación.

### Handler `Events`

`back/api/handlers/sse_handler.go`:

1. Verifica que `ResponseWriter` soporte `http.Flusher`.
2. Lee `claims.Permisos` del usuario autenticado.
3. Calcula los canales permitidos desde `sseChannels` (module + name + permiso).
4. Si no hay ningún canal permitido → `403` y no abre el stream.
5. Escribe cabeceras `text/event-stream`, `no-cache`, `keep-alive`.
6. Registra el cliente en `SSEHub` con sus canales permitidos.
7. Envía un evento `type: "current"` por cada canal permitido con su snapshot.
8. Itera leyendo eventos del canal y escribiendo `data: <json>\n\n`.
9. Al cortarse el contexto (cliente desconectado), sale y se desregistra.

### `SSEHub`

`back/api/handlers/sse.go`:

- `clients map[*sseClient]struct{}`; cada cliente guarda sus canales
  `(module, name)` permitidos.
- `Register(channels)` crea un canal **bufferizado de 64** y registra el set de
  canales.
- `Broadcast(module, name, event)` envía solo a los clientes suscritos a ese
  canal, con `select` no bloqueante: si un cliente está lento y su buffer está
  lleno, **se descarta el evento** para ese cliente.
- `Unregister()` elimina y cierra el canal.

El hub no filtra por identidad de usuario: filtra por canales permitidos, que el
handler calculó a partir de los permisos del JWT al conectar.

### `ActivosStore`

`back/infrastructure/cache/activos.go`:

- Caché en memoria de mensajes no finalizados.
- Se siembra al arrancar desde la BD (`seedActivos` en
  `back/cmd/server/main.go:119`).
- `GetAll()` ordena por `hora_solicitada` descendente.
- Es por proceso. No se comparte entre instancias.

### Mutaciones y eventos

| Acción             | BD (MensajeStore) | Caché (`ActivosStore`) | Evento SSE        |
| ------------------ | ----------------- | ---------------------- | ----------------- |
| Crear mensaje      | `Create`          | `Set`                  | `create`          |
| Asignar / actualizar | `Update`        | `Set`                  | `update`          |
| Finalizar          | `Update`          | `Delete`               | `delete` (`{id}`) |
| Eliminar           | `Delete`          | `Delete`               | `delete` (`{id}`) |

Todos los eventos de mensajes usan `module: "mensajes_pendientes"` y
`name: "activos"`. Payloads:

```
data: {"module":"mensajes_pendientes","name":"activos","type":"current","data":[ ... ]}

data: {"module":"mensajes_pendientes","name":"activos","type":"create","data":{ ... }}

data: {"module":"mensajes_pendientes","name":"activos","type":"update","data":{ ... }}

data: {"module":"mensajes_pendientes","name":"activos","type":"delete","data":{"id":12}}
```

### Consultas

- `GET /mensajes_pendientes` devuelve `ActivosStore.GetAll()` (no consulta BD).
- `GET /mensajes_pendientes/{id}` busca primero en la caché y, si no está, en BD.
- `GET /mensajes_pendientes/pasadas` pagina en BD los `finalizado`.

## Frontend

### `SSE_CHANNELS` y `hasAnySseAccess`

`front/src/app/services/sse-channels.ts`:

- Lista de canales SSE con `module`, `name` y `permission`.
- Debe coincidir con `sseChannels` del backend.
- `permission: null` o vacío significa **solo autenticado**: cualquier usuario
  con sesión puede recibir ese canal.
- `hasAnySseAccess(user)` devuelve `true` si hay un canal solo-autenticado o si
  el usuario tiene el permiso de algún canal.
- Canales actuales:
  - `mensajes_pendientes/activos` → permiso `mensajes_pendientes`.
  - `laboratorio/tareas` → solo autenticado.

### `SSEService` (conexión compartida)

`front/src/app/services/sse.service.ts`:

- Una sola conexión `EventSource(API_URL + '/sse/events', { withCredentials: true })`
  para toda la aplicación.
- `events$`: stream compartido con `{ module, name, type, data }`.
- `setEnabled(enabled)`: abre o cierra la conexión según `hasAnySseAccess`.
- `onmessage`: parsea JSON y emite por `events$`.
- `onerror`: cierra y reintenta con backoff (3 s → 30 s).
- `disconnect()`: cierre forzado para logout/expiración.

### Consumidores

- `MensajesStoreService`:
  - `init()` solo si el usuario tiene el permiso `mensajes_pendientes`.
  - Se suscribe a `sse.events$` y filtra `mensajes_pendientes/activos`.
  - Aplica `current/create/update/delete` (con deduplicación por `id`).
  - No abre ni cierra la conexión global; solo su propia suscripción.
- `LaboratorioComponent`:
  - Se suscribe a `sse.events$` y filtra `laboratorio/tareas`.
  - Hidrata la grilla con `GET /lab/slow-queries` y aplica
    `current/create/update`.
  - Se desuscribe al destruir el componente; las tareas en segundo plano siguen
    vivas y se recuperan al volver a entrar.

### Ciclo de vida

`DashboardLayoutComponent` usa un `effect` sobre `auth.user()`
(`front/src/app/components/dashboard/layout/layout.ts`):

- `sse.setEnabled(hasAnySseAccess(user))`.
- Si el usuario tiene `mensajes_pendientes` → `mensajes.init()`; si no,
  `mensajes.disconnect()`.
- En expiración/logout: `mensajes.disconnect()` y `sse.disconnect()`.
- Como el canal `laboratorio/tareas` es solo-autenticado, cualquier usuario
  logueado abre la conexión global y recibe los eventos de laboratorio. Los
  mensajes siguen filtrados por su permiso.

### Pantalla de mensajes

`front/src/app/components/dashboard/mensajes/list.ts`:

- **Activas**: usa el store global. Los filtros (`estado`, `teléfono`) y la
  paginación se resuelven **en cliente** sobre la lista completa del store. No
  se vuelve a llamar a la API al filtrar.
- **Pasadas**: usa `GET /pasadas` con `offset`/`limit`; paginación de servidor.
  El listado devuelve `has_more` (sin `total`); el total se consulta en
  `GET /pasadas/count` y se cachea hasta que cambien los filtros.
- Crear, asignar, finalizar y eliminar usan REST normal; el cambio llega a los
  demás clientes por SSE.

### Secuencia de una creación

```
Usuario A          Frontend A        API Go                    Frontend B/C
   |                   |                |                          |
   |-- Crear --------->|                |                          |
   |                   |-- POST ------->|                          |
   |                   |                |-- INSERT BD              |
   |                   |                |-- ActivosStore.Set       |
   |                   |                |-- SSEHub.Broadcast ----->| (create)
   |                   |<-- 201 --------|                          |
   |                   | (no depende del SSE)                     |
```

---

# Parte 3 — Dos patrones de actualización

El template combina dos formas de refrescar datos. Conviene distinguirlas al
adaptar o crear módulos.

## Patrón A: estado compartido en vivo (SSE)

- Fuente: caché en memoria (`ActivosStore`).
- Inicio: al entrar al dashboard (layout).
- Actualización: el servidor hace broadcast; se enteran los clientes suscritos
  al canal.
- Filtros y paginación: en cliente, sobre el estado completo.
- Ejemplo: mensajes pendientes → activas.

## Patrón B: consulta bajo demanda (paginación y filtros)

- Fuente: BD directa.
- Inicio: al montar el componente.
- Actualización: solo cuando el usuario busca, filtra, cambia de página o hace
  un CRUD.
- Filtros y paginación: en servidor (`q`, `tipo`, `categoria_id`, `offset`,
  `limit`).
- Ejemplos: buses, categorías, productos, archivos, mensajes → pasadas.

En el patrón B, `search()` normalmente resetea a página 1 y vuelve a pedir
`count` + página actual (`forkJoin`); `onPageChange()` solo pide la página. **No
se refresca toda la tabla ni se notifica a otros usuarios.** Cada usuario ve el
resultado de su propia consulta hasta que vuelva a buscar o refrescar.

| Aspecto            | SSE (mensajes activas)          | Bajo demanda (buses, tienda, archivos, pasadas) |
| ------------------ | ------------------------------- | ----------------------------------------------- |
| Fuente             | Caché en memoria                | Base de datos                                   |
| Inicio             | Entrar al dashboard             | Montar el componente                            |
| Actualización      | Broadcast inmediato             | Búsqueda / página / CRUD / refresh              |
| Filtros            | Cliente                         | Servidor                                        |
| Paginación         | Cliente                         | Servidor                                        |
| Multiusuario       | Todos ven lo mismo en vivo      | Cada uno ve su consulta                         |
| Complejidad        | Mayor (conexión, hub, caché)    | Menor                                           |

## Cuándo usar cada uno

- **SSE**: colas operativas, estados compartidos, paneles donde varios usuarios
  necesitan ver el mismo cambio al instante.
- **Bajo demanda**: catálogos, CRUD, históricos, búsquedas con filtros; es más
  simple y no mantiene estado en el servidor.
- **Polling**: si SSE no es viable por proxy/infraestructura y la frecuencia es
  baja.
- **WebSocket**: solo si necesitas comunicación bidireccional en tiempo real.

---

# Parte 4 — Limitaciones y decisiones para producción

1. **Un solo proceso**: `SSEHub` y `ActivosStore` viven en memoria. Con varias
   instancias, cada una tiene sus clientes y su caché; un evento emitido en una
   no llega a las demás. Para escalar hace falta un broker (Redis pub/sub) o
   sticky sessions.
2. **Sin heartbeat ni `Last-Event-ID`**: si el proxy corta la conexión, el cliente
   reintenta a los 3 s y recibe un snapshot `current` completo. No hay replay de
   eventos perdidos.
3. **Ventana entre snapshot y registro**: `Events` envía `current` y luego
   registra el canal. Un evento creado justo en ese instante puede perderse hasta
   la próxima reconexión. Es una ventana muy pequeña.
4. **Cliente lento**: si su buffer de 64 se llena, los eventos se descartan para
   ese cliente; se resincroniza al reconectar.
5. **Cambios directos en BD**: si se modifica `mensajes_pendientes` por SQL, la
   caché y los clientes no se enteran. Todo cambio debe pasar por la API.
6. **Token expirado o revocado**: el `EventSource` recibe `401`, emite error y
   el store reintenta cada 3 s **solo si el usuario sigue teniendo algún permiso
   SSE**. La redirección a login la provoca el interceptor de las peticiones HTTP
   normales, no el SSE. Al hacer logout se llama a `disconnect()`.
7. **Filtrado por permisos/canal, no por usuario**: el hub conoce los canales
   `(module, name)` permitidos de cada cliente, calculados desde el JWT al
   conectar. Los canales con permiso llegan solo a quienes lo tienen; los
   canales solo-autenticados (`laboratorio/tareas`) llegan a cualquier usuario
   logueado. No hay filtrado por propietario ni por sala. Si el contenido es
   sensible, habría que añadir canales o filtros por usuario.
8. **nginx**: requiere `proxy_buffering off` (ya está en `deploy/nginx.conf`) y un
   `proxy_read_timeout` amplio para que el stream no se corte.

---

# Parte 5 — Cómo agregar SSE a otro módulo

No se crea otra ruta: todo viaja por el único `GET /sse/events`.

1. **Backend**
   - Añadir el canal a `sseChannels` en `back/api/handlers/sse_handler.go`:
     `{Module: "<modulo>", Name: "<recurso>", Permission: "<permiso>"}`.
     Usa `Permission: ""` si el canal es solo para usuarios autenticados.
   - En `writeCurrent`, enviar el snapshot inicial con
     `type: "current"` para ese `(module, name)`.
   - Tras cada mutación: actualizar BD y llamar
     `SSEHub.Broadcast("<modulo>", "<recurso>", SSEEvent{...})`.
2. **Frontend**
   - Añadir el canal a `SSE_CHANNELS` en
     `front/src/app/services/sse-channels.ts` con su permiso o `permission: null`
     si es solo-autenticado.
   - Crear un store/servicio que se suscriba a `SSEService.events$` y filtre por
     `module` y `name`.
   - Manejar `current`, `create`, `update`, `delete`.
   - Inicializar en el layout del dashboard si debe vivir en toda la sesión, o en
     el componente si es efímero.
   - No cerrar la conexión global desde el store; solo desuscribirse.
   - El `DashboardLayoutComponent` decide `sse.setEnabled(hasAnySseAccess(user))`.
3. **Documentación**
   - Añadir el flujo a `AGENTS.md` y su archivo en `docs/flujos/`.

---

## Archivos involucrados

| Capa     | Archivo                                                          |
| -------- | ---------------------------------------------------------------- |
| Backend  | `back/api/router.go`                                             |
| Backend  | `back/api/handlers/mensaje.go`                                   |
| Backend  | `back/api/handlers/sse.go`                                       |
| Backend  | `back/api/handlers/sse_handler.go`                               |
| Backend  | `back/infrastructure/cache/activos.go`                           |
| Backend  | `back/cmd/server/main.go`                                        |
| Backend  | `back/infrastructure/database/mensajes.go`                       |
| Frontend | `front/src/app/services/sse.service.ts`                          |
| Frontend | `front/src/app/services/sse-channels.ts`                         |
| Frontend | `front/src/app/services/mensajes-store.service.ts`               |
| Frontend | `front/src/app/services/mensajes.service.ts`                     |
| Frontend | `front/src/app/components/dashboard/layout/layout.ts`            |
| Frontend | `front/src/app/components/dashboard/mensajes/list.ts`            |
| Frontend | `front/src/app/components/dashboard/mensajes/list.html`          |
| Frontend | `front/src/app/models/mensaje.ts`                                |
| Deploy   | `deploy/nginx.conf`                                              |

## Ver también

- [Login](./login.md): la cookie HttpOnly que autentica el `EventSource`.
- [Logout y revocación](./logout.md): desconexión del SSE y revocación del token.
- [Usuarios y permisos](./usuarios-permisos.md): permiso
  `mensajes_pendientes`.
