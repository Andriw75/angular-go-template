package database

import "fmt"

func schema(d Dialect) []string {
	pk := d.AutoIncrementPK()
	flt := d.FloatType()

	return []string{
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS permissions (
			id %s,
			nombre TEXT NOT NULL UNIQUE,
			descripcion TEXT NOT NULL DEFAULT ''
		)`, pk),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS users (
			id %s,
			username TEXT NOT NULL UNIQUE,
			email TEXT NOT NULL,
			password TEXT NOT NULL,
			activo INTEGER NOT NULL DEFAULT 1,
			must_change_password INTEGER NOT NULL DEFAULT 0,
			creado_en TEXT NOT NULL,
			actualizado_en TEXT NOT NULL
		)`, pk),

		`CREATE TABLE IF NOT EXISTS user_permissions (
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
			PRIMARY KEY (user_id, permission_id)
		)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS buses (
			id %s,
			placa TEXT NOT NULL,
			nombre TEXT NOT NULL,
			marca TEXT NOT NULL DEFAULT '',
			modelo TEXT NOT NULL DEFAULT '',
			anio INTEGER NOT NULL DEFAULT 0,
			capacidad INTEGER NOT NULL DEFAULT 0,
			tipo TEXT NOT NULL DEFAULT '',
			activo INTEGER NOT NULL DEFAULT 1,
			fecha_compra TEXT NOT NULL,
			ultimo_mantenimiento TEXT,
			precio %s NOT NULL DEFAULT 0,
			peso %s NOT NULL DEFAULT 0,
			color TEXT NOT NULL DEFAULT '',
			descripcion TEXT NOT NULL DEFAULT '',
			creado_en TEXT NOT NULL,
			actualizado_en TEXT NOT NULL
		)`, pk, flt, flt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS mensajes_pendientes (
			id %s,
			telefono TEXT NOT NULL,
			hora_solicitada TEXT NOT NULL,
			hora_desactivacion TEXT NOT NULL,
			usuario_acargo TEXT,
			hora_usuario_asignado TEXT,
			estado TEXT NOT NULL DEFAULT 'pendiente',
			creado_en TEXT NOT NULL,
			actualizado_en TEXT NOT NULL
		)`, pk),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS categorias (
			id %s,
			nombre TEXT NOT NULL,
			descripcion TEXT NOT NULL DEFAULT '',
			activo INTEGER NOT NULL DEFAULT 1,
			creado_en TEXT NOT NULL,
			actualizado_en TEXT NOT NULL
		)`, pk),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS productos (
			id %s,
			nombre TEXT NOT NULL,
			descripcion TEXT NOT NULL DEFAULT '',
			precio %s NOT NULL DEFAULT 0,
			stock INTEGER NOT NULL DEFAULT 0,
			categoria_id BIGINT NOT NULL REFERENCES categorias(id),
			activo INTEGER NOT NULL DEFAULT 1,
			creado_en TEXT NOT NULL,
			actualizado_en TEXT NOT NULL
		)`, pk, flt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS imagenes (
			id %s,
			tipo TEXT NOT NULL,
			entidad_id BIGINT NOT NULL,
			file_name TEXT NOT NULL,
			orden INTEGER NOT NULL DEFAULT 0,
			creado_en TEXT NOT NULL
		)`, pk),

		`CREATE INDEX IF NOT EXISTS idx_imagenes_entidad ON imagenes (tipo, entidad_id)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS archivos (
			id %s,
			file_name TEXT NOT NULL UNIQUE,
			nombre TEXT NOT NULL DEFAULT '',
			tipo TEXT NOT NULL DEFAULT '',
			tamano BIGINT NOT NULL DEFAULT 0,
			visibilidad TEXT NOT NULL DEFAULT 'publico',
			user_id BIGINT NOT NULL DEFAULT 0,
			creado_en TEXT NOT NULL
		)`, pk),

		`CREATE INDEX IF NOT EXISTS idx_archivos_user ON archivos (user_id)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS instalaciones (
			id %s,
			tipo TEXT NOT NULL,
			nombre TEXT NOT NULL,
			estado TEXT NOT NULL,
			lat %s NOT NULL DEFAULT 0,
			lng %s NOT NULL DEFAULT 0,
			direccion TEXT NOT NULL DEFAULT '',
			zona TEXT NOT NULL DEFAULT '',
			fecha_instalacion TEXT NOT NULL DEFAULT '',
			serial TEXT NOT NULL DEFAULT '',
			descripcion TEXT,
			fabricante TEXT,
			modelo TEXT,
			instalador TEXT,
			proveedor TEXT,
			ultima_conexion TEXT,
			firmware TEXT,
			ip TEXT,
			mac TEXT,
			resolucion TEXT,
			contenido TEXT,
			senal %s,
			bateria %s,
			potencia %s,
			contacto TEXT,
			creado_en TEXT NOT NULL,
			actualizado_en TEXT NOT NULL
		)`, pk, flt, flt, flt, flt, flt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS ventas (
			id %s,
			serie TEXT NOT NULL DEFAULT '',
			correlativo BIGINT NOT NULL DEFAULT 0,
			codigo TEXT UNIQUE,
			cliente_nombre TEXT NOT NULL DEFAULT '',
			cliente_documento TEXT NOT NULL DEFAULT '',
			cliente_direccion TEXT NOT NULL DEFAULT '',
			fecha TEXT NOT NULL,
			moneda TEXT NOT NULL DEFAULT 'PEN',
			subtotal %s NOT NULL DEFAULT 0,
			descuento %s NOT NULL DEFAULT 0,
			impuesto %s NOT NULL DEFAULT 0,
			total %s NOT NULL DEFAULT 0,
			estado TEXT NOT NULL DEFAULT 'borrador',
			observaciones TEXT NOT NULL DEFAULT '',
			user_id BIGINT NOT NULL DEFAULT 0,
			creado_en TEXT NOT NULL,
			actualizado_en TEXT NOT NULL
		)`, pk, flt, flt, flt, flt),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS venta_items (
			id %s,
			venta_id BIGINT NOT NULL REFERENCES ventas(id) ON DELETE CASCADE,
			producto_id BIGINT,
			descripcion TEXT NOT NULL DEFAULT '',
			cantidad %s NOT NULL DEFAULT 0,
			precio_unitario %s NOT NULL DEFAULT 0,
			descuento %s NOT NULL DEFAULT 0,
			impuesto_porcentaje %s NOT NULL DEFAULT 0,
			subtotal %s NOT NULL DEFAULT 0,
			impuesto %s NOT NULL DEFAULT 0,
			total %s NOT NULL DEFAULT 0
		)`, pk, flt, flt, flt, flt, flt, flt, flt),

		`CREATE INDEX IF NOT EXISTS idx_venta_items_venta ON venta_items (venta_id)`,

		`CREATE TABLE IF NOT EXISTS venta_secuencias (
			serie TEXT NOT NULL,
			anio INTEGER NOT NULL,
			ultimo BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (serie, anio)
		)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS documentos (
			id %s,
			tipo TEXT NOT NULL,
			serie TEXT NOT NULL,
			anio INTEGER NOT NULL,
			correlativo BIGINT NOT NULL,
			codigo TEXT NOT NULL UNIQUE,
			politica TEXT NOT NULL DEFAULT 'inmutable',
			estado TEXT NOT NULL DEFAULT 'emitido',
			plantilla TEXT NOT NULL,
			plantilla_version TEXT NOT NULL,
			parametros_hash TEXT NOT NULL,
			parametros_json TEXT NOT NULL,
			snapshot_json TEXT NOT NULL DEFAULT '',
			version INTEGER NOT NULL DEFAULT 1,
			user_id BIGINT NOT NULL DEFAULT 0,
			creado_en TEXT NOT NULL,
			actualizado_en TEXT NOT NULL,
			emitido_en TEXT NOT NULL,
			origen_id BIGINT,
			UNIQUE (tipo, parametros_hash)
		)`, pk),

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS documento_archivos (
			id %s,
			documento_id BIGINT NOT NULL REFERENCES documentos(id) ON DELETE CASCADE,
			formato TEXT NOT NULL,
			file_name TEXT NOT NULL,
			mime TEXT NOT NULL,
			tamano BIGINT NOT NULL DEFAULT 0,
			version INTEGER NOT NULL DEFAULT 1,
			creado_en TEXT NOT NULL,
			UNIQUE (documento_id, formato, version)
		)`, pk),

		`CREATE INDEX IF NOT EXISTS idx_documento_archivos_doc ON documento_archivos (documento_id)`,

		`CREATE TABLE IF NOT EXISTS documento_secuencias (
			tipo TEXT NOT NULL,
			serie TEXT NOT NULL,
			anio INTEGER NOT NULL,
			ultimo BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (tipo, serie, anio)
		)`,

		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS documento_eventos (
			id %s,
			documento_id BIGINT NOT NULL REFERENCES documentos(id) ON DELETE CASCADE,
			accion TEXT NOT NULL,
			detalle TEXT NOT NULL DEFAULT '',
			user_id BIGINT NOT NULL DEFAULT 0,
			creado_en TEXT NOT NULL
		)`, pk),
	}
}
