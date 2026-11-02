# Mensajes pendientes y SSE

Este documento tiene dos partes:

1. **SSE como concepto reutilizable** (el núcleo del módulo).
2. **Su implementación concreta** en mensajes pendientes.

La idea es que, al clonar el template, puedas eliminar o reutilizar el módulo sin
arrastrar dependencias raras.

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

- Un solo tipo de mensaje: `SSEEvent { type, data }`.
- El `type` viaja **dentro** del JSON, no como campo `event:` de SSE.
- Autenticación por cookie HttpOnly: el frontend usa
  `new EventSource(url, { withCredentials: true })`.
- La ruta SSE vive bajo `AuthMiddleware("permiso")`, igual que el resto de la API.

## Piezas reutilizables vs específicas

| Pieza                              | Dónde vive                                | ¿Reutilizable? |
| ---------------------------------- | ----------------------------------------- | -------------- |
| `SSEHub`                           | `back/api/handlers/sse.go`                | Sí, genérico   |
| `SSEService`                       | `front/src/app/services/sse.service.ts`   | Sí, genérico   |
| `ActivosStore`                     | `back/infrastructure/cache/activos.go`    | No, es de mensajes |
| `MensajeHandler.Events`            | `back/api/handlers/mensaje.go`            | No, endpoint de mensajes |
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
  memoria (`ActivosStore`) y se sincronizan por SSE a todos los clientes.
- **Pasadas**: mensajes `finalizado`. Se consultan a la BD con paginación normal.
- Cuando alguien crea, asigna o finaliza un mensaje, la BD se actualiza y el
  servidor emite un evento a **todos** los clientes conectados.

## Backend

### Ruta

`GET /mensajes_pendientes/events` bajo `AuthMiddleware("mensajes_pendientes")`
(`back/api/router.go:69`). Solo pueden conectarse usuarios con ese permiso.

### Handler `Events`

`back/api/handlers/mensaje.go:25`:

1. Verifica que `ResponseWriter` soporte `http.Flusher`.
2. Escribe cabeceras `text/event-stream`, `no-cache`, `keep-alive`.
3. Envía un evento `current` con el snapshot completo de `ActivosStore`.
4. Registra el cliente en `SSEHub`.
5. Itera leyendo eventos del canal y escribiendo `data: <json>\n\n`.
6. Al cortarse el contexto (cliente desconectado), sale y se desregistra.

### `SSEHub`

`back/api/handlers/sse.go`:

- `clients map[chan SSEEvent]struct{}`.
- `Register()` crea un canal **bufferizado de 64**.
- `Broadcast()` envía a todos los canales con `select` no bloqueante: si un
  cliente está lento y su buffer está lleno, **se descarta el evento** para ese
  cliente.
- `Unregister()` elimina y cierra el canal.

No hay identidad por usuario ni canales por sala: **todos los conectados reciben
todos los eventos**.

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

Payloads:

```
data: {"type":"current","data":[ ... ]}

data: {"type":"create","data":{ ... }}

data: {"type":"update","data":{ ... }}

data: {"type":"delete","data":{"id":12}}
```

### Consultas

- `GET /mensajes_pendientes` devuelve `ActivosStore.GetAll()` (no consulta BD).
- `GET /mensajes_pendientes/{id}` busca primero en la caché y, si no está, en BD.
- `GET /mensajes_pendientes/pasadas` pagina en BD los `finalizado`.

## Frontend

### `SSEService`

`front/src/app/services/sse.service.ts`:

- Cierra cualquier conexión previa y abre una nueva con
  `EventSource(API_URL + path, { withCredentials: true })`.
- `onmessage`: parsea JSON y emite `{ type, data }`.
- `onerror`: emite error y cierra la conexión.
- `disconnect()`: cierra y limpia la referencia.

### `MensajesStoreService`

`front/src/app/services/mensajes-store.service.ts`:

- `init()` evita conexiones duplicadas (`if (this.sseSub) return`).
- Conecta a `/mensajes_pendientes/events`.
- Aplica los eventos al signal `activos`:
  - `current`: reemplaza la lista completa.
  - `create`: inserta al inicio.
  - `update`: reemplaza el mensaje por `id`.
  - `delete`: lo quita por `id`.
- `onerror`: limpia la suscripción y reintenta en 3 s.
- `disconnect()`: cancela la suscripción, cierra el `EventSource` y vacía la lista.

### Ciclo de vida

`DashboardLayoutComponent` llama a `MensajesStoreService.init()` en su
constructor (`front/src/app/components/dashboard/layout/layout.ts`). Por eso:

- La conexión SSE se abre al entrar a **cualquier** ruta del dashboard, no solo
  a la página de mensajes.
- Se mantiene mientras el usuario navega por el dashboard.
- Se cierra en `SidebarComponent.logout()` (ver [logout.md](./logout.md)).

### Pantalla de mensajes

`front/src/app/components/dashboard/mensajes/list.ts`:

- **Activas**: usa el store global. Los filtros (`estado`, `teléfono`) y la
  paginación se resuelven **en cliente** sobre la lista completa del store. No
  se vuelve a llamar a la API al filtrar.
- **Pasadas**: usa `GET /pasadas` con `offset`/`limit`; paginación de servidor.
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
- Actualización: el servidor hace broadcast; todos los clientes se enteran.
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
   reintenta cada 3 s. La redirección a login la provoca el interceptor de las
   peticiones HTTP normales, no el SSE. Al hacer logout se llama a
   `disconnect()`.
7. **Sin filtrado por usuario**: el hub no conoce identidad. Todos los usuarios
   con permiso `mensajes_pendientes` reciben todos los eventos. Si el contenido
   es sensible, habría que filtrar por cliente o crear canales por usuario/sala.
8. **nginx**: requiere `proxy_buffering off` (ya está en `deploy/nginx.conf`) y un
   `proxy_read_timeout` amplio para que el stream no se corte.

---

# Parte 5 — Cómo agregar SSE a otro módulo

1. **Backend**
   - Registrar `GET /<modulo>/events` bajo `AuthMiddleware("permiso")`.
   - Enviar un snapshot inicial (`type: "current"`).
   - Registrar el canal en `SSEHub` y transmitir cambios.
   - Tras cada mutación: actualizar BD y hacer `Broadcast`.
2. **Frontend**
   - Crear un store/servicio que use `SSEService.connect('/<modulo>/events')`.
   - Manejar `current`, `create`, `update`, `delete` (o los eventos del módulo).
   - Inicializar en el layout del dashboard si debe vivir en toda la sesión, o en
     el componente si es efímero.
   - Desconectar en logout.
3. **Documentación**
   - Añadir el flujo a `AGENTS.md` y su archivo en `docs/flujos/`.

---

## Archivos involucrados

| Capa     | Archivo                                                          |
| -------- | ---------------------------------------------------------------- |
| Backend  | `back/api/router.go`                                             |
| Backend  | `back/api/handlers/mensaje.go`                                   |
| Backend  | `back/api/handlers/sse.go`                                       |
| Backend  | `back/infrastructure/cache/activos.go`                           |
| Backend  | `back/cmd/server/main.go`                                        |
| Backend  | `back/infrastructure/database/mensajes.go`                       |
| Frontend | `front/src/app/services/sse.service.ts`                          |
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
