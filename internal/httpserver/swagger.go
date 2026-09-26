package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"
)

// swaggerInitializerJS is the custom swagger-initializer.js payload.
//
// The requestInterceptor rewrites every request URL so it always uses the
// current page's origin (window.location.protocol + window.location.host).
// This fixes two common Swagger UI v3+ problems:
//
//  1. Swagger UI ignores the "schemes" field and uses the browser's own
//     protocol — so https://localhost fails against a plain-HTTP dev server.
//
//  2. The OpenAPI spec "host" field may be stale (e.g. "localhost:5000") when
//     the API is accessed via a different host (e.g. a Render deployment).
const swaggerInitializerJS = `
window.onload = function() {
  const ui = SwaggerUIBundle({
    url: "doc.json",
    dom_id: '#swagger-ui',
    validatorUrl: null,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    plugins: [
      SwaggerUIBundle.plugins.DownloadUrl
    ],
    layout: "StandaloneLayout",
    docExpansion: "list",
    deepLinking: true,
    defaultModelsExpandDepth: 1,
    persistAuthorization: true,

    // Always rewrite the request origin to match the current page's origin.
    // This works in every environment: local dev (http), Render/ngrok (https), etc.
    requestInterceptor: function(request) {
      try {
        var parsed   = new URL(request.url);
        var pageOrigin = window.location.protocol + '//' + window.location.host;
        var specOrigin = parsed.protocol + '//' + parsed.host;
        if (specOrigin !== pageOrigin) {
          request.url = pageOrigin + parsed.pathname + parsed.search + parsed.hash;
        }
      } catch(e) {}
      return request;
    },
  })

  window.ui = ui
}
`

// newSwaggerRouter returns a gin.HandlerFunc that wraps the standard
// gin-swagger wildcard handler but intercepts two paths:
//
//   - swagger-initializer.js → serves our custom initializer (requestInterceptor fix)
//   - doc.json               → serves the live OpenAPI spec, dynamically
//     patching "host" and "schemes" from the real incoming request headers
//
// Using a single wildcard avoids Gin radix-tree conflicts.
func newSwaggerRouter(fallback gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := strings.TrimPrefix(c.Param("any"), "/")

		switch path {
		case "swagger-initializer.js":
			c.Header("Content-Type", "application/javascript")
			c.Header("Cache-Control", "no-store")
			fmt.Fprint(c.Writer, swaggerInitializerJS)

		case "doc.json":
			serveDocJSON(c)

		default:
			fallback(c)
		}
	}
}

// serveDocJSON reads the registered OpenAPI spec from swaggo, dynamically
// patches "host" and "schemes" to match the actual incoming request, then
// writes the result as JSON.
//
// This is critical for proxied/cloud deployments (Render, Railway, Fly, ngrok)
// where the compile-time @host annotation ("localhost:5000") does not match
// the publicly visible hostname.
func serveDocJSON(c *gin.Context) {
	raw, err := swag.ReadDoc(swag.Name)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Unmarshal into a generic map so we can patch individual fields.
	var spec map[string]any
	if jsonErr := json.Unmarshal([]byte(raw), &spec); jsonErr != nil {
		// Fall back to serving the raw spec as-is.
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.String(http.StatusOK, raw)
		return
	}

	// Determine the actual host visible to the client.
	// Render (and most reverse proxies) sets X-Forwarded-Host.
	host := c.GetHeader("X-Forwarded-Host")
	if host == "" {
		host = c.Request.Host
	}
	if host != "" {
		spec["host"] = host
	}

	// Determine the actual scheme.
	// Render sets X-Forwarded-Proto when it terminates TLS at the edge.
	scheme := c.GetHeader("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	// Only keep the first value if the header contains a comma-separated list.
	if idx := strings.Index(scheme, ","); idx != -1 {
		scheme = strings.TrimSpace(scheme[:idx])
	}
	spec["schemes"] = []string{scheme}

	c.Header("Content-Type", "application/json; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	if out, jsonErr := json.Marshal(spec); jsonErr == nil {
		c.String(http.StatusOK, string(out))
	} else {
		c.String(http.StatusOK, raw)
	}
}
