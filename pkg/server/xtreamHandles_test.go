package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/buger/jsonparser"
	xtream "github.com/tellytv/go.xtream-codes"
)

var useAdvancedParsing bool

func init() {
	useAdvancedParsing = os.Getenv("USE_XTREAM_ADVANCED_PARSING") == "true"
}

// TestAdvancedParsingResponses tests the advanced parsing code paths
func TestAdvancedParsingResponses(t *testing.T) {
	tests := []struct {
		name     string
		jsonFile string
		setupFn  func([]byte) (interface{}, error)
	}{
		{
			name: "GetVideoOnDemandInfo Advanced Parsing",
			// jsonFile2: "get_vod_info.json",
			jsonFile: "get_vod_info_1152184.json",
			setupFn: func(data []byte) (interface{}, error) {
				// Done
				if useAdvancedParsing {
					t.Logf("- GetVideoOnDemandInfo using Advanced Parsing for: VideoOnDemandInfo")

					vodInfo := &xtream.VideoOnDemandInfo{
						Fields: data,
					}

					return vodInfo, nil
				} else {
					t.Logf("- GetVideoOnDemandInfo using Legacy Parsing for: VideoOnDemandInfo")

					// data, _ = jsonparser.Set(data, []byte(`""`), "info", "bitrate")

					bitrate, _, _, _ := jsonparser.Get(data, "info", "bitrate")
					tmdb_id, _, _, _ := jsonparser.Get(data, "info", "tmdb_id")
					t.Logf("- GetVideoOnDemandInfo bitrate(%T): %s, tmdb_id(%T): %s", bitrate, string(bitrate), tmdb_id, string(tmdb_id))

					vodInfo := &xtream.VideoOnDemandInfo{}

					jsonErr := json.Unmarshal(data, &vodInfo)
					if jsonErr != nil {
						t.Fatal("- GetVideoOnDemandInfo unmarshalling VideoOnDemandInfo - error: " + jsonErr.Error())
					}
					t.Logf("- GetVideoOnDemandInfo bitrate: %v, tmdb_id: %v", vodInfo.Info.Bitrate, vodInfo.Info.TmdbID)

					return vodInfo, jsonErr
				}
			},
		},
		{
			name:     "GetSeriesInfo Advanced Parsing",
			jsonFile: "get_series_info.json",
			setupFn: func(data []byte) (interface{}, error) {
				// Done
				if useAdvancedParsing {
					t.Logf("- GetSeriesInfo using Advanced Parsing for: Series")

					seriesInfo := &xtream.Series{
						Fields: data,
					}

					return seriesInfo, nil
				} else {
					t.Logf("- GetSeriesInfo using Legacy Parsing for: Series")

					seriesInfo := &xtream.Series{}

					jsonErr := json.Unmarshal(data, &seriesInfo)

					return seriesInfo, jsonErr
				}

			},
		},
		{
			name:     "GetSeries Advanced Parsing",
			jsonFile: "get_series.json",
			setupFn: func(data []byte) (interface{}, error) {
				// Done
				seriesInfos := make([]xtream.SeriesInfo, 0)

				if useAdvancedParsing {
					t.Logf("- GetSeries using Advanced Parsing for: []SeriesInfo")

					_, jsonErr := jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
						if err != nil {
							t.Logf(">> GetSeries - Error iterating through array: %s", err.Error())
							return
						}

						seriesInfo := xtream.SeriesInfo{
							Fields: value,
						}

						seriesInfos = append(seriesInfos, seriesInfo)
					})

					return seriesInfos, jsonErr
				} else {
					t.Logf("- GetSeries using Legacy Parsing for: []SeriesInfo")

					jsonErr := json.Unmarshal(data, &seriesInfos)

					return seriesInfos, jsonErr
				}

			},
		},
		{
			name:     "GetCategories Advanced Parsing",
			jsonFile: "get_live_categories.json",
			setupFn: func(data []byte) (interface{}, error) {
				// Done
				cats := make([]xtream.Category, 0)

				if useAdvancedParsing {
					t.Logf("- GetCategories using Advanced Parsing for: []Category")

					_, jsonErr := jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
						if err != nil {
							t.Logf(">> GetCategories - Error iterating through array: %s", err.Error())
							return
						}

						cat := xtream.Category{
							Fields: value, // Directly assign the raw JSON bytes
						}
						cats = append(cats, cat)
					})

					return cats, jsonErr
				} else {
					t.Logf("- GetCategories using Legacy Parsing for: []Category")

					jsonErr := json.Unmarshal(data, &cats)

					for idx := range cats {
						cats[idx].Type = "live"
					}

					return cats, jsonErr
				}

			},
		},
		{
			name:     "GetStreams Advanced Parsing",
			jsonFile: "get_live_streams.json",
			setupFn: func(data []byte) (interface{}, error) {
				// Done
				streams := make([]xtream.Stream, 0)

				if useAdvancedParsing {
					t.Logf("- GetStreams using Advanced Parsing for: []Stream")

					_, jsonErr := jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
						if err != nil {
							t.Logf(">> GetStreams - Error iterating through array: %s", err.Error())
							return
						}

						stream := xtream.Stream{
							Fields: value,
						}

						streams = append(streams, stream)

						// The below code is to replicate the behavior of the old code:
						//   c.streams[int(stream.ID)] = stream

						streamID, _, _, err := jsonparser.Get(value, xtream.StreamFieldID)
						if err != nil {
							streamID = []byte("0")
						}
						streamType, _, _, err := jsonparser.Get(value, xtream.StreamFieldType)
						if err != nil {
							streamType = []byte("live")
						}

						// Now you can use streamID and streamType as []byte
						// To convert to string if needed:
						streamTypeStr := string(streamType)

						var flexID xtream.FlexInt
						flexID.UnmarshalJSON(streamID)

						_ = xtream.Stream{
							ID:   flexID,
							Type: streamTypeStr,
						}
						// c.streams[int(s.ID)] = s
					})

					return streams, jsonErr
				} else {
					t.Logf("- GetStreams using Legacy Parsing for: []Stream")

					if jsonErr := json.Unmarshal(data, &streams); jsonErr != nil {
						return nil, jsonErr
					}

					// for _, stream := range streams {
					// 	c.streams[int(stream.ID)] = stream
					// }

					return streams, nil
				}

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
			case []interface{}, []xtream.SeriesInfo, []xtream.Category, []xtream.Stream:
				t.Logf("Processed array: %T", v)
				// Use reflection to get length since we have multiple slice types
				length := reflect.ValueOf(v).Len()
				t.Logf("Array length: %d", length)
				if length == 0 {
					t.Error("Processed array should not be empty")
				}
			case map[string]interface{}, *xtream.Series, *xtream.VideoOnDemandInfo:
				t.Logf("Processed object: %T", v)
				// For map type we can check fields/length
				if m, ok := v.(map[string]interface{}); ok {
					t.Logf("Map length: %d", len(m))
					if len(m) == 0 {
						t.Error("Processed map should not be empty")
					}
				}
				// For struct types, we just verify it's not nil
				if reflect.ValueOf(v).IsNil() {
					t.Error("Processed struct should not be nil")
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
