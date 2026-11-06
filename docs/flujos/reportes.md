# Reportes, PDF/Excel y documentos con correlativo

## Resumen

El sistema permite generar reportes en **dos motores**:

- **Navegador**: el frontend transforma el JSON en PDF (`jsPDF` + `jspdf-autotable`) o Excel (`ExcelJS`).
- **Servidor**: la API genera PDF (`go-pdf/fpdf`) o Excel (`excelize/v2`).

Y dos modos:

- **Reporte dinámico**: no se persiste; se regenera con los datos actuales.
- **Documento**: se persiste con correlativo, snapshot e idempotencia.

Fuentes de datos:

- **Buses** para reportes tabulares.
- **Ventas** para documentos con montos, ítems e impuestos.

## Permisos

Nuevos permisos sembrados:

| Permiso | Uso |
| --- | --- |
| `reportes` | Acceso al submódulo Reportes y endpoints de reportes |
| `ventas` | CRUD de ventas |
| `documentos` | Historial, descarga, anulación y generación de documentos |

El seed reconcilia al usuario definido en `BOOTSTRAP_ADMIN_USERNAME` para que reciba los permisos nuevos en instalaciones existentes.

## Contrato de datos

```
GET /reportes/buses?q=&tipo=&activo=&desde=&hasta=&modo=actual|generado&cantidad=N
GET /reportes/buses/count?...
GET /reportes/ventas?q=&serie=&estado=&desde=&hasta=
GET /reportes/ventas/count?...
```

`ReporteBuses` incluye `resumen`, `por_tipo` y `data`.
`ReporteVentas` incluye `resumen` y `data`.

El mismo JSON alimenta los cuatro formatos (frontend/backend × PDF/Excel).

## Datos actuales vs autogenerados

La pantalla Reportes permite elegir:

- **Datos actuales**: usa la base de datos.
- **Datos autogenerados**: usa un entero `cantidad` y genera esa cantidad de
  buses sintéticos en el servidor.

Reglas:

- Solo está disponible para **Buses**; en Ventas se deshabilita.
- `cantidad` se limita a **1000** registros por solicitud.
- El conteo devuelve directamente `cantidad` en modo generado.
- Los endpoints aceptan `modo=generado&cantidad=N`.
- Los filtros `tipo`, `activo`, `desde` y `hasta` condicionan los datos
  sintéticos; `q` se incorpora al nombre/descripción generada para que el filtro
  coincida.
- Los datos generados no se insertan en la tabla `buses`; viven en el JSON del
  reporte.
- Si se genera un **documento** con datos autogenerados, se guarda el snapshot y
  se usa la serie `GEN` (`GEN-2026-000001`) para no mezclar correlativos reales.
- El hash de idempotencia incluye `modo` y `cantidad`.

## Plantillas

| Código | Fuente | Formato | Características |
| --- | --- | --- | --- |
| `buses-a4-p` | Buses | A4 vertical | Cabecera/pie, página X/Y, resumen, tabla |
| `buses-a4-l` | Buses | A4 horizontal | Tabla ancha, paginación |
| `buses-mixto` | Buses | Mixto | Portada vertical + tabla horizontal + resumen vertical |
| `buses-dinamico` | Buses | Alto dinámico | Altura calculada según filas |
| `venta-factura` | Ventas | A4 vertical | Cliente, ítems, impuestos, totales, páginas |
| `venta-ticket` | Ventas | 80 mm dinámico | Ticket térmico, totales |
| `venta-nota` | Ventas | A4 vertical | Nota de venta |

Implementación:

- Backend `fpdf`: `AddPageFormat` para mixto, `NewCustom` para ticket/dinámico, `AliasNbPages` para “Página X de Y”.
- Frontend `jsPDF`: `addPage('a4','landscape')`, `addPage([80, alto])`, `autoTable`, cabecera/pie con `didDrawPage`.
- Excel: `excelize` / `ExcelJS` con estilos, merges, autofiltro, panel congelado y totales.

## Ventas

### Modelo

- `ventas`: cliente, documento, dirección, fecha, moneda, subtotal, descuento, impuesto, total, estado, serie, correlativo, código.
- `venta_items`: producto opcional, descripción, cantidad, precio unitario, descuento, impuesto %, subtotal, impuesto, total.
- Estados: `borrador`, `emitido`, `anulado`.

### Reglas

- El correlativo se asigna al **emitir**, no al crear el borrador.
- Formato: `FAC-2026-000001`, `TKT-2026-000001`, `NOT-2026-000001`.
- Los totales se calculan en el backend; no se confía en el cliente.
- No se modifica stock.
- Solo los borradores se editan/eliminan. Los emitidos se anulan.

## Documentos con correlativo

### Tablas

- `documentos`: tipo, serie, año, correlativo, código único, política, estado, plantilla, versión de plantilla, hash de parámetros, parámetros JSON, snapshot JSON, versión, usuario y fechas.
- `documento_archivos`: PDF/XLSX por documento y versión.
- `documento_secuencias`: último correlativo por tipo/serie/año.
- `documento_eventos`: auditoría (`creado`, `regenerado`, `anulado`).

