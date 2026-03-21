package config

import "testing"

func makeProxyConfig() *ProxyConfig {
	return &ProxyConfig{
		User:     CredentialString("admin"),
		Password: CredentialString("adminpass"),
		Users: []User{
			{Username: "alice", Password: "alice123", Enabled: true, CreatedAt: "2026-03-21T10:00:00Z"},
			{Username: "bob", Password: "bob456", Enabled: false, CreatedAt: "2026-03-21T11:00:00Z"},
		},
	}
}

func TestFindUser_DefaultUser(t *testing.T) {
	p := makeProxyConfig()
	u := p.FindUser("admin")
	if u == nil {
		t.Fatal("expected non-nil user")
	}
	if u.Username != "admin" || u.Password != "adminpass" {
		t.Errorf("got %+v", u)
	}
}

func TestFindUser_AdditionalUser(t *testing.T) {
	p := makeProxyConfig()
	u := p.FindUser("alice")
	if u == nil {
		t.Fatal("expected non-nil user")
	}
	if u.Username != "alice" || u.Password != "alice123" {
		t.Errorf("got %+v", u)
	}
}

func TestFindUser_NotFound(t *testing.T) {
	p := makeProxyConfig()
	u := p.FindUser("unknown")
	if u != nil {
		t.Errorf("expected nil, got %+v", u)
	}
}

func TestFindUser_Empty(t *testing.T) {
	p := makeProxyConfig()
	if u := p.FindUser(""); u != nil {
		t.Errorf("expected nil for empty username")
	}
}

func TestValidateCredentials_DefaultUser(t *testing.T) {
	p := makeProxyConfig()
	if got := p.ValidateCredentials("admin", "adminpass"); got != "admin" {
		t.Errorf("got %q", got)
	}
}

func TestValidateCredentials_AdditionalUser(t *testing.T) {
	p := makeProxyConfig()
	if got := p.ValidateCredentials("alice", "alice123"); got != "alice" {
		t.Errorf("got %q", got)
	}
}

func TestValidateCredentials_WrongPassword(t *testing.T) {
	p := makeProxyConfig()
	if got := p.ValidateCredentials("alice", "wrong"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestValidateCredentials_DisabledUser(t *testing.T) {
	p := makeProxyConfig()
	if got := p.ValidateCredentials("bob", "bob456"); got != "" {
		t.Errorf("expected empty for disabled user, got %q", got)
	}
}

func TestValidateCredentials_EmptyUsername(t *testing.T) {
	p := makeProxyConfig()
	if got := p.ValidateCredentials("", "anything"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestAllUsers_DefaultOnly(t *testing.T) {
	p := &ProxyConfig{User: "admin", Password: "pass"}
	users := p.AllUsers()
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if !users[0].IsDefault || users[0].Username != "admin" {
		t.Errorf("unexpected default user: %+v", users[0])
	}
}

func TestAllUsers_Multiple(t *testing.T) {
	p := makeProxyConfig()
	users := p.AllUsers()
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}
	if !users[0].IsDefault {
		t.Error("first user should be default")
	}
	if users[1].IsDefault || users[1].Username != "alice" {
		t.Errorf("unexpected second user: %+v", users[1])
	}
}

func TestValidUsername(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"alice", true},
		{"bob-123", true},
		{"user_name", true},
		{"", false},
		{"a b", false},
		{"live", false},      // reserved
		{"get.php", false},   // reserved
		{"a/b", false},
		{"admin", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidUsername(tt.name); got != tt.valid {
				t.Errorf("ValidUsername(%q) = %v, want %v", tt.name, got, tt.valid)
			}
		})
	}
}

func TestSettingsJSON_UsersRoundTrip(t *testing.T) {
	s := SettingsJSON{
		Users: []User{
			{Username: "test", Password: "pass", Enabled: true},
		},
	}
	if len(s.Users) != 1 || s.Users[0].Username != "test" {
		t.Errorf("unexpected: %+v", s.Users)
	}
}
