package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	demoAdminEmail    = "admin@inventoryintel.demo"
	legacyDemoEmail   = "kenneth@inventoryintel.demo"
	demoAdminUsername = "kenneth"
	demoAdminPassword = "DemoAdmin123!"
	demoAdminName     = "Kenneth"
	adminRole         = "Administrator"
	sessionCookieName = "inventory_intel_session"
)

var sessionTTL = 7 * 24 * time.Hour

type contextKey string

const authUserContextKey contextKey = "auth-user"

type User struct {
	ID        int64  `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"createdAt"`
	LastLogin string `json:"lastLogin"`
}

type authSessionPayload struct {
	Authenticated bool  `json:"authenticated"`
	User          *User `json:"user,omitempty"`
}

func (a *App) ensureDemoAdminUser() error {
	passwordHash, err := hashPassword(demoAdminPassword)
	if err != nil {
		return err
	}

	var userID int64
	err = a.db.QueryRow(`SELECT id FROM users WHERE email = ?`, demoAdminEmail).Scan(&userID)
	switch {
	case err == nil:
		_, err = a.db.Exec(`
			UPDATE users
			SET password_hash = ?, name = ?
			WHERE id = ?`,
			passwordHash, demoAdminName, userID,
		)
		return err
	case errors.Is(err, sql.ErrNoRows):
		var legacyUserID int64
		legacyErr := a.db.QueryRow(`SELECT id FROM users WHERE email = ?`, legacyDemoEmail).Scan(&legacyUserID)
		switch {
		case legacyErr == nil:
			_, err = a.db.Exec(`
				UPDATE users
				SET email = ?, password_hash = ?, name = ?
				WHERE id = ?`,
				demoAdminEmail, passwordHash, demoAdminName, legacyUserID,
			)
			return err
		case errors.Is(legacyErr, sql.ErrNoRows):
			_, err = a.db.Exec(`
				INSERT INTO users (email, password_hash, name, created_at, last_login)
				VALUES (?, ?, ?, ?, ?)`,
				demoAdminEmail, passwordHash, demoAdminName, nowString(), "",
			)
			return err
		default:
			return legacyErr
		}
	default:
		return err
	}
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func authenticatePassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func currentUser(r *http.Request) (User, bool) {
	user, ok := r.Context().Value(authUserContextKey).(User)
	return user, ok
}

func (a *App) sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err == nil && strings.TrimSpace(cookie.Value) != "" {
			user, userErr := a.userForSession(cookie.Value)
			if userErr == nil {
				r = r.WithContext(context.WithValue(r.Context(), authUserContextKey, user))
			} else {
				clearSessionCookie(w)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) apiAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/") || strings.HasPrefix(r.URL.Path, "/api/auth/") {
			next.ServeHTTP(w, r)
			return
		}
		if _, ok := currentUser(r); !ok {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if err := decodeJSON(r, &input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, token, err := a.login(strings.TrimSpace(input.Identifier), input.Password)
	if err != nil {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return
	}

	setSessionCookie(w, token, sessionTTL, r.TLS != nil)
	writeJSON(w, http.StatusOK, authSessionPayload{Authenticated: true, User: &user})
}

func (a *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		_, _ = a.db.Exec(`DELETE FROM user_sessions WHERE token_hash = ?`, tokenHash(cookie.Value))
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) handleSession(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		writeJSON(w, http.StatusOK, authSessionPayload{Authenticated: false})
		return
	}
	writeJSON(w, http.StatusOK, authSessionPayload{Authenticated: true, User: &user})
}

func (a *App) login(identifier, password string) (User, string, error) {
	identifier = strings.ToLower(strings.TrimSpace(identifier))
	password = strings.TrimSpace(password)
	if identifier == "" || password == "" {
		return User{}, "", errors.New("identifier and password are required")
	}

	var user User
	var passwordHash string
	err := a.db.QueryRow(`
		SELECT id, email, name, created_at, last_login, password_hash
		FROM users
		WHERE LOWER(email) = ? OR LOWER(name) = ?
		LIMIT 1`,
		identifier, identifier,
	).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.LastLogin, &passwordHash)
	if err != nil {
		return User{}, "", err
	}
	if err := authenticatePassword(password, passwordHash); err != nil {
		return User{}, "", err
	}

	token, err := randomToken()
	if err != nil {
		return User{}, "", err
	}

	now := nowString()
	expiresAt := time.Now().UTC().Add(sessionTTL).Format(time.RFC3339)

	tx, err := a.db.BeginTx(context.Background(), nil)
	if err != nil {
		return User{}, "", err
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err := tx.Exec(`UPDATE users SET last_login = ? WHERE id = ?`, now, user.ID); err != nil {
		return User{}, "", err
	}
	if _, err := tx.Exec(`DELETE FROM user_sessions WHERE user_id = ?`, user.ID); err != nil {
		return User{}, "", err
	}
	if _, err := tx.Exec(`
		INSERT INTO user_sessions (user_id, token_hash, expires_at, created_at, last_seen_at)
		VALUES (?, ?, ?, ?, ?)`,
		user.ID, tokenHash(token), expiresAt, now, now,
	); err != nil {
		return User{}, "", err
	}
	if _, err := tx.Exec(`DELETE FROM user_sessions WHERE expires_at <= ?`, now); err != nil {
		return User{}, "", err
	}
	if err := tx.Commit(); err != nil {
		return User{}, "", err
	}
	tx = nil

	user.LastLogin = now
	user.Role = adminRole
	return user, token, nil
}

func (a *App) userForSession(token string) (User, error) {
	var user User
	now := nowString()
	err := a.db.QueryRow(`
		SELECT u.id, u.email, u.name, u.created_at, u.last_login
		FROM user_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = ? AND s.expires_at > ?`,
		tokenHash(token), now,
	).Scan(&user.ID, &user.Email, &user.Name, &user.CreatedAt, &user.LastLogin)
	if err != nil {
		return User{}, err
	}
	user.Role = adminRole
	_, _ = a.db.Exec(`UPDATE user_sessions SET last_seen_at = ? WHERE token_hash = ?`, now, tokenHash(token))
	return user, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func setSessionCookie(w http.ResponseWriter, token string, ttl time.Duration, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		MaxAge:   int(ttl.Seconds()),
		Expires:  time.Now().Add(ttl),
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
	})
}

func authDemoCredentialsJSON() string {
	payload := map[string]string{
		"email":    demoAdminEmail,
		"username": demoAdminUsername,
		"password": demoAdminPassword,
	}
	blob, _ := json.Marshal(payload)
	return string(blob)
}
