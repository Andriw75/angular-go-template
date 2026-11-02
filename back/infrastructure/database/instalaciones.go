package database

import (
	"context"
	"database/sql"
	"time"

	"back/domain"
)

type InstalacionStore struct {
	db *DB
}

func NewInstalacionStore(db *DB) *InstalacionStore {
	return &InstalacionStore{db: db}
}

const instalacionColumns = `id, tipo, nombre, estado, lat, lng, direccion, zona, fecha_instalacion, serial,
	descripcion, fabricante, modelo, instalador, proveedor, ultima_conexion, firmware, ip, mac,
	resolucion, contenido, senal, bateria, potencia, contacto, creado_en, actualizado_en`

func (s *InstalacionStore) List() []domain.Instalacion {
	out := make([]domain.Instalacion, 0)

	rows, err := s.db.QueryContext(context.Background(), `SELECT `+instalacionColumns+` FROM instalaciones ORDER BY id`)
	if err != nil {
		return out
	}
	defer rows.Close()

	for rows.Next() {
		i, err := scanInstalacion(rows)
		if err != nil {
			return out
		}
		out = append(out, *i)
	}
	return out
}

func (s *InstalacionStore) FindByID(id int64) (*domain.Instalacion, error) {
	q := s.db.rebind(`SELECT ` + instalacionColumns + ` FROM instalaciones WHERE id = ?`)
	return scanInstalacion(s.db.QueryRowContext(context.Background(), q, id))
}

func (s *InstalacionStore) Create(i domain.Instalacion) (domain.Instalacion, error) {
	now := time.Now()
	i.CreadoEn = now
	i.ActualizadoEn = now

	q := s.db.rebind(`INSERT INTO instalaciones
		(tipo, nombre, estado, lat, lng, direccion, zona, fecha_instalacion, serial,
		 descripcion, fabricante, modelo, instalador, proveedor, ultima_conexion, firmware,
		 ip, mac, resolucion, contenido, senal, bateria, potencia, contacto, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`)

	err := s.db.QueryRowContext(context.Background(), q,
		i.Tipo, i.Nombre, i.Estado, i.Lat, i.Lng, i.Direccion, i.Zona, i.FechaInstalacion, i.Serial,
		strPtr(i.Descripcion), strPtr(i.Fabricante), strPtr(i.Modelo), strPtr(i.Instalador),
		strPtr(i.Proveedor), strPtr(i.UltimaConexion), strPtr(i.Firmware), strPtr(i.IP), strPtr(i.MAC),
		strPtr(i.Resolucion), strPtr(i.Contenido), floatPtr(i.Senal), floatPtr(i.Bateria),
		floatPtr(i.Potencia), strPtr(i.Contacto), formatTime(now), formatTime(now),
	).Scan(&i.ID)
	if err != nil {
		return domain.Instalacion{}, err
	}
	return i, nil
}

func (s *InstalacionStore) Update(id int64, i domain.Instalacion) error {
	i.ActualizadoEn = time.Now()
	q := s.db.rebind(`UPDATE instalaciones SET
		tipo = ?, nombre = ?, estado = ?, lat = ?, lng = ?, direccion = ?, zona = ?, fecha_instalacion = ?, serial = ?,
		descripcion = ?, fabricante = ?, modelo = ?, instalador = ?, proveedor = ?, ultima_conexion = ?, firmware = ?,
		ip = ?, mac = ?, resolucion = ?, contenido = ?, senal = ?, bateria = ?, potencia = ?, contacto = ?, actualizado_en = ?
		WHERE id = ?`)

	res, err := s.db.ExecContext(context.Background(), q,
		i.Tipo, i.Nombre, i.Estado, i.Lat, i.Lng, i.Direccion, i.Zona, i.FechaInstalacion, i.Serial,
		strPtr(i.Descripcion), strPtr(i.Fabricante), strPtr(i.Modelo), strPtr(i.Instalador),
		strPtr(i.Proveedor), strPtr(i.UltimaConexion), strPtr(i.Firmware), strPtr(i.IP), strPtr(i.MAC),
		strPtr(i.Resolucion), strPtr(i.Contenido), floatPtr(i.Senal), floatPtr(i.Bateria),
		floatPtr(i.Potencia), strPtr(i.Contacto), formatTime(i.ActualizadoEn), id,
	)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *InstalacionStore) Delete(id int64) error {
	res, err := s.db.ExecContext(context.Background(), s.db.rebind(`DELETE FROM instalaciones WHERE id = ?`), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func scanInstalacion(sc rowScanner) (*domain.Instalacion, error) {
	var (
		i                   domain.Instalacion
		descripcion         sql.NullString
		fabricante          sql.NullString
		modelo              sql.NullString
		instalador          sql.NullString
		proveedor           sql.NullString
		ultimaConexion      sql.NullString
		firmware            sql.NullString
		ip                  sql.NullString
		mac                 sql.NullString
		resolucion          sql.NullString
		contenido           sql.NullString
		senal               sql.NullFloat64
		bateria             sql.NullFloat64
		potencia            sql.NullFloat64
		contacto            sql.NullString
		creado, actualizado string
	)

	if err := sc.Scan(
		&i.ID, &i.Tipo, &i.Nombre, &i.Estado, &i.Lat, &i.Lng, &i.Direccion, &i.Zona,
		&i.FechaInstalacion, &i.Serial, &descripcion, &fabricante, &modelo, &instalador,
		&proveedor, &ultimaConexion, &firmware, &ip, &mac, &resolucion, &contenido,
		&senal, &bateria, &potencia, &contacto, &creado, &actualizado,
	); err != nil {
		return nil, err
	}

	i.Descripcion = nullStr(descripcion)
	i.Fabricante = nullStr(fabricante)
	i.Modelo = nullStr(modelo)
	i.Instalador = nullStr(instalador)
	i.Proveedor = nullStr(proveedor)
	i.UltimaConexion = nullStr(ultimaConexion)
	i.Firmware = nullStr(firmware)
	i.IP = nullStr(ip)
	i.MAC = nullStr(mac)
	i.Resolucion = nullStr(resolucion)
	i.Contenido = nullStr(contenido)
	i.Senal = nullFloat(senal)
	i.Bateria = nullFloat(bateria)
	i.Potencia = nullFloat(potencia)
	i.Contacto = nullStr(contacto)
	i.CreadoEn = parseTime(creado)
	i.ActualizadoEn = parseTime(actualizado)
	return &i, nil
}

func nullStr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	v := ns.String
	return &v
}

func nullFloat(nf sql.NullFloat64) *float64 {
	if !nf.Valid {
		return nil
	}
	v := nf.Float64
	return &v
}