### Idempotencia

```
hash = sha256(tipo + plantillaVersion + JSON canónico de parámetros)
```

- Unique `(tipo, parametros_hash)`.
- Si el documento existe y la política es `inmutable` → se devuelve el existente con `exists: true`.
- Si la política es `actualizable` → se regenera, incrementa `version` y se agregan nuevos archivos.

### Ciclo de vida

| Situación | Acción |
| --- | --- |
| Borrador | Editable/eliminable, sin correlativo |
| Emitir | Correlativo + snapshot + PDF + XLSX + evento |
| Mismo hash, inmutable | Devuelve existente |
| Mismo hash, actualizable | Nueva versión de archivos, mismo código |
| Error de generación | Rollback; no deja documento a medias |
| Archivo perdido | Regenerar desde snapshot con el mismo código |
| Datos cambiados en inmutable | Anular y emitir nuevo documento |
| Anular | Estado `anulado`, conserva código/archivos/evento |
| Eliminar | Solo borradores; emitidos se anulan |

## Jobs asíncronos

Sin límite de filas. La generación es síncrona hasta `REPORT_SYNC_TIMEOUT_SECONDS` (60 s). Si supera:

- Responde `202 { job_id }`.
- El trabajo continúa en segundo plano.
- El resultado se guarda temporalmente en memoria (TTL `JOB_TTL_MINUTES`, 30 min).
  Un ticker interno elimina los jobs vencidos aunque no se creen nuevos.
- El estado mutable se lee y escribe bajo mutex; los handlers consultan un
  `JobSnapshot` para evitar carreras con la goroutine de generación.
- El documento final se persiste en BD/disco.

Endpoints:

```
POST /documentos/buses
POST /documentos/ventas/{id}
GET  /documentos/jobs/{jobID}
GET  /documentos
GET  /documentos/{id}
GET  /documentos/{id}/file?formato=pdf|xlsx
POST /documentos/{id}/anular
```

El frontend hace polling cada 2 s y muestra el progreso. Los jobs son efímeros y
compartidos para quien tenga el permiso `documentos` (igual que los documentos).
Si el job expira o el proceso se reinicia, el job en memoria se pierde; el
documento ya persistido sigue disponible.

## UI

- **Reportes** (`/dashboard/reportes`): fuente Buses/Ventas, filtros, rango de fechas, botón **Contar**, plantilla, motor, formato, modo, política, generar, resultado e historial de documentos.
- **Ventas** (`/dashboard/ventas`): listado, creación/edición de borradores, emisión, anulación y eliminación de borradores.
- Menú: **Reportes** (`reportes`), **Tienda → Ventas** (`ventas`).

## Almacenamiento

- Documentos en `PUBLIC_DIR/archivos/documentos` (privado).
- Servidos por `GET /documentos/{id}/file` con autenticación y permiso `documentos`.
- nginx bloquea `/template/archivos`.

## Dependencias

Backend:

```bash
go get github.com/go-pdf/fpdf
go get github.com/xuri/excelize/v2
```

Frontend:

```bash
npm install jspdf jspdf-autotable exceljs
```

## Licencias

Todas las dependencias actuales y nuevas son permisivas (MIT, BSD, Apache, 0BSD). No hay GPL/AGPL. Ver `THIRD_PARTY_NOTICES.md`.

## Variables

```
REPORT_SYNC_TIMEOUT_SECONDS=60
JOB_TTL_MINUTES=30
```

## Archivos involucrados

| Capa | Archivo |
| --- | --- |
| Backend | `back/domain/venta.go`, `back/domain/documento.go` |
| Backend | `back/domain/inputs/venta.go`, `back/domain/outputs/venta.go`, `reporte.go`, `documento.go` |
| Backend | `back/infrastructure/database/ventas.go`, `documentos.go`, `schema.go` |
| Backend | `back/services/reportes.go`, `documentos.go`, `jobs.go` |
| Backend | `back/infrastructure/report/pdf.go`, `excel.go` |
| Backend | `back/api/handlers/venta.go`, `reporte.go`, `documento.go` |
| Backend | `back/api/router.go`, `back/store/store.go`, `back/cmd/server/main.go` |
| Frontend | `front/src/app/components/dashboard/reportes/*` |
| Frontend | `front/src/app/components/dashboard/ventas/*` |
| Frontend | `front/src/app/services/reportes.service.ts`, `report-export.service.ts`, `ventas.service.ts` |
| Frontend | `front/src/app/models/reporte.ts`, `venta.ts` |

## Ver también

- [Login](./login.md) y [Usuarios y permisos](./usuarios-permisos.md): permisos.
- [Archivos e imágenes](./archivos.md): almacenamiento privado.
- [Buses](./buses.md): fuente de reportes.
