package main

import (
	"log"
	"net/http"

	"kormsen.com/machine-ui/pkg/sandbox/api"
	"kormsen.com/machine-ui/pkg/sandbox/discovery"
	"kormsen.com/machine-ui/pkg/sandbox/renderer"
	// "kormsen.com/machine-ui/pkg/models" - Models are used by discovery and api packages, not directly in main
)

const (
	componentsDir = "static/components"     // Relative to project root
	templateDir   = "cmd/sandbox/templates" // Relative to project root
	staticDir     = "static"                // Relative to project root
	listenAddr    = ":8080"
)

func main() {
	// Discover stories
	discoveredComponents, err := discovery.DiscoverStories(componentsDir)
	if err != nil {
		log.Printf("Warning: Error discovering stories from %s: %v", componentsDir, err)
	}
	if len(discoveredComponents) == 0 {
		log.Println("No component stories were found. The sidebar will be empty.")
	}

	// Define a slice of strings for the template directories
	templateDirs := []string{"cmd/sandbox/templates", "static"}

	// Pass the slice to LoadTemplates
	templateSet, err := renderer.LoadTemplates(templateDirs)
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	// Create the main router
	// Note: api.NewRouter expects []models.ComponentGroup, which discovery.DiscoverStories returns.
	// The models package is imported by the api and discovery packages themselves.
	mainRouter := api.NewRouter(staticDir, templateSet, discoveredComponents)

	// Start the HTTP server
	log.Printf("Sandbox application starting. Listening on http://localhost%s ...", listenAddr)
	err = http.ListenAndServe(listenAddr, mainRouter)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
