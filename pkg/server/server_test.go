package server

import (
	"net/url"
	"testing"

	"github.com/pierre-emmanuelJ/iptv-proxy/pkg/config"
)

func TestNewServer_EmptyRemoteURL(t *testing.T) {
	emptyURL, _ := url.Parse("")
	conf := &config.ProxyConfig{
		HostConfig:       &config.HostConfiguration{Hostname: "localhost", Port: 8080},
		RemoteURL:       emptyURL,
		User:             config.CredentialString("u"),
		Password:         config.CredentialString("p"),
		AdvertisedPort:   8080,
	}
	srv, err := NewServer(conf)
	if err != nil {
		t.Fatalf("NewServer() err = %v", err)
	}
	if srv == nil {
		t.Fatal("NewServer() returned nil")
	}
	if srv.ProxyConfig != conf {
		t.Error("ProxyConfig not set")
	}
}

func TestReplaceURL_M3U(t *testing.T) {
	conf := &Config{
		ProxyConfig: &config.ProxyConfig{
			HostConfig:      &config.HostConfiguration{Hostname: "proxy.example.com", Port: 8080},
			AdvertisedPort: 8080,
			User:           config.CredentialString("u"),
			Password:       config.CredentialString("p"),
			HTTPS:          false,
		},
		endpointAntiColision: "abc",
	}
	got, err := conf.replaceURL("http://upstream.example.com/live/123.ts", 0, false)
	if err != nil {
		t.Fatalf("replaceURL() err = %v", err)
	}
	if got == "" {
		t.Fatal("replaceURL() returned empty")
	}
	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("url.Parse(got) err = %v", err)
	}
	if parsed.Host != "proxy.example.com:8080" {
		t.Errorf("Host = %q, want proxy.example.com:8080", parsed.Host)
	}
}
