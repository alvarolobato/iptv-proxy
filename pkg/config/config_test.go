package config

import (
	"testing"
)

func TestCredentialString_String(t *testing.T) {
	c := CredentialString("user@pass")
	if got := c.String(); got != "user@pass" {
		t.Errorf("String() = %q, want %q", got, "user@pass")
	}
}

func TestCredentialString_PathEscape(t *testing.T) {
	tests := []struct {
		name string
		c    CredentialString
		want string
	}{
		{"simple", CredentialString("user"), "user"},
		{"with slash", CredentialString("a/b"), "a%2Fb"},
		{"empty", CredentialString(""), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.PathEscape(); got != tt.want {
				t.Errorf("PathEscape() = %q, want %q", got, tt.want)
			}
		})
	}
}
