package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/buger/jsonparser"
	xtream "github.com/tellytv/go.xtream-codes"
)

func init() {
	// Ensure advanced parsing is enabled for tests
	os.Setenv("USE_XTREAM_ADVANCED_PARSING", "true")
}

// TestAdvancedParsingResponses tests the advanced parsing code paths
func TestAdvancedParsingResponses(t *testing.T) {
	tests := []struct {
		name     string
		jsonFile string
		setupFn  func([]byte) (interface{}, error)
	}{
		{
			name:     "GetVideoOnDemandInfo Advanced Parsing",
			jsonFile: "get_vod_info.json",
			setupFn: func(data []byte) (interface{}, error) {
				// Done
				vodInfo := &xtream.VideoOnDemandInfo{
					Fields: data,
				}
				return vodInfo, nil
			},
		},
		{
			name:     "GetSeriesInfo Advanced Parsing",
			jsonFile: "get_series_info.json",
			setupFn: func(data []byte) (interface{}, error) {
				// Done
				seriesInfo := &xtream.Series{
					Fields: data,
				}
				return seriesInfo, nil
			},
		},
		{
			name:     "GetSeries Advanced Parsing",
			jsonFile: "get_series.json",
			setupFn: func(data []byte) (interface{}, error) {
				// Done
				seriesInfos := make([]xtream.SeriesInfo, 0)

				_, jsonErr := jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					if err != nil {
						return
					}
					seriesInfo := xtream.SeriesInfo{
						Fields: value,
					}

					seriesInfos = append(seriesInfos, seriesInfo)
				})

				return seriesInfos, jsonErr
			},
		},
		{
			name:     "GetCategories Advanced Parsing",
			jsonFile: "get_live_categories.json",
			setupFn: func(data []byte) (interface{}, error) {
				// Done
				cats := make([]xtream.Category, 0)

				_, jsonErr := jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					if err != nil {
						return
					}

					cat := xtream.Category{
						Fields: value, // Use the modified value
					}
					cats = append(cats, cat)
				})
				return cats, jsonErr
			},
		},
		{
			name:     "GetStreams Advanced Parsing",
			jsonFile: "get_live_streams.json",
			setupFn: func(data []byte) (interface{}, error) {
				streams := make([]xtream.Stream, 0)

				_, jsonErr := jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
					if err != nil {
						return
					}

					stream := xtream.Stream{
						Fields: value,
					}

					// Extract stream_id and stream_type to maintain compatibility
					streamID, _, _, err := jsonparser.Get(value, xtream.StreamFieldID)
					if err != nil {
						streamID = []byte("0")
					}
					streamType, _, _, err := jsonparser.Get(value, xtream.StreamFieldType)
					if err != nil {
						streamType = []byte("live")
					}

					// Set the basic fields
					var flexID xtream.FlexInt
					flexID.UnmarshalJSON(streamID)
					_ = xtream.Stream{
						ID:   flexID,
						Type: string(streamType),
					}

					streams = append(streams, stream)
				})

				return streams, jsonErr
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read the cached JSON file
			data, err := readCachedJSON(tt.jsonFile)
			if err != nil {
				t.Fatalf("Failed to read cached JSON file: %v", err)
			}

			// Setup the test data using the provided setup function
			response, err := tt.setupFn(data)
			if err != nil {
				t.Fatalf("Failed to setup test data: %v", err)
			}

			// Process the response using the exported ProcessResponse function
			processed := ProcessResponse(response)

			// Verify the processed response
			if processed == nil {
				t.Error("Processed response should not be nil")
			}

			// Marshal the processed response
			readableJSON, err := json.Marshal(processed)
			if err != nil {
				t.Errorf("Failed to marshal processed response: %v", err)
			} else {
				// Print first 150 characters of marshaled JSON
				preview := string(readableJSON)
				if len(preview) > 150 {
					preview = preview[:150] + "..."
				}
				t.Logf("Marshaled JSON preview for %s:\n%s", tt.name, preview)
			}

			// Type-specific verifications
			switch v := processed.(type) {
			case []interface{}:
				t.Logf("Processed array: %T  having length: %d", v, len(v))
				if len(v) == 0 {
					t.Error("Processed array should not be empty")
				}
			case map[string]interface{}:
				t.Logf("Processed map: %T  having length: %d", v, len(v))
				if len(v) == 0 {
					t.Error("Processed map should not be empty")
				}
			default:
				t.Errorf("Unexpected type returned: %T", processed)
			}
		})
	}
}

// readCachedJSON reads a JSON file from the cache directory
func readCachedJSON(filename string) ([]byte, error) {
	cacheDir := "xtreamHandles_test_data/" // Update this path to match your cache directory
	filepath := filepath.Join(cacheDir, filename)
	return os.ReadFile(filepath)
}
