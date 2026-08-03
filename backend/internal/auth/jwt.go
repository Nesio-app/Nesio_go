package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(getJWTSecret())

type Claims struct {
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

func GenerateToken(userID uuid.UUID, email string) (string, error) {
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func getJWTSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	return "nesio_dev_secret_change_in_prod"
}

func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ResetTokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func GenerateResetToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func CreateUser(db *sqlx.DB, email, password string) (uuid.UUID, error) {
	email = NormalizeEmail(email)
	hash, err := HashPassword(password)
	if err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	err = db.QueryRow(
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id",
		email, hash,
	).Scan(&id)
	return id, err
}

func AuthenticateUser(db *sqlx.DB, email, password string) (uuid.UUID, error) {
	email = NormalizeEmail(email)
	var user struct {
		ID           uuid.UUID `db:"id"`
		PasswordHash string    `db:"password_hash"`
	}
	err := db.Get(&user, "SELECT id, password_hash FROM users WHERE lower(email) = lower($1)", email)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid credentials")
	}
	if !CheckPassword(password, user.PasswordHash) {
		return uuid.Nil, fmt.Errorf("invalid credentials")
	}
	return user.ID, nil
}

func CreatePasswordResetToken(db *sqlx.DB, email string) (string, time.Time, error) {
	email = NormalizeEmail(email)
	var userID uuid.UUID
	if err := db.Get(&userID, "SELECT id FROM users WHERE lower(email) = lower($1)", email); err != nil {
		return "", time.Time{}, fmt.Errorf("user not found")
	}

	token, err := GenerateResetToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().Add(30 * time.Minute)
	if _, err := db.Exec(
		"INSERT INTO password_reset_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		userID,
		ResetTokenHash(token),
		expiresAt,
	); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func ResetPasswordWithToken(db *sqlx.DB, email, token, newPassword string) error {
	email = NormalizeEmail(email)
	tx, err := db.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var user struct {
		ID uuid.UUID `db:"id"`
	}
	if err := tx.Get(&user, "SELECT id FROM users WHERE lower(email) = lower($1)", email); err != nil {
		return fmt.Errorf("invalid reset token")
	}

	var resetToken struct {
		ID        uuid.UUID  `db:"id"`
		UsedAt    *time.Time `db:"used_at"`
		ExpiresAt time.Time  `db:"expires_at"`
		UserID    uuid.UUID  `db:"user_id"`
	}
	if err := tx.Get(&resetToken, "SELECT id, user_id, used_at, expires_at FROM password_reset_tokens WHERE token_hash = $1 ORDER BY created_at DESC LIMIT 1", ResetTokenHash(token)); err != nil {
		return fmt.Errorf("invalid reset token")
	}
	if resetToken.UserID != user.ID || resetToken.UsedAt != nil || time.Now().After(resetToken.ExpiresAt) {
		return fmt.Errorf("invalid reset token")
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE users SET password_hash = $1 WHERE id = $2", hash, user.ID); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE password_reset_tokens SET used_at = now() WHERE id = $1", resetToken.ID); err != nil {
		return err
	}
	return tx.Commit()
}
