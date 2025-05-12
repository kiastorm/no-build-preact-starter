package renderer

import (
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// LoadTemplates parses all HTML templates from the given base directories.
// It walks each directory tree and parses all files ending with .gohtml or .html.
// It includes custom functions like "safeJS", "dict", and "html" in the template FuncMap.
func LoadTemplates(templateBaseDirs []string) (*template.Template, error) {
	funcMap := template.FuncMap{
		"safeJS": func(s string) template.JS {
			return template.JS(s)
		},
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("dict: invalid number of arguments")
			}
			d := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict: key must be a string")
				}
				d[key] = values[i+1]
			}
			return d, nil
		},
		"html": func(text string) template.HTML {
			return template.HTML(text)
		},
		"defaultVal": func(value interface{}, defaultValue interface{}) interface{} {
			// Check if the value is its zero value, specifically for strings if it's empty.
			// This can be expanded for other types if needed.
			if vStr, ok := value.(string); ok && vStr == "" {
				return defaultValue
			}
			// Add more sophisticated zero-value checks for other types if necessary
			if value == nil { // A general check for nil, though type assertion is safer
				return defaultValue
			}
			// If value is not empty or nil, return the original value
			return value
		},
		"ToUpper": strings.ToUpper,
	}

	tmpl := template.New("").Funcs(funcMap)
	templatesToParse := []string{}

	for _, baseDir := range templateBaseDirs {
		err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && (strings.HasSuffix(info.Name(), ".gohtml") || strings.HasSuffix(info.Name(), ".html")) {
				templatesToParse = append(templatesToParse, path)
				log.Printf("Found template for parsing: %s", path) // More verbose logging
			}
			return nil
		})
		if err != nil {
			log.Printf("Error walking template directory %s: %v", baseDir, err)
			// Decide if we should return error or continue with other dirs
			// For now, let's collect all errors and return at the end, or just the first one.
			return nil, err // Return on first walk error for simplicity
		}
	}

	if len(templatesToParse) == 0 {
		log.Println("No .html or .gohtml templates found in any specified directories.")
		return tmpl, nil // Return an empty (but valid) template set
	}

	// Parse all collected template files.
	tmpl, err := tmpl.ParseFiles(templatesToParse...)
	if err != nil {
		return nil, err
	}

	log.Printf("Successfully parsed %d template files from all specified directories.", len(templatesToParse))
	// for _, t := range tmpl.Templates() { log.Printf(" - Defined template: %s", t.Name()) } // Debug: list all defined templates

	return tmpl, nil
}

// Execute renders the specified template with the given data to the http.ResponseWriter.
func Execute(w http.ResponseWriter, tmpl *template.Template, name string, data interface{}) (executeErr error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC during template execution '%s': %v", name, r)
			// Set error state so caller knows something went wrong
			executeErr = fmt.Errorf("panic executing template %s: %v", name, r)
			// Try to send an error response if headers aren't written
			if _, ok := w.(http.Flusher); ok { // Check if headers likely written
				// Cannot send http.Error reliably
			} else {
				// Attempt to send internal server error
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}
	}()

	if tmpl == nil {
		log.Println("Error executing template: template set is nil")
		http.Error(w, "Internal Server Error: Templates not loaded", http.StatusInternalServerError)
		return errors.New("template set is nil")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute the template - potential panic source
	executeErr = tmpl.ExecuteTemplate(w, name, data)

	if executeErr != nil {
		log.Printf("Error executing template %s: %v", name, executeErr)
		// Don't try to write http.Error here if template execution already started writing
		return executeErr // Return the error
	}

	return nil // Success
}
