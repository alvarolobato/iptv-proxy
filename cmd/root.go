/*
 * Iptv-Proxy is a project to proxyfie an m3u file and to proxyfie an Xtream iptv service (client API).
 * Copyright (C) 2020  Pierre-Emmanuel Jacquier
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alvarolobato/iptv-proxy/pkg/config"

	"github.com/alvarolobato/iptv-proxy/pkg/server"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "iptv-proxy",
	Short: "Reverse proxy for IPTV M3U playlists and Xtream Codes API",
	Long:  `IPTV-Proxy fetches your provider's playlist and rewrites URLs to point to this server. Input is either an M3U URL (--m3u-url) or Xtream credentials (--xtream-user, --xtream-password, --xtream-base-url). Set --hostname and optionally --data-folder for settings and the configuration UI.`,
	Example: `  # Input type: M3U URL — proxy on port 8080
  iptv-proxy --m3u-url "http://example.com/get.php?username=user&password=pass&type=m3u_plus&output=m3u8" --hostname localhost --port 8080

  # Input type: M3U URL with auth and data folder (settings.json, UI)
  iptv-proxy --m3u-url "http://example.com/get.php?username=user&password=pass&type=m3u_plus&output=m3u8" --hostname localhost --user myuser --password mypass --data-folder ./data --ui-port 9090

  # Using config file and env (e.g. Docker)
  IPTV_PROXY_M3U_URL="http://example.com/get.php?username=user&password=pass&type=m3u_plus&output=m3u8" IPTV_PROXY_HOSTNAME=localhost iptv-proxy --data-folder /data`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("[iptv-proxy] Server is starting...")
		m3uURL := viper.GetString("m3u-url")
		remoteHostURL, err := url.Parse(m3uURL)
		if err != nil {
			log.Fatal(err)
		}

		xtreamUser := viper.GetString("xtream-user")
		xtreamPassword := viper.GetString("xtream-password")
		xtreamBaseURL := viper.GetString("xtream-base-url")

		var username, password string
		if strings.Contains(m3uURL, "/get.php") {
			username = remoteHostURL.Query().Get("username")
			password = remoteHostURL.Query().Get("password")
		}

		if xtreamBaseURL == "" && xtreamPassword == "" && xtreamUser == "" {
			if username != "" && password != "" {
				log.Printf("[iptv-proxy] INFO: It's seams you are using an Xtream provider!")

				xtreamUser = username
				xtreamPassword = password
				xtreamBaseURL = fmt.Sprintf("%s://%s", remoteHostURL.Scheme, remoteHostURL.Host)
				log.Printf("[iptv-proxy] INFO: xtream service enable with xtream base url: %q xtream username: %q xtream password: %q", xtreamBaseURL, xtreamUser, xtreamPassword)
			}
		}

		cacheFolder := viper.GetString("cache-folder")
		if cacheFolder != "" {
			cacheFolder = filepath.Clean(cacheFolder)
		}

		conf := &config.ProxyConfig{
			HostConfig: &config.HostConfiguration{
				Hostname: viper.GetString("hostname"),
				Port:     viper.GetInt("port"),
			},
			RemoteURL:                remoteHostURL,
			XtreamUser:               config.CredentialString(xtreamUser),
			XtreamPassword:           config.CredentialString(xtreamPassword),
			XtreamBaseURL:            xtreamBaseURL,
			M3UCacheExpiration:       viper.GetInt("m3u-cache-expiration"),
			XMLTVCacheTTL:            parseDuration(viper.GetString("xmltv-cache-ttl")),
			XMLTVCacheMaxEntries:     viper.GetInt("xmltv-cache-max-entries"),
			User:                     config.CredentialString(viper.GetString("user")),
			Password:                 config.CredentialString(viper.GetString("password")),
			AdvertisedPort:           viper.GetInt("advertised-port"),
			HTTPS:                    viper.GetBool("https"),
			M3UFileName:              viper.GetString("m3u-file-name"),
			CustomEndpoint:           viper.GetString("custom-endpoint"),
			CustomId:                 viper.GetString("custom-id"),
			XtreamGenerateApiGet:     viper.GetBool("xtream-api-get"),
			DataFolder:               viper.GetString("data-folder"),
			DivideByRes:              viper.GetBool("divide-by-res"),
			UseXtreamAdvancedParsing: viper.GetBool("use-xtream-advanced-parsing"),
			DebugLoggingEnabled:      viper.GetBool("debug-logging"),
			CacheFolder:              cacheFolder,
			UIPort:                   viper.GetInt("ui-port"),
		}

		if conf.AdvertisedPort == 0 {
			conf.AdvertisedPort = conf.HostConfig.Port
		}

		defaultForSettings := config.CurrentFromProxyConfig(conf)

		var settings *config.SettingsJSON
		var overriddenBySettings []string
		if conf.DataFolder != "" {
			config.EnsureStubSettings(conf.DataFolder)
			var errLoad error
			settings, errLoad = config.LoadSettings(conf.DataFolder)
			if errLoad != nil {
				log.Printf("[iptv-proxy] WARN: could not load settings.json: %v", errLoad)
			} else if settings != nil {
				overriddenBySettings = config.ApplyTo(settings, conf, parseDuration)
			}
		}

		startupCtx := buildStartupContext(conf, settings, overriddenBySettings, viper.ConfigFileUsed(), viper.GetBool("hide-passwords"))

		srv, err := server.NewServer(conf, settings, &defaultForSettings)
		if err != nil {
			log.Fatal(err)
		}

		if e := srv.ServeWithContext(startupCtx); e != nil {
			log.Fatal(e)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "iptv-proxy-config", "", "Config file path (default $HOME/.iptv-proxy.yaml)")
	// Most-used / mandatory first
	rootCmd.Flags().StringP("m3u-url", "u", "", "Input type: M3U URL. Use this or Xtream flags (--xtream-user, etc.). Example: http://example.com/get.php?username=user&password=pass&type=m3u_plus&output=m3u8")
	rootCmd.Flags().String("hostname", "", "Hostname or IP used in generated playlist/stream URLs (set for correct links)")
	rootCmd.Flags().Int("port", 8080, "Port the proxy listens on")
	rootCmd.Flags().String("user", "", "Proxy auth username (M3U and Xtream); set via flag, env, or Settings UI")
	rootCmd.Flags().String("password", "", "Proxy auth password (M3U and Xtream); set via flag, env, or Settings UI")
	rootCmd.Flags().String("data-folder", "", "Folder for settings.json and replacement rules (enables UI when --ui-port set)")
	rootCmd.Flags().Int("ui-port", 8081, "Port for configuration UI (default 8081, one above proxy port 8080); set 0 to disable")
	rootCmd.Flags().Bool("hide-passwords", false, "Hide passwords in startup log and URL examples")
	// Alphabetical
	rootCmd.Flags().Int("advertised-port", 0, "Port in generated URLs (default = port); set when behind reverse proxy (e.g. 443)")
	rootCmd.Flags().String("cache-folder", "", "Folder to save provider/client responses (debug)")
	rootCmd.Flags().String("custom-endpoint", "", "Path prefix for M3U (e.g. api → …/api/iptv.m3u)")
	rootCmd.Flags().String("custom-id", "", "Anti-collision path segment for track URLs")
	rootCmd.Flags().Bool("debug-logging", false, "Enable verbose debug logging")
	rootCmd.Flags().Bool("divide-by-res", false, "Add resolution suffix to groups (FHD/HD/SD) and strip from names")
	rootCmd.Flags().Bool("https", false, "Use https in generated URLs")
	rootCmd.Flags().String("m3u-file-name", "iptv.m3u", "Filename of the proxified playlist")
	rootCmd.Flags().Int("m3u-cache-expiration", 1, "M3U cache TTL in hours (Xtream-generated M3U)")
	rootCmd.Flags().Bool("use-xtream-advanced-parsing", false, "Use alternate Xtream response parsing for some providers")
	rootCmd.Flags().String("xtream-base-url", "", "Input type: Xtream — base URL (e.g. http://provider.tv:8080). Use when not using --m3u-url.")
	rootCmd.Flags().String("xtream-password", "", "Input type: Xtream — provider password (can be inferred from get.php URL)")
	rootCmd.Flags().String("xtream-user", "", "Input type: Xtream — provider username (can be inferred from get.php URL)")
	rootCmd.Flags().Bool("xtream-api-get", false, "Serve get.php from Xtream API instead of provider endpoint")
	rootCmd.Flags().String("xmltv-cache-ttl", "", "XMLTV (EPG) cache TTL (e.g. 1h, 30m); empty = no cache")
	rootCmd.Flags().Int("xmltv-cache-max-entries", 100, "Max cached XMLTV responses")

	if e := viper.BindPFlags(rootCmd.Flags()); e != nil {
		log.Fatal("error binding PFlags to viper")
	}
}

func parseDuration(s string) time.Duration {
	if s == "" {
		return 0
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		log.Printf("[iptv-proxy] WARN: invalid duration %q: %v; using 0", s, err)
		return 0
	}
	return d
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".iptv-proxy" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigName(".iptv-proxy")
	}

	viper.SetEnvPrefix("IPTV_PROXY")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	viper.AutomaticEnv() // read in environment variables that match (e.g. IPTV_PROXY_USER, not USER)

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}

// buildStartupContext builds context for the server-ready log (data folder, files, overrides).
func buildStartupContext(conf *config.ProxyConfig, settings *config.SettingsJSON, overridden []string, configFilePath string, hidePasswords bool) *server.StartupContext {
	ctx := &server.StartupContext{
		HidePasswords:        hidePasswords,
		ConfigFilePath:       configFilePath,
		OverriddenBySettings: overridden,
	}
	if conf.DataFolder != "" {
		ctx.DataFolder = conf.DataFolder
		ctx.SettingsPath = filepath.Join(conf.DataFolder, "settings.json")
		if settings != nil {
			ctx.SettingsPresent = true
			if settings.Replacements != nil {
				ctx.ReplacementsInFile = "settings.json"
				ctx.ReplacementCounts.Global = len(settings.Replacements.Global)
				ctx.ReplacementCounts.Names = len(settings.Replacements.Names)
				ctx.ReplacementCounts.Groups = len(settings.Replacements.Groups)
			}
		} else {
			legacyPath := filepath.Join(conf.DataFolder, "replacements.json")
			if _, err := os.Stat(legacyPath); err == nil {
				ctx.SettingsPresent = false
				ctx.ReplacementsInFile = "replacements.json"
				// Count would require reading the file; leave 0 or try read
				if data, err := os.ReadFile(legacyPath); err == nil {
					var raw struct {
						Global []interface{} `json:"global-replacements"`
						Names  []interface{} `json:"names-replacements"`
						Groups []interface{} `json:"groups-replacements"`
					}
					if json.Unmarshal(data, &raw) == nil {
						ctx.ReplacementCounts.Global = len(raw.Global)
						ctx.ReplacementCounts.Names = len(raw.Names)
						ctx.ReplacementCounts.Groups = len(raw.Groups)
					}
				}
			}
		}
	}
	return ctx
}
