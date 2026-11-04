package database

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Las tablas de schema() deben declararse como CREATE TABLE IF NOT EXISTS para
// que el validador pueda extraer sus columnas.
var createTablePattern = regexp.MustCompile(`(?is)^\s*CREATE TABLE IF NOT EXISTS\s+([a-z0-9_]+)\s*\((.*)\)\s*$`)

// EnsureSchema crea las tablas e índices que falten. Antes de tocar la base,
// comprueba que una base con tablas no sea de otro sistema: si no existe
// ninguna tabla esperada, se niega a continuar en lugar de mezclar esquemas.
func (d *DB) EnsureSchema(ctx context.Context) error {
	expected := expectedSchema(d.dialect)

	actual, err := d.actualSchema(ctx)
	if err != nil {
		return err
	}
	if len(actual) > 0 && !hasAnyExpectedTable(actual, expected) {
		return fmt.Errorf("la base de datos no corresponde al sistema actual: no se encontró ninguna de sus tablas; usa una base vacía")
	}

	for _, stmt := range schema(d.dialect) {
		if _, err := d.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("crear esquema: %w", err)
		}
	}
	return nil
}

// ValidateSchema compara el esquema esperado con el real. Si falta una tabla
// o una columna, el backend no debe arrancar: los CREATE TABLE IF NOT EXISTS
// no modifican tablas ya existentes.
func (d *DB) ValidateSchema(ctx context.Context) error {
	expected := expectedSchema(d.dialect)
	actual, err := d.actualSchema(ctx)
	if err != nil {
		return err
	}

	var missing []string
	for table, columns := range expected {
		actualColumns, ok := actual[table]
		if !ok {
			missing = append(missing, table)
			continue
		}
		for _, column := range columns {
			if !actualColumns[column] {
				missing = append(missing, table+"."+column)
			}
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("la base de datos no corresponde al esquema actual; faltan: %s; usa una base vacía o actualiza el esquema", strings.Join(missing, ", "))
	}
	return nil
}

func expectedSchema(d Dialect) map[string][]string {
	expected := map[string][]string{}

	for _, stmt := range schema(d) {
		match := createTablePattern.FindStringSubmatch(stmt)
		if match == nil {
			continue
		}

		table := strings.ToLower(match[1])
		body := match[2]
		columns := make([]string, 0)

		for _, raw := range strings.Split(body, "\n") {
			line := strings.TrimSpace(raw)
			if line == "" {
				continue
			}

			upper := strings.ToUpper(line)
			if strings.HasPrefix(upper, "PRIMARY ") ||
				strings.HasPrefix(upper, "UNIQUE ") ||
				strings.HasPrefix(upper, "FOREIGN ") ||
				strings.HasPrefix(upper, "CHECK ") ||
				strings.HasPrefix(upper, "CONSTRAINT ") {
				continue
			}

			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			columns = append(columns, strings.ToLower(fields[0]))
		}

		expected[table] = columns
	}

	return expected
}

func hasAnyExpectedTable(actual map[string]map[string]bool, expected map[string][]string) bool {
	for table := range expected {
		if _, ok := actual[table]; ok {
			return true
		}
	}
	return false
}

func (d *DB) actualSchema(ctx context.Context) (map[string]map[string]bool, error) {
	switch d.dbType {
	case "postgres":
		return d.actualSchemaPostgres(ctx)
	case "sqlite":
		return d.actualSchemaSQLite(ctx)
	default:
		return nil, fmt.Errorf("DB_TYPE no soportado: %q", d.dbType)
	}
}

func (d *DB) actualSchemaPostgres(ctx context.Context) (map[string]map[string]bool, error) {
	rows, err := d.QueryContext(ctx, `SELECT table_name, column_name FROM information_schema.columns WHERE table_schema = 'public'`)
	if err != nil {
		return nil, fmt.Errorf("leer esquema de postgres: %w", err)
	}
	defer rows.Close()

	result := map[string]map[string]bool{}
	for rows.Next() {
		var table, column string
		if err := rows.Scan(&table, &column); err != nil {
			return nil, fmt.Errorf("leer columna de postgres: %w", err)
		}
		table = strings.ToLower(table)
		if result[table] == nil {
			result[table] = map[string]bool{}
		}
		result[table][strings.ToLower(column)] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("recorrer esquema de postgres: %w", err)
	}
	return result, nil
}

func (d *DB) actualSchemaSQLite(ctx context.Context) (map[string]map[string]bool, error) {
	// SQLite usa una sola conexión en este proyecto: hay que cerrar el
	// resultado de tablas antes de consultar sus columnas.
	rows, err := d.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%'`)
	if err != nil {
		return nil, fmt.Errorf("leer tablas de sqlite: %w", err)
	}

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			rows.Close()
			return nil, fmt.Errorf("leer tabla de sqlite: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("recorrer tablas de sqlite: %w", err)
	}
	rows.Close()

	result := map[string]map[string]bool{}
	for _, table := range tables {
		columnRows, err := d.QueryContext(ctx, `SELECT name FROM pragma_table_info(?)`, table)
		if err != nil {
			return nil, fmt.Errorf("leer columnas de %s: %w", table, err)
		}

		table = strings.ToLower(table)
		result[table] = map[string]bool{}
		for columnRows.Next() {
			var column string
			if err := columnRows.Scan(&column); err != nil {
				columnRows.Close()
				return nil, fmt.Errorf("leer columna de %s: %w", table, err)
			}
			result[table][strings.ToLower(column)] = true
		}
		if err := columnRows.Err(); err != nil {
			columnRows.Close()
			return nil, fmt.Errorf("recorrer columnas de %s: %w", table, err)
		}
		columnRows.Close()
	}

	return result, nil
}
