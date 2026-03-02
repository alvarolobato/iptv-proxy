package xtreamcodes

import (
	"encoding/json"
	"log"
	"testing"
)

func TestAuthenticationResponseUnmarshal(t *testing.T) {
	log.Printf("TestAuthenticationResponseUnmarshal: called")
	jsonData := `{
		"user_info": {
			"username": "myxtreamuser",
			"password": "myxtreampass",
			"message": "Less is more..",
			"auth": 1,
			"status": "Active",
			"exp_date": "1733423929",
			"is_trial": "0",
			"active_cons": 0,
			"created_at": "1725565129",
			"max_connections": "4",
			"allowed_output_formats": [
				"m3u8",
				"ts"
			]
		},
		"server_info": {
			"url": "provider.com",
			"port": "80",
			"https_port": "443",
			"server_protocol": "https",
			"rtmp_port": "30002",
			"timezone": "Pacific/Easter",
			"timestamp_now": 1729207595,
			"time_now": "2024-10-17 18:26:35",
			"process": true
		}
	}`

	var authResponse AuthenticationResponse
	err := json.Unmarshal([]byte(jsonData), &authResponse)
	if err != nil {
		t.Fatalf("Failed to unmarshal AuthenticationResponse: %v", err)
	}

	// Pretty print the entire AuthenticationResponse
	prettyJSON, err := json.MarshalIndent(authResponse, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal AuthenticationResponse to pretty JSON: %v", err)
	}
	log.Printf("AuthenticationResponse:\n%s", string(prettyJSON))

	// Pretty print UserInfo
	prettyUserInfo, err := json.MarshalIndent(authResponse.UserInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal UserInfo to pretty JSON: %v", err)
	}
	log.Printf("UserInfo:\n%s", string(prettyUserInfo))

	// Pretty print ServerInfo
	prettyServerInfo, err := json.MarshalIndent(authResponse.ServerInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal ServerInfo to pretty JSON: %v", err)
	}
	log.Printf("ServerInfo:\n%s", string(prettyServerInfo))
}
