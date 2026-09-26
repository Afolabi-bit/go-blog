package httpserver

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/swag"
)

// swaggerInitializerJS is the custom swagger-initializer.js payload.
// It injects a requestInterceptor into SwaggerUIBundle that rewrites
// https://localhost... → http://localhost... at the JavaScript level.
//
// This is necessary because Swagger UI v3+ ignores the "schemes" field
// in the OpenAPI spec and instead derives the scheme from the browser's
// window.location.protocol. On many browsers localhost gets upgraded to
// https (HSTS, mixed-content policy, etc.) which breaks a plain-HTTP server.
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

    // Force http on localhost regardless of the page's own scheme so that
    // the API server running in plain-HTTP mode is always reachable.
    requestInterceptor: function(request) {
      if (/^https:\/\/localhost/i.test(request.url)) {
        request.url = request.url.replace(/^https:\/\//i, 'http://');
      }
      return request;
    },
  })

  window.ui = ui
}
`

// newSwaggerRouter returns a gin.HandlerFunc that wraps the standard
// gin-swagger wildcard handler but intercepts two paths:
//   - swagger-initializer.js → serves our custom initializer (with http fix)
//   - doc.json               → serves the live OpenAPI spec from swaggo registry
//
// This avoids registering competing specific routes alongside /*any, which
// panics in Gin's radix tree implementation.
func newSwaggerRouter(fallback gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Param("any") // e.g. "/swagger-initializer.js"
		path = strings.TrimPrefix(path, "/")

		switch path {
		case "swagger-initializer.js":
			c.Header("Content-Type", "application/javascript")
			c.Header("Cache-Control", "no-store")
			fmt.Fprint(c.Writer, swaggerInitializerJS)
		case "doc.json":
			doc, err := swag.ReadDoc(swag.Name)
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
			c.Header("Content-Type", "application/json; charset=utf-8")
			c.String(http.StatusOK, doc)
		default:
			fallback(c)
		}
	}
}
