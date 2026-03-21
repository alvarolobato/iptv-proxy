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

package server

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alvarolobato/iptv-proxy/pkg/config"
	"github.com/alvarolobato/iptv-proxy/pkg/stats"
	xtreamapi "github.com/alvarolobato/iptv-proxy/pkg/xtream-proxy"
	xtreamcodes "github.com/tellytv/go.xtream-codes"
	"github.com/gin-gonic/gin"
	"github.com/jamesnetherton/m3u"
	uuid "github.com/satori/go.uuid"
)

// (rest of file omitted here for brevity in this snippet)
