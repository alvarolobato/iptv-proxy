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
	p.MigrateDefaultUser()
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
	p.MigrateDefaultUser()
	if got := p.ValidateCredentials("admin", "adminpass"); got != "admin" {
		t.Errorf("got %q", got)
	}
}

func TestValidateCredentials_DefaultUser_BeforeMigration(t *testing.T) {
	// Without migration, CLI user is NOT in Users slice and won't authenticate
	p := makeProxyConfig()
	if got := p.ValidateCredentials("admin", "adminpass"); got != "" {
		t.Errorf("expected empty before migration, got %q", got)
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

func TestAllUsers_Empty(t *testing.T) {
	p := &ProxyConfig{User: "admin", Password: "pass"}
	users := p.AllUsers()
	if len(users) != 0 {
		t.Fatalf("expected 0 users (CLI user not auto-added to AllUsers), got %d", len(users))
	}
}

func TestAllUsers_WithMigration(t *testing.T) {
	p := makeProxyConfig()
	p.MigrateDefaultUser()
	users := p.AllUsers()
	if len(users) != 3 {
		t.Fatalf("expected 3 users, got %d", len(users))
	}
	if users[0].Username != "admin" {
		t.Errorf("first user should be admin, got %q", users[0].Username)
	}
}

func TestAllUsers_FromSlice(t *testing.T) {
	p := makeProxyConfig()
	users := p.AllUsers()
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if users[0].Username != "alice" {
		t.Errorf("first user should be alice, got %q", users[0].Username)
	}
}

func TestMigrateDefaultUser(t *testing.T) {
	p := &ProxyConfig{User: "admin", Password: "pass"}
	p.MigrateDefaultUser()
	if len(p.Users) != 1 {
		t.Fatalf("expected 1 user after migration, got %d", len(p.Users))
	}
	if p.Users[0].Username != "admin" || p.Users[0].Password != "pass" {
		t.Errorf("migrated user mismatch: %+v", p.Users[0])
	}
	// Calling again should not duplicate
	p.MigrateDefaultUser()
	if len(p.Users) != 1 {
		t.Fatalf("expected 1 user after second migration, got %d", len(p.Users))
	}
}

func TestMigrateDefaultUser_EmptyUser(t *testing.T) {
	p := &ProxyConfig{}
	p.MigrateDefaultUser()
	if len(p.Users) != 0 {
		t.Fatalf("expected 0 users when CLI user is empty, got %d", len(p.Users))
	}
}

func TestMigrateDefaultUser_AlreadyInSlice(t *testing.T) {
	p := makeProxyConfig()
	p.Users = append([]User{{Username: "admin", Password: "adminpass", Enabled: true}}, p.Users...)
	p.MigrateDefaultUser()
	// Should not duplicate admin
	count := 0
	for _, u := range p.Users {
		if u.Username == "admin" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected 1 admin, got %d", count)
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
