package postgresRepo

import (
	"context"
	"golang.org/x/crypto/bcrypt"
	"log"
	"time"

	"github.com/namhuydao/vigilate/internal/models"
)

// AllUsers returns all users
func (m *postgresDBRepo) AllUsers() ([]models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT id, last_name, first_name, email, user_active, created_at, updated_at FROM users
		WHERE deleted_at IS NULL`

	rows, err := m.DB.QueryContext(ctx, stmt)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []models.User

	for rows.Next() {
		s := models.User{}
		err = rows.Scan(&s.ID, &s.LastName, &s.FirstName, &s.Email, &s.UserActive, &s.CreatedAt, &s.UpdatedAt)
		if err != nil {
			return nil, err
		}
		// Append it to the slice
		users = append(users, s)
	}

	if err = rows.Err(); err != nil {
		log.Println(err)
		return nil, err
	}

	return users, nil
}

// GetUserById returns a user by id
func (m *postgresDBRepo) GetUserById(id int) (models.User, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `SELECT id, first_name, last_name,  user_active, access_level, email,
			created_at, updated_at
			FROM users WHERE id = $1`
	row := m.DB.QueryRowContext(ctx, stmt, id)

	var u models.User

	err := row.Scan(
		&u.ID,
		&u.FirstName,
		&u.LastName,
		&u.UserActive,
		&u.AccessLevel,
		&u.Email,
		&u.CreatedAt,
		&u.UpdatedAt,
	)

	if err != nil {
		log.Println(err)
		return u, err
	}

	return u, nil
}

// InsertUser Insert method to add a new record to the users table.
func (m *postgresDBRepo) InsertUser(u models.User) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create a bcrypt hash of the plain-text password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), 12)
	if err != nil {
		return 0, err
	}

	stmt := `
	INSERT INTO users
	    (
		first_name,
		last_name,
		email,
		password,
		access_level,
		user_active
		)
   VALUES($1, $2, $3, $4, $5, $6) RETURNING id `

	var newId int
	err = m.DB.QueryRowContext(ctx, stmt,
		u.FirstName,
		u.LastName,
		u.Email,
		hashedPassword,
		u.AccessLevel,
		&u.UserActive).Scan(&newId)
	if err != nil {
		return 0, err
	}

	return newId, err
}

// UpdateUser updates a user by id
func (m *postgresDBRepo) UpdateUser(u models.User) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `
		UPDATE
			users
		SET
			first_name = $1,
			last_name = $2,
			user_active = $3,
			email = $4,
			access_level = $5,
			updated_at = $6
		WHERE
			id = $7`

	_, err := m.DB.ExecContext(ctx, stmt,
		u.FirstName,
		u.LastName,
		u.UserActive,
		u.Email,
		u.AccessLevel,
		u.UpdatedAt,
		u.ID,
	)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// DeleteUser sets a user to deleted by populating deleted_at value
func (m *postgresDBRepo) DeleteUser(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := `UPDATE users SET deleted_at = $1, user_active = 0  WHERE id = $2`

	_, err := m.DB.ExecContext(ctx, stmt, time.Now(), id)
	if err != nil {
		log.Println(err)
		return err
	}

	return nil
}

// UpdatePassword resets a password
func (m *postgresDBRepo) UpdatePassword(id int, newPassword string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Create a bcrypt hash of the plain-text password.
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), 12)
	if err != nil {
		log.Println(err)
		return err
	}

	stmt := `UPDATE users SET password = $1 WHERE id = $2`
	_, err = m.DB.ExecContext(ctx, stmt, hashedPassword, id)
	if err != nil {
		log.Println(err)
		return err
	}

	// delete all remember tokens, if any
	stmt = "DELETE FROM remember_tokens WHERE user_id = $1"
	_, err = m.DB.ExecContext(ctx, stmt, id)
	if err != nil {
		return err
	}

	return nil
}
