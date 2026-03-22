package config

import (
	"crypto/subtle"
	"regexp"
)

// usernameRE validates proxy usernames: 1-64 alphanumeric, hyphen, or underscore characters.
var usernameRE = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

// reservedPathSegments are URL path segments that cannot be used as usernames to avoid route conflicts.
var reservedPathSegments = map[string]bool{
	"live": true, "movie": true, "series": true, "hls": true, "hlsr": true,
	"play": true, "get.php": true, "player_api.php": true, "xmltv.php": true,
	"api": true, "apiget": true, "timeshift": true,
}

// ValidUsername returns true if username matches the allowed pattern and is not a reserved path segment.
func ValidUsername(username string) bool {
	return usernameRE.MatchString(username) && !reservedPathSegments[username]
}

// UserInfo is a user entry returned by AllUsers (password omitted).
type UserInfo struct {
	Username    string `json:"username"`
	Description string `json:"description,omitempty"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at,omitempty"`
	IsDefault   bool   `json:"is_default"`
}

// MigrateDefaultUser ensures the CLI --user/--password is present in the Users slice.
// Call once at startup after ApplyTo. If User is non-empty and not already in Users,
// it is prepended. After this, all user operations go through the Users slice only.
func (p *ProxyConfig) MigrateDefaultUser() {
	if p.User.String() == "" {
		return
	}
	for _, u := range p.Users {
		if u.Username == p.User.String() {
			return // already in the slice
		}
	}
	p.Users = append([]User{{
		Username: p.User.String(),
		Password: p.Password.String(),
		Enabled:  true,
	}}, p.Users...)
}

// FindUser returns the User matching username, or nil.
func (p *ProxyConfig) FindUser(username string) *User {
	if username == "" {
		return nil
	}
	for i := range p.Users {
		if p.Users[i].Username == username {
			return &p.Users[i]
		}
	}
	return nil
}

// ValidateCredentials checks username/password against all users in the Users slice.
// Returns the matched username or "" if invalid. Uses constant-time comparison
// for both username and password to avoid timing-based user enumeration.
// Returns "" for disabled users.
func (p *ProxyConfig) ValidateCredentials(username, password string) string {
	if username == "" {
		return ""
	}
	for _, u := range p.Users {
		uMatch := constantTimeEqual(username, u.Username)
		pMatch := constantTimeEqual(password, u.Password)
		if uMatch && pMatch {
			if !u.Enabled {
				return ""
			}
			return username
		}
	}
	return ""
}

// AllUsers returns a list of all users (password omitted).
func (p *ProxyConfig) AllUsers() []UserInfo {
	out := make([]UserInfo, 0, len(p.Users))
	for _, u := range p.Users {
		out = append(out, UserInfo{
			Username:    u.Username,
			Description: u.Description,
			Enabled:     u.Enabled,
			CreatedAt:   u.CreatedAt,
		})
	}
	return out
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
