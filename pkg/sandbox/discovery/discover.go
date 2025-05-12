package discovery

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"kormsen.com/machine-ui/pkg/sandbox/models"
)

// Regular expressions to parse story files (unexported)
var (
	storyKeyRegex           = regexp.MustCompile(`export\s+const\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*{`)
	storyTitleRegex         = regexp.MustCompile(`title:\s*["']([^"']+)["']`)
	defaultTitleRegex       = regexp.MustCompile(`export\s+default\s*{\s*title:\s*["']([^"']+)["']`)
	storyArgsBlockRegex     = regexp.MustCompile(`(?s)args:\s*{([^{}]*)}`)
	argKeyStringValueRegex  = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*):\s*\"((?:\\\"|[^"])*)\"`)
	argKeyBooleanValueRegex = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*):\s*(true|false)`)
	argKeyNumericValueRegex = regexp.MustCompile(`([A-Za-z_][A-Za-z0-9_]*):\s*([0-9]+(?:\.[0-9]+)?)`)
	argKeyHTMLValueRegex    = regexp.MustCompile("children:\\s*html`((?:\\\\`|[^`])*)`")
	storyIDRegex            = regexp.MustCompile(`id:\s*["']([^"']+)["']`)
	pendingTextRegex        = regexp.MustCompile(`pendingText:\s*["']([^"']+)["']`)
)

// ArgType represents the data type of an argument
type ArgType string

const (
	ArgTypeString  ArgType = "string"
	ArgTypeBoolean ArgType = "boolean"
	ArgTypeNumber  ArgType = "number"
	ArgTypeHTML    ArgType = "html"
)

// ArgTypeInfo stores information about an argument's type and optional metadata
type ArgTypeInfo struct {
	Type     ArgType
	Required bool
	Control  string   // Optional UI control type (select, radio, etc.)
	Options  []string // For select/radio controls
	Min      *float64 // For number type
	Max      *float64 // For number type
	Default  *string  // Default value as string
}

// parseArgs extracts key-value pairs from a story's args block string.
// This is a simplified parser and might not cover all JS object literal syntax.
func parseArgs(argsBlock string) (map[string]interface{}, map[string]models.ArgTypeInfo) {
	args := make(map[string]interface{})
	argTypes := make(map[string]models.ArgTypeInfo)

	// Handle children: html`...` separately first because its value can contain colons/commas
	htmlMatch := argKeyHTMLValueRegex.FindStringSubmatch(argsBlock)
	if len(htmlMatch) > 1 {
		// Replace backticks and escaped backticks correctly for Go string
		childrenContent := strings.ReplaceAll(htmlMatch[1], "\\`", "`")
		args["Children"] = childrenContent // Store as string, Go template will use template.HTML
		argTypes["Children"] = models.ArgTypeInfo{Type: models.ArgTypeHTML, Required: false}
		// Remove the html`...` part to avoid reprocessing by simpler regexes
		argsBlock = strings.Replace(argsBlock, htmlMatch[0], "", 1)
	}

	// String values: key: "value"
	matchesStr := argKeyStringValueRegex.FindAllStringSubmatch(argsBlock, -1)
	for _, match := range matchesStr {
		if len(match) == 3 {
			value := strings.ReplaceAll(match[2], "\\\"", "\"")
			args[match[1]] = value
			// Store default value
			defaultValStr := value
			argTypes[match[1]] = models.ArgTypeInfo{Type: models.ArgTypeString, Required: false, Default: &defaultValStr}
		}
	}

	// Boolean values: key: true or key: false
	matchesBool := argKeyBooleanValueRegex.FindAllStringSubmatch(argsBlock, -1)
	for _, match := range matchesBool {
		if len(match) == 3 {
			boolValue := (match[2] == "true")
			args[match[1]] = boolValue
			// Store default value
			defaultValStr := match[2]
			argTypes[match[1]] = models.ArgTypeInfo{Type: models.ArgTypeBoolean, Required: false, Default: &defaultValStr}
		}
	}

	// Numeric values: key: 123 or key: 12.34 (currently captures as string, could parse to float/int)
	matchesNum := argKeyNumericValueRegex.FindAllStringSubmatch(argsBlock, -1)
	for _, match := range matchesNum {
		if len(match) == 3 {
			args[match[1]] = match[2] // Store as string, template can handle
			// Store default value
			defaultValStr := match[2]
			argTypes[match[1]] = models.ArgTypeInfo{Type: models.ArgTypeNumber, Required: false, Default: &defaultValStr}
		}
	}

	return args, argTypes
}

// DiscoverStories scans the specified directory for component story files (*.stories.js)
// and parses them to extract component and story variant information.
func DiscoverStories(componentsDir string) ([]models.ComponentGroup, error) {
	log.Printf("Discovering stories from directory: %s", componentsDir)
	var discoveredComponents []models.ComponentGroup
	staticDirRoot := "static"

	err := filepath.Walk(componentsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Printf("Error accessing path %s: %v", path, err)
			return err
		}
		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".stories.js") {
			log.Printf("Found .stories.js file: %s", path)
			componentNameFromFile := strings.TrimSuffix(info.Name(), ".stories.js")

			jsStoryPath, relErr := filepath.Rel(staticDirRoot, path)
			if relErr != nil {
				log.Printf("Could not make .stories.js path relative for %s (staticDirRoot: %s): %v", path, staticDirRoot, relErr)
				jsStoryPath = filepath.ToSlash(path)
			} else {
				jsStoryPath = filepath.ToSlash(jsStoryPath)
			}

			gohtmlStoriesPath := strings.Replace(jsStoryPath, ".stories.js", ".stories.gohtml", 1)
			componentGoHTMLPath := strings.Replace(jsStoryPath, ".stories.js", ".gohtml", 1)

			// Determine if this component can be SSR'd based on existence of Go template files
			// Basic check: are the paths non-empty? A more robust check would be os.Stat in main or handler.
			componentCanSSR := false
			// To truly check, we need the full path. Assuming staticDirRoot is cwd or accessible for Stat.
			if _, err := os.Stat(filepath.Join(staticDirRoot, gohtmlStoriesPath)); err == nil { // Check if .stories.gohtml exists
				if _, err := os.Stat(filepath.Join(staticDirRoot, componentGoHTMLPath)); err == nil { // Check if component .gohtml exists
					componentCanSSR = true
					log.Printf("    Component %s determined to support SSR (found %s and %s)", componentNameFromFile, gohtmlStoriesPath, componentGoHTMLPath)
				} else {
					log.Printf("    Component %s SSR check: %s found, but component GoHTML %s not found (%v)", componentNameFromFile, gohtmlStoriesPath, componentGoHTMLPath, err)
				}
			} else {
				log.Printf("    Component %s SSR check: stories GoHTML %s not found (%v)", componentNameFromFile, gohtmlStoriesPath, err)
			}

			contentBytes, readErr := os.ReadFile(filepath.Join(staticDirRoot, jsStoryPath)) // Ensure reading from correct base
			if readErr != nil {
				log.Printf("Error reading story file %s: %v", jsStoryPath, readErr)
				return readErr
			}
			content := string(contentBytes)

			var variants []models.StoryVariant
			storyMatches := storyKeyRegex.FindAllStringSubmatch(content, -1)
			if len(storyMatches) == 0 {
				log.Printf("No story keys (export const XXX) found in %s", path)
			}

			for _, storyMatch := range storyMatches {
				if len(storyMatch) > 1 {
					storyKey := storyMatch[1]
					log.Printf("  Found story key: %s in %s", storyKey, path)
					variantBlockRegex := regexp.MustCompile(`export\s+const\s+` + regexp.QuoteMeta(storyKey) + `\s*=\s*{([^}]*)}`)
					variantBlockMatch := variantBlockRegex.FindStringSubmatch(content)
					variantTitle := storyKey
					storyArgs := make(map[string]interface{})
					storyArgTypes := make(map[string]models.ArgTypeInfo)

					if len(variantBlockMatch) > 1 {
						blockContent := variantBlockMatch[1]
						titleMatch := storyTitleRegex.FindStringSubmatch(blockContent)
						if len(titleMatch) > 1 {
							variantTitle = titleMatch[1]
						}

						argsBlockMatch := storyArgsBlockRegex.FindStringSubmatch(blockContent)
						if len(argsBlockMatch) > 1 {
							storyArgs, storyArgTypes = parseArgs(argsBlockMatch[1])
						} else {
							log.Printf("    No args block found for story %s in %s", storyKey, path)
						}
					} else {
						log.Printf("    Could not find variant block for story %s in %s", storyKey, path)
					}

					if _, ok := storyArgs["IsSquare"]; !ok {
						isSquareVal := strings.Contains(storyKey, "Icon")
						storyArgs["IsSquare"] = isSquareVal
						defaultValStr := fmt.Sprintf("%v", isSquareVal)
						storyArgTypes["IsSquare"] = models.ArgTypeInfo{Type: models.ArgTypeBoolean, Required: false, Default: &defaultValStr}
					}

					// Check for PendingText in the story
					hasPendingText := false
					pendingTextMatch := pendingTextRegex.FindStringSubmatch(content) // Check against the full story content, not just the block
					if len(pendingTextMatch) > 1 {
						hasPendingText = true
						if _, ok := storyArgs["PendingText"]; !ok {
							pendingTextVal := pendingTextMatch[1]
							storyArgs["PendingText"] = pendingTextVal
							defaultValStr := pendingTextVal
							storyArgTypes["PendingText"] = models.ArgTypeInfo{Type: models.ArgTypeString, Required: false, Default: &defaultValStr}
						}
					}

					// Special handling for HTML Children when using template.HTML
					if childrenVal, ok := storyArgs["Children"]; ok {
						if childrenStr, isString := childrenVal.(string); isString {
							storyArgs["Children"] = template.HTML(childrenStr)
						}
					}

					variants = append(variants, models.StoryVariant{
						Key:            storyKey,
						Title:          variantTitle,
						Args:           storyArgs,
						ArgTypes:       storyArgTypes,
						HasCSR:         true,            // Variants from .stories.js are always CSR capable
						HasSSR:         componentCanSSR, // SSR capability depends on Go templates for the component
						HasPendingText: hasPendingText,
					})
				}
			}

			componentTitle := strings.Title(strings.ReplaceAll(componentNameFromFile, "-", " "))
			defaultTitleMatch := defaultTitleRegex.FindStringSubmatch(content)
			if len(defaultTitleMatch) > 1 {
				componentTitle = defaultTitleMatch[1]
			}

			if len(variants) > 0 {
				discoveredComponents = append(discoveredComponents, models.ComponentGroup{
					Name:                componentNameFromFile,
					Title:               componentTitle,
					Path:                jsStoryPath,
					StoryContent:        content,
					Variants:            variants,
					SSRGoHTMLPath:       gohtmlStoriesPath,
					ComponentGoHTMLPath: componentGoHTMLPath,
					CanSSR:              componentCanSSR,
				})
				log.Printf("Successfully discovered component: %s (%s) with %d variants. CanSSR: %t", componentTitle, componentNameFromFile, len(variants), componentCanSSR)
				for _, v := range variants {
					log.Printf("  Variant: %s, Title: %s, Args: %+v", v.Key, v.Title, v.Args)
					for argName, argType := range v.ArgTypes {
						log.Printf("    Arg: %s, Type: %s, Default: %v", argName, argType.Type, argType.Default)
					}
				}
			} else {
				log.Printf("No variants found for component %s in file %s, component not added.", componentNameFromFile, path)
			}
		} else {
			// log.Printf("Skipping non .stories.js file: %s", info.Name()) // Optional: very verbose
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking components directory %s: %w", componentsDir, err)
	}
	if len(discoveredComponents) == 0 {
		log.Println("WARNING: No component stories were discovered after walking the components directory.")
	}

	// Now enrich the GoHTML templates with the arg type information from JS
	err = enrichGoHTMLTemplates(discoveredComponents, staticDirRoot)
	if err != nil {
		log.Printf("WARNING: Error enriching GoHTML templates: %v", err)
	}

	return discoveredComponents, nil
}

// enrichGoHTMLTemplates reads the .stories.gohtml files and adds type annotations
// using HTML comments that can be parsed by the template engine
func enrichGoHTMLTemplates(components []models.ComponentGroup, staticDirRoot string) error {
	for _, component := range components {
		if !component.CanSSR {
			continue
		}

		// Read the GoHTML stories file
		goHTMLPath := filepath.Join(staticDirRoot, component.SSRGoHTMLPath)
		if _, err := os.Stat(goHTMLPath); err != nil {
			log.Printf("Skipping GoHTML enrichment for %s: file not found", goHTMLPath)
			continue
		}

		// Check if we need to inject arg type information
		// This could be implemented by adding HTML comments with type information
		// or by generating a JSON file that maps story keys to arg types

		// For now, we'll just log that we would process the file
		log.Printf("Would enrich GoHTML template for %s with arg type information", component.Name)
	}

	return nil
}
