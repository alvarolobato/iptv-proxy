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
	Username  string `json:"username"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"created_at,omitempty"`
	IsDefault bool   `json:"is_default"`
}

// FindUser returns the User matching username, or nil.
// Checks both the default user (User/Password fields) and the Users slice.
func (p *ProxyConfig) FindUser(username string) *User {
	if username == "" {
		return nil
	}
	if username == p.User.String() {
		return &User{
			Username: p.User.String(),
			Password: p.Password.String(),
			Enabled:  true,
		}
	}
	for i := range p.Users {
		if p.Users[i].Username == username {
			return &p.Users[i]
		}
	}
	return nil
}

// ValidateCredentials checks username/password against all users (default + Users slice).
// Returns the matched username or "" if invalid. Uses constant-time comparison.
// Returns "" for disabled users.
func (p *ProxyConfig) ValidateCredentials(username, password string) string {
	if username == "" {
		return ""
	}
	// Check default user.
	if constantTimeEqual(username, p.User.String()) && constantTimeEqual(password, p.Password.String()) {
		return username
	}
	// Check additional users.
	for _, u := range p.Users {
		if constantTimeEqual(username, u.Username) && constantTimeEqual(password, u.Password) {
			if !u.Enabled {
				return ""
			}
			return username
		}
	}
	return ""
}

// AllUsers returns a list of all users (default + Users slice) with is_default flag.
func (p *ProxyConfig) AllUsers() []UserInfo {
	out := make([]UserInfo, 0, 1+len(p.Users))
	out = append(out, UserInfo{
		Username:  p.User.String(),
		Enabled:   true,
		IsDefault: true,
	})
	for _, u := range p.Users {
		out = append(out, UserInfo{
			Username:  u.Username,
			Enabled:   u.Enabled,
			CreatedAt: u.CreatedAt,
			IsDefault: false,
		})
	}
	return out
}

func constantTimeEqual(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
