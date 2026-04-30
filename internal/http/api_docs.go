package http

import (
	"net/http"

	swaggerdocs "github.com/ryunosukekurokawa/idol-auth/docs/swagger"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

var swaggerHandler = httpSwagger.Handler(
	httpSwagger.URL("/docs/doc.json"),
	httpSwagger.DocExpansion("list"),
	httpSwagger.DeepLinking(true),
	httpSwagger.PersistAuthorization(true),
	httpSwagger.Layout(httpSwagger.StandaloneLayout),
	httpSwagger.UIConfig(map[string]string{
		"displayRequestDuration": "true",
		"defaultModelsExpandDepth": "1",
	}),
)

func init() {
	// Keep default values predictable for tests and local development.
	swaggerdocs.SwaggerInfo.Title = "idol-auth API"
	swaggerdocs.SwaggerInfo.Version = "1.0.0"
	swaggerdocs.SwaggerInfo.BasePath = "/"
}

func (s *server) handleDocsIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/docs/index.html", http.StatusMovedPermanently)
}

func (s *server) handleDocs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; "+
			"script-src 'self' 'unsafe-inline'; "+
			"style-src 'self' 'unsafe-inline'; "+
			"img-src 'self' data:; "+
			"font-src 'self' data:; "+
			"connect-src 'self'; "+
			"frame-ancestors 'none'; "+
			"base-uri 'self'")
	swaggerHandler(w, r)
}
