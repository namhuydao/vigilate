package postgresRepo

import (
	"context"
	"database/sql"
	"errors"
	"golang.org/x/crypto/bcrypt"
	"log"
	"time"

	"github.com/namhuydao/vigilate/internal/models"
)

// Authenticate authenticates
func (m *postgresDBRepo) Authenticate(email, testPassword string) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var id int
	var hashedPassword string
	var userActive int

	query := `
		SELECT
		    id, password, user_active
		FROM
			users
		WHERE
			email = $1
			AND deleted_at IS NULL`

	row := m.DB.QueryRowContext(ctx, query, email)
	err := row.Scan(&id, &hashedPassword, &userActive)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, models.ErrInvalidCredentials
	} else if err != nil {
		log.Println(err)
		return 0, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(testPassword))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return 0, models.ErrInvalidCredentials
	} else if err != nil {
		log.Println(err)
		return 0, err
	}

	if userActive == 0 {
		return 0, models.ErrInactiveAccount
	}

	// Otherwise, the password is correct. Return the user ID and hashed password.
	return id, nil
}

// InsertRememberMeToken inserts a remember me token into remember_tokens for a user
func (m *postgresDBRepo) InsertRememberMeToken(id int, token string) error {
	//// Begin the transaction
	//tx, err := m.DB.Begin()
	//if err != nil {
	//	log.Println(err)
	//}
	//
	//query := "INSERT INTO remember_tokens (user_id, remember_token) VALUES ($1, $2)"
	//
	//rows, err := m.DB.Query(query, id, token)
	//if err != nil {
	//	err = tx.Rollback()
	//	if err != nil {
	//		return err
	//	}
	//	log.Println(err)
	//}
	//defer func(rows *sql.Rows) {
	//	err := rows.Close()
	//	if err != nil {
	//
	//	}
	//}(rows)
	//
	//// Commit the transaction
	//if err := tx.Commit(); err != nil {
	//	log.Fatal(err)
	//}

	return nil
}

// DeleteRememberMeToken deletes a remember me token
func (m *postgresDBRepo) DeleteRememberMeToken(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := "DELETE FROM remember_tokens WHERE remember_token = $1"
	_, err := m.DB.ExecContext(ctx, stmt, token)
	if err != nil {
		return err
	}

	return nil
}

// CheckForRememberMeToken checks for a valid remember me token
func (m *postgresDBRepo) CheckForRememberMeToken(id int, token string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	stmt := "SELECT id FROM remember_tokens WHERE user_id = $1 AND remember_token = $2"
	row := m.DB.QueryRowContext(ctx, stmt, id, token)
	err := row.Scan(&id)
	return err == nil
}
