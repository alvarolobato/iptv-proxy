package xtreamcodes

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"testing"
)

// -- Helper Functions --
// loadTestData loads JSON test data from the structs_test_data folder
func loadTestData(filename string) ([]byte, error) {
	path := filepath.Join("structs_test_data", filename)
	return os.ReadFile(path)
}

// withEnv sets a single environment variable for the duration of a function
func withEnv(key, value string, f func()) {
	oldValue := os.Getenv(key)
	os.Setenv(key, value)
	defer os.Setenv(key, oldValue)
	f()
}

// withEnvs sets multiple environment variables for the duration of a function
func withEnvs(envVars map[string]string, f func()) {
	// Save old values
	oldValues := make(map[string]string)
	for key, value := range envVars {
		oldValues[key] = os.Getenv(key)
		os.Setenv(key, value)
	}

	// Defer restoration of old values
	defer func() {
		for key, oldValue := range oldValues {
			os.Setenv(key, oldValue)
		}
	}()

	// Run the function
	f()
}

// -- Tests --
func TestAuthenticationResponseUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("authentication_response.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "authentication_response.json")

	var authResponse AuthenticationResponse
	// Using withEnvs for multiple environment variables
	withEnvs(map[string]string{"TEST_ALL_VALUES": "true", "DETECT_NEW_FIELDS": "true"}, func() {
		err = json.Unmarshal(jsonData, &authResponse)
		if err != nil {
			t.Fatalf("Failed to unmarshal AuthenticationResponse: %v", err)
		}
	})

	// Pretty print the entire AuthenticationResponse
	prettyJSON, err := json.MarshalIndent(authResponse, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal AuthenticationResponse to pretty JSON: %v", err)
	}
	log.Printf("AuthenticationResponse:\n%s", string(prettyJSON))
}

func TestServerInfoUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("server_info.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "server_info.json")

	var serverInfo ServerInfo
	err = json.Unmarshal(jsonData, &serverInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal ServerInfo: %v", err)
	}

	// Pretty print the entire ServerInfo
	prettyJSON, err := json.MarshalIndent(serverInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal ServerInfo to pretty JSON: %v", err)
	}
	log.Printf("ServerInfo:\n%s", string(prettyJSON))
}

func TestUserInfoUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("user_info.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "user_info.json")

	var userInfo UserInfo
	err = json.Unmarshal(jsonData, &userInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal UserInfo: %v", err)
	}

	// Pretty print the entire UserInfo
	prettyJSON, err := json.MarshalIndent(userInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal UserInfo to pretty JSON: %v", err)
	}
	log.Printf("UserInfo:\n%s", string(prettyJSON))
}

func TestCategoryUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("category.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "category.json")

	var category Category
	err = json.Unmarshal(jsonData, &category)
	if err != nil {
		t.Fatalf("Failed to unmarshal Category: %v", err)
	}

	// Pretty print the entire Category
	prettyJSON, err := json.MarshalIndent(category, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal Category to pretty JSON: %v", err)
	}
	log.Printf("Category:\n%s", string(prettyJSON))
}

func TestStreamUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("stream.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "stream.json")

	var stream Stream
	err = json.Unmarshal(jsonData, &stream)
	if err != nil {
		t.Fatalf("Failed to unmarshal Stream: %v", err)
	}

	// Pretty print the entire Stream
	prettyJSON, err := json.MarshalIndent(stream, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal Stream to pretty JSON: %v", err)
	}
	log.Printf("Stream:\n%s", string(prettyJSON))
}

func TestSeriesInfoUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("series_info.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "series_info.json")

	var seriesInfo SeriesInfo
	err = json.Unmarshal(jsonData, &seriesInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal SeriesInfo: %v", err)
	}

	// Pretty print the entire SeriesInfo
	prettyJSON, err := json.MarshalIndent(seriesInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal SeriesInfo to pretty JSON: %v", err)
	}
	log.Printf("SeriesInfo:\n%s", string(prettyJSON))
}

func TestSeriesEpisodeUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("series_episode.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "series_episode.json")

	var seriesEpisode SeriesEpisode
	err = json.Unmarshal(jsonData, &seriesEpisode)
	if err != nil {
		t.Fatalf("Failed to unmarshal SeriesEpisode: %v", err)
	}

	// Pretty print the entire SeriesEpisode
	prettyJSON, err := json.MarshalIndent(seriesEpisode, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal SeriesEpisode to pretty JSON: %v", err)
	}
	log.Printf("SeriesEpisode:\n%s", string(prettyJSON))
}

func TestSeriesUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("series.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "series.json")

	var series Series
	err = json.Unmarshal(jsonData, &series)
	if err != nil {
		t.Fatalf("Failed to unmarshal Series: %v", err)
	}

	// Pretty print the entire Series
	prettyJSON, err := json.MarshalIndent(series, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal Series to pretty JSON: %v", err)
	}
	log.Printf("Series:\n%s", string(prettyJSON))
}

func TestVideoOnDemandInfoUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("video_on_demand_info.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "video_on_demand_info.json")

	var vodInfo VideoOnDemandInfo
	err = json.Unmarshal(jsonData, &vodInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal VideoOnDemandInfo: %v", err)
	}

	// Pretty print the entire VideoOnDemandInfo
	prettyJSON, err := json.MarshalIndent(vodInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal VideoOnDemandInfo to pretty JSON: %v", err)
	}
	log.Printf("VideoOnDemandInfo:\n%s", string(prettyJSON))
}

func TestVODInfoUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("vod_info.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "vod_info.json")

	var vodInfo VODInfo
	err = json.Unmarshal(jsonData, &vodInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal VODInfo: %v", err)
	}

	// Pretty print the entire VODInfo
	prettyJSON, err := json.MarshalIndent(vodInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal VODInfo to pretty JSON: %v", err)
	}
	log.Printf("VODInfo:\n%s", string(prettyJSON))
}

func TestEPGInfoUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("epg_info.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "epg_info.json")

	var epgInfo EPGInfo
	err = json.Unmarshal(jsonData, &epgInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal EPGInfo: %v", err)
	}

	// Pretty print the entire EPGInfo
	prettyJSON, err := json.MarshalIndent(epgInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal EPGInfo to pretty JSON: %v", err)
	}
	log.Printf("EPGInfo:\n%s", string(prettyJSON))
}

func TestEpisodeInfoUnmarshal(t *testing.T) {
	jsonData, err := loadTestData("episode_info.json")
	if err != nil {
		t.Fatalf("Failed to load test data: %v", err)
	}
	log.Printf("Loaded test data file: %s", "episode_info.json")

	var episodeInfo EpisodeInfo
	err = json.Unmarshal(jsonData, &episodeInfo)
	if err != nil {
		t.Fatalf("Failed to unmarshal EpisodeInfo: %v", err)
	}

	// Pretty print the entire EpisodeInfo
	prettyJSON, err := json.MarshalIndent(episodeInfo, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal EpisodeInfo to pretty JSON: %v", err)
	}
	log.Printf("EpisodeInfo:\n%s", string(prettyJSON))
}
