// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package hsts provides HTTP Strict Transport Security.
//
// HTTP Strict Transport Security informs browsers that a website
// should only be accessed using HTTPS and not HTTP. This plugin enforces HSTS by
// redirecting all HTTP traffic to HTTPS and by setting the
// Strict-Transport-Security header on all HTTPS responses. Please note that this
// only applies if the framework is not run in dev mode.
//
// More info:
//   - MDN: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security
//   - Wikipedia: https://en.wikipedia.org/wiki/HTTP_Strict_Transport_Security
//   - RFC 6797: https://tools.ietf.org/html/rfc6797
//
// # Usage
//
// To construct the plugin with safe default settings, use Default. Otherwise,
// create the Interceptor yourself.
// Install it using safehttp.ServerMux.Install.
package hsts

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-safeweb/safehttp"
)

// Interceptor implements automatic HSTS functionality.
// See https://tools.ietf.org/html/rfc6797 for more info.
type Interceptor struct {
	// MaxAge is the duration that the browser should remember
	// that a site is only to be accessed using HTTPS. MaxAge
	// must be positive. It will be rounded to seconds before use.
	MaxAge time.Duration

	// DisableIncludeSubDomains disables the includeSubDomains directive.
	// When DisableIncludeSubDomains is false, all subdomains
	// of the domain where this service is hosted will also be added
	// to the browsers HSTS list.
	DisableIncludeSubDomains bool

	// Preload enables the preload directive.
	// This should only be enabled if this site should be
	// added to the browser HSTS preload list, which is supported
	// by all major browsers. See https://hstspreload.org/ for
	// more info.
	Preload bool

	// BehindProxy controls how the plugin should behave with regards
	// to HTTPS. If this server is behind a proxy that terminates HTTPS
	// traffic then this should be enabled. If this is enabled
	// then the plugin will always send the Strict-Transport-Security
	// header and will not redirect HTTP traffic to HTTPS traffic.
	BehindProxy bool
}

var _ safehttp.Interceptor = Interceptor{}

// Default creates a new HSTS interceptor with safe defaults.
// These safe defaults are:
//   - max-age set to 2 years,
//   - includeSubDomains is enabled,
//   - preload is disabled.
func Default() Interceptor {
	return Interceptor{MaxAge: 63072000 * time.Second} // two years in seconds
}

// Before should be executed before the request is sent to the handler.
// The function redirects HTTP requests to HTTPS. When HTTPS traffic
// is received the Strict-Transport-Security header is applied to the
// response.
func (it Interceptor) Before(w safehttp.ResponseWriter, r *safehttp.IncomingRequest, _ safehttp.InterceptorConfig) safehttp.Result {
	if safehttp.IsLocalDev() {
		return safehttp.NotWritten()
	}

	if it.MaxAge < 0 {
		return w.WriteError(safehttp.StatusInternalServerError)
	}

	if !it.BehindProxy && r.TLS == nil {
		u, err := url.Parse(r.URL().String())
		if err != nil {
			return w.WriteError(safehttp.StatusInternalServerError)
		}
		u.Scheme = "https"
		return safehttp.Redirect(w, r, u.String(), safehttp.StatusMovedPermanently)
	}

	var value strings.Builder
	value.WriteString("max-age=")
	value.WriteString(strconv.FormatInt(int64(it.MaxAge.Seconds()), 10))
	if !it.DisableIncludeSubDomains {
		value.WriteString("; includeSubDomains")
	}
	if it.Preload {
		value.WriteString("; preload")
	}
	set := w.Header().Claim("Strict-Transport-Security")
	set([]string{value.String()})
	return safehttp.NotWritten()
}

// Commit is a no-op, required to satisfy the safehttp.Interceptor interface.
func (Interceptor) Commit(w safehttp.ResponseHeadersWriter, r *safehttp.IncomingRequest, resp safehttp.Response, _ safehttp.InterceptorConfig) {
}

// Match returns false since there are no supported configurations.
func (Interceptor) Match(safehttp.InterceptorConfig) bool {
	return false
}
