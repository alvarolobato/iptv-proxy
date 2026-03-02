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
	Short: "Reverse proxy on iptv m3u file and xtream codes server api",
	Run: func(cmd *cobra.Command, args []string) {
		log.Printf("[iptv-proxy] INFO: Server is starting...")
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
			RemoteURL:                 remoteHostURL,
			XtreamUser:                config.CredentialString(xtreamUser),
			XtreamPassword:            config.CredentialString(xtreamPassword),
			XtreamBaseURL:             xtreamBaseURL,
			M3UCacheExpiration:        viper.GetInt("m3u-cache-expiration"),
			XMLTVCacheTTL:             parseDuration(viper.GetString("xmltv-cache-ttl")),
			XMLTVCacheMaxEntries:      viper.GetInt("xmltv-cache-max-entries"),
			User:                      config.CredentialString(viper.GetString("user")),
			Password:                  config.CredentialString(viper.GetString("password")),
			AdvertisedPort:            viper.GetInt("advertised-port"),
			HTTPS:                     viper.GetBool("https"),
			M3UFileName:               viper.GetString("m3u-file-name"),
			CustomEndpoint:            viper.GetString("custom-endpoint"),
			CustomId:                  viper.GetString("custom-id"),
			XtreamGenerateApiGet:      viper.GetBool("xtream-api-get"),
			GroupRegex:                viper.GetString("group-regex"),
			ChannelRegex:              viper.GetString("channel-regex"),
			JSONFolder:                viper.GetString("json-folder"),
			DivideByRes:               viper.GetBool("divide-by-res"),
			UseXtreamAdvancedParsing:   viper.GetBool("use-xtream-advanced-parsing"),
			DebugLoggingEnabled:       viper.GetBool("debug-logging"),
			CacheFolder:               cacheFolder,
		}

		if conf.AdvertisedPort == 0 {
			conf.AdvertisedPort = conf.HostConfig.Port
		}

		server, err := server.NewServer(conf)
		if err != nil {
			log.Fatal(err)
		}

		if e := server.Serve(); e != nil {
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "iptv-proxy-config", "C", "Config file (default is $HOME/.iptv-proxy.yaml)")
	rootCmd.Flags().StringP("m3u-url", "u", "", `Iptv m3u file or url e.g: "http://example.com/iptv.m3u"`)
	rootCmd.Flags().StringP("m3u-file-name", "", "iptv.m3u", `Name of the new proxified m3u file e.g "http://poxy.com/iptv.m3u"`)
	rootCmd.Flags().StringP("custom-endpoint", "", "", `Custom endpoint "http://poxy.com/<custom-endpoint>/iptv.m3u"`)
	rootCmd.Flags().StringP("custom-id", "", "", `Custom anti-collison ID for each track "http://proxy.com/<custom-id>/..."`)
	rootCmd.Flags().Int("port", 8080, "Iptv-proxy listening port")
	rootCmd.Flags().Int("advertised-port", 0, "Port to expose the IPTV file and xtream (by default, it's taking value from port) useful to put behind a reverse proxy")
	rootCmd.Flags().String("hostname", "", "Hostname or IP to expose the IPTVs endpoints")
	rootCmd.Flags().BoolP("https", "", false, "Activate https for urls proxy")
	rootCmd.Flags().String("user", "usertest", "User auth to access proxy (m3u/xtream)")
	rootCmd.Flags().String("password", "passwordtest", "Password auth to access proxy (m3u/xtream)")
	rootCmd.Flags().String("xtream-user", "", "Xtream-code user login")
	rootCmd.Flags().String("xtream-password", "", "Xtream-code password login")
	rootCmd.Flags().String("xtream-base-url", "", "Xtream-code base url e.g(http://expample.tv:8080)")
	rootCmd.Flags().Int("m3u-cache-expiration", 1, "M3U cache expiration in hour")
	rootCmd.Flags().String("xmltv-cache-ttl", "", "XMLTV cache TTL (e.g. 1h, 30m); empty = no cache")
	rootCmd.Flags().Int("xmltv-cache-max-entries", 100, "Max XMLTV cache entries (evicts oldest when full)")
	rootCmd.Flags().BoolP("xtream-api-get", "", false, "Generate get.php from xtream API instead of get.php original endpoint")
	rootCmd.Flags().String("group-regex", "", "Include only M3U tracks whose group-title matches this regex (empty = all)")
	rootCmd.Flags().String("channel-regex", "", "Include only M3U tracks whose channel name matches this regex (empty = all)")
	rootCmd.Flags().String("json-folder", "", "Folder containing replacements.json for name/group replacement rules")
	rootCmd.Flags().Bool("divide-by-res", false, "Divide groups by resolution (FHD/HD/SD)")
	rootCmd.Flags().Bool("debug-logging", false, "Enable debug logging")
	rootCmd.Flags().String("cache-folder", "", "Folder to save provider/client responses for debugging (optional)")
	rootCmd.Flags().Bool("use-xtream-advanced-parsing", false, "Use alternate Xtream parsing to preserve raw provider response")

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

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
