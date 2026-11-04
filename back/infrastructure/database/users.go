package database

import (
	"context"
	"time"

	"back/domain"
	"back/domain/inputs"
	"back/infrastructure/auth"
)

type UserStore struct {
	db    *DB
	crypt *auth.CryptManager
}

func NewUserStore(db *DB, crypt *auth.CryptManager) *UserStore {
	return &UserStore{db: db, crypt: crypt}
}

func (s *UserStore) Create(input inputs.UserInput) (domain.User, error) {
	hash, err := s.crypt.Hash(input.Password)
	if err != nil {
		return domain.User{}, err
	}

	now := time.Now()
	ctx := context.Background()
	q := s.db.rebind(`INSERT INTO users (username, email, password, activo, must_change_password, creado_en, actualizado_en)
		VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`)

	var id int64
	err = s.db.QueryRowContext(ctx, q,
		input.Username, input.Email, hash, boolInt(input.Activo), boolInt(input.MustChangePassword),
		formatTime(now), formatTime(now),
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.User{}, domain.Conflict("conflict", "el username ya existe")
		}
		return domain.User{}, err
	}

	permisos := input.Permisos
	if permisos == nil {
		permisos = []string{}
	}
	if err := s.replacePermisos(ctx, id, permisos); err != nil {
		return domain.User{}, err
	}

	return domain.User{
		ID:            id,
		Username:      input.Username,
		Email:         input.Email,
		Password:           hash,
		Activo:             input.Activo,
		Permisos:           permisos,
		MustChangePassword: input.MustChangePassword,
		CreadoEn:           now,
		ActualizadoEn:      now,
	}, nil
}

func (s *UserStore) Update(id int64, input inputs.UserUpdateInput) error {
	ctx := context.Background()
	existing, err := s.FindByID(id)
	if err != nil {
		return err
	}

	if input.Username != nil {
		existing.Username = *input.Username
	}
	if input.Email != nil {
		existing.Email = *input.Email
	}
	if input.Activo != nil {
		existing.Activo = *input.Activo
	}
	if input.Password != nil && *input.Password != "" {
		hash, err := s.crypt.Hash(*input.Password)
		if err != nil {
			return err
		}
		existing.Password = hash
		existing.MustChangePassword = true
	}
	if input.MustChangePassword != nil {
		existing.MustChangePassword = *input.MustChangePassword
	}

	existing.ActualizadoEn = time.Now()

	q := s.db.rebind(`UPDATE users SET username = ?, email = ?, password = ?, activo = ?, must_change_password = ?, actualizado_en = ? WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, q,
		existing.Username, existing.Email, existing.Password, boolInt(existing.Activo),
		boolInt(existing.MustChangePassword), formatTime(existing.ActualizadoEn), id,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.Conflict("conflict", "el username ya existe")
		}
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.NotFound("usuario no encontrado")
	}

	if input.Permisos != nil {
		if err := s.replacePermisos(ctx, id, *input.Permisos); err != nil {
			return err
		}
	}
	return nil
}

// UpdatePassword cambia el hash y el flag de cambio obligatorio. Se usa tanto
// para el cambio de contraseña del propio usuario como para resets de admin.
func (s *UserStore) UpdatePassword(id int64, hash string, mustChange bool) error {
	ctx := context.Background()
	q := s.db.rebind(`UPDATE users SET password = ?, must_change_password = ?, actualizado_en = ? WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, q, hash, boolInt(mustChange), formatTime(time.Now()), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.NotFound("usuario no encontrado")
	}
	return nil
}

func (s *UserStore) Delete(id int64) error {
	ctx := context.Background()
	q := s.db.rebind(`DELETE FROM users WHERE id = ?`)
	res, err := s.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.NotFound("usuario no encontrado")
	}
	return nil
}

func (s *UserStore) FindByUsername(username string) (*domain.User, error) {
	ctx := context.Background()
	q := s.db.rebind(`SELECT id, username, email, password, activo, must_change_password, creado_en, actualizado_en FROM users WHERE username = ?`)
	user, err := s.scanUser(ctx, q, username)
	if err != nil {
		return nil, translateDBError(err, "usuario no encontrado")
	}
	return user, nil
}

func (s *UserStore) FindByID(id int64) (*domain.User, error) {
	ctx := context.Background()
	q := s.db.rebind(`SELECT id, username, email, password, activo, must_change_password, creado_en, actualizado_en FROM users WHERE id = ?`)
	user, err := s.scanUser(ctx, q, id)
	if err != nil {
		return nil, translateDBError(err, "usuario no encontrado")
	}
	return user, nil
}

func (s *UserStore) FindAll() []domain.User {
	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx, `SELECT id, username, email, password, activo, must_change_password, creado_en, actualizado_en FROM users ORDER BY id`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		u, err := scanUserRow(rows)
		if err != nil {
			return users
		}
		users = append(users, *u)
	}
	// Cerrar rows antes de consultar permisos: SQLite usa una sola conexi�n y
	// una consulta anidada dentro del bucle bloquear�a la conexi�n.
	rows.Close()

	for i := range users {
		users[i].Permisos = s.loadPermisos(ctx, users[i].ID)
	}
	return users
}

func (s *UserStore) IsValidPassword(crypt *auth.CryptManager, username, password string) (*domain.User, bool) {
	user, err := s.FindByUsername(username)
	if err != nil {
		return nil, false
	}
	if !crypt.Verify(user.Password, password) {
		return nil, false
	}
	return user, true
}

func (s *UserStore) scanUser(ctx context.Context, query string, arg any) (*domain.User, error) {
	row := s.db.QueryRowContext(ctx, query, arg)
	u, err := scanUserRow(row)
	if err != nil {
		return nil, err
	}
	u.Permisos = s.loadPermisos(ctx, u.ID)
	return u, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUserRow(sc rowScanner) (*domain.User, error) {
	var (
		u           domain.User
		activo      int64
		mustChange  int64
		creado      string
		actualizado string
	)
	if err := sc.Scan(&u.ID, &u.Username, &u.Email, &u.Password, &activo, &mustChange, &creado, &actualizado); err != nil {
		return nil, err
	}
	u.Activo = intBool(activo)
	u.MustChangePassword = intBool(mustChange)
	u.CreadoEn = parseTime(creado)
	u.ActualizadoEn = parseTime(actualizado)
	u.Permisos = []string{}
	return &u, nil
}

func (s *UserStore) loadPermisos(ctx context.Context, userID int64) []string {
	q := s.db.rebind(`SELECT p.nombre FROM permissions p
		JOIN user_permissions up ON up.permission_id = p.id
		WHERE up.user_id = ? ORDER BY p.nombre`)
	rows, err := s.db.QueryContext(ctx, q, userID)
	if err != nil {
		return []string{}
	}
	defer rows.Close()

	permisos := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			permisos = append(permisos, name)
		}
	}
	return permisos
}

func (s *UserStore) replacePermisos(ctx context.Context, userID int64, permisos []string) error {
	if _, err := s.db.ExecContext(ctx, s.db.rebind(`DELETE FROM user_permissions WHERE user_id = ?`), userID); err != nil {
		return err
	}
	q := s.db.rebind(`INSERT INTO user_permissions (user_id, permission_id)
		SELECT ?, id FROM permissions WHERE nombre = ?`)
	for _, name := range permisos {
		res, err := s.db.ExecContext(ctx, q, userID, name)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return domain.Validation("permiso no encontrado: " + name)
		}
	}
	return nil
}
