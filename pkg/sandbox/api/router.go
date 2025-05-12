package api

import (
	"html/template"
	"net/http"

	"kormsen.com/machine-ui/pkg/sandbox/models"
)

// NewRouter creates and configures the main HTTP router for the application.
// It sets up static file serving and registers handlers for application routes.
func NewRouter(staticDir string, templateSet *template.Template, components []models.ComponentGroup) *http.ServeMux {
	router := http.NewServeMux()

	// Initialize handlers with dependencies
	appHandlers := &AppHandlers{
		Components: components,
		Templates:  templateSet,
	}

	// Serve static files
	fs := http.FileServer(http.Dir(staticDir))
	router.Handle("/static/", http.StripPrefix("/static/", fs))

	// Register application routes
	router.HandleFunc("/", appHandlers.Home)
	router.HandleFunc("/sandbox/", appHandlers.Home) // Redirect /sandbox/ to / to show component list
	router.HandleFunc("/sandbox/{componentName}", appHandlers.ViewStory)
	router.HandleFunc("/sandbox/{componentName}/{storyKey}", appHandlers.ViewStory)

	// New Universal Endpoint for Iframe Content (Dynamic)
	router.HandleFunc("/sandbox-content/{componentName}", appHandlers.ServeSandboxContent)
	router.HandleFunc("/sandbox-content/{componentName}/{storyKey}", appHandlers.ServeSandboxContent)

	// Removed old iframe content routes:
	// router.HandleFunc("/sandbox-frame-csr", appHandlers.ServeCSRFramePage) // Remove old
	// router.HandleFunc("/sandbox-ssr-content/{componentName}/{storyKey}", appHandlers.ServeSSRStoryContent) // Remove old

	// Route for full body content swapping (e.g., for theme changes)
	router.HandleFunc("/sandbox-body-swap/", appHandlers.ServeFullBodyContent) // Trailing slash for path prefix matching

	return router
}
