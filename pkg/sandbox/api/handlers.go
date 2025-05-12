package api

import (
	"bytes"         // Added for capturing template output to a buffer
	"encoding/json" // Added for marshalling sandbox config

	// Added for errors.New
	"fmt"
	"slices"

	// Added for Sprintf
	"html/template"
	"log"
	"net/http"
	"net/url" // Added for URL manipulation

	// Added for parsing numbers from query
	"strconv"
	"strings"

	"kormsen.com/machine-ui/pkg/sandbox/models" // Updated path
	// Updated path
	// Added for slices.Contains
)

// AppHandlers holds dependencies for HTTP handlers, such as the list of discovered components
// and the parsed HTML templates.
// It's a good practice to pass dependencies to handlers explicitly rather than using globals.
type AppHandlers struct {
	Components []models.ComponentGroup // A cache of discovered components
	Templates  *template.Template
}

// Home renders the home page of the component playground.
// It lists all available components.
func (h *AppHandlers) Home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Create a fresh copy of components for this request to avoid modifying the shared slice
	pageComponents := make([]models.ComponentGroup, len(h.Components))
	copy(pageComponents, h.Components)
	// Reset IsSelected flags for all components and their variants for the homepage
	for i := range pageComponents {
		pageComponents[i].IsSelected = false
		for j := range pageComponents[i].Variants {
			pageComponents[i].Variants[j].IsSelected = false
		}
	}

	currentTheme := "light"
	if r.URL.Query().Get("theme") == "dark" {
		currentTheme = "dark"
	}

	// Construct ToggleThemeURL
	toggleThemeQueryForHome := r.URL.Query()
	newThemeForHomeToggle := "dark"
	if currentTheme == "light" {
		newThemeForHomeToggle = "dark"
	} else {
		newThemeForHomeToggle = "light"
	}
	toggleThemeQueryForHome.Set("theme", newThemeForHomeToggle)
	// Ensure other relevant params for home are preserved if any (e.g., if home ever gets params)
	// Path for home page theme toggle should hit the body-swap handler's home context.
	toggleThemeURLValue := (&url.URL{Path: "/sandbox-body-swap/", RawQuery: toggleThemeQueryForHome.Encode()}).String()

	// Construct ResetArgsURL for Home
	resetArgsQueryHome := r.URL.Query()
	// No specific args on home, but keep structure. It should also point to body-swap home.
	resetArgsURLValue := (&url.URL{Path: "/sandbox-body-swap/", RawQuery: resetArgsQueryHome.Encode()}).String()

	data := models.PageData{
		Title:                 "Component Playground",
		Components:            pageComponents,
		StaticBaseURL:         "/static",
		Theme:                 currentTheme,
		ToggleThemeURL:        toggleThemeURLValue,
		ResetArgsURL:          resetArgsURLValue,
		CurrentPath:           r.URL.Path, // Current path is "/"
		CanClientSideNavigate: true,       // Assuming JS is available for mach-link
		RenderMode:            "csr",      // Home doesn't have specific modes, toolbar needs a default
		IframeSrcURL:          "/sandbox-content?renderMode=csr&componentName=fallback",
	}

	// Pre-render ToolbarHTML and StoryArgsEditorHTML for the body template
	var toolbarBuf bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&toolbarBuf, "toolbar-content", data); err != nil {
		log.Printf("Home: Error rendering toolbar template: %v", err)
		// Potentially set to an error message or empty
	}
	data.ToolbarHTML = template.HTML(toolbarBuf.String())

	var argsEditorBuf bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&argsEditorBuf, "story-args-editor-content", data); err != nil {
		log.Printf("Home: Error rendering story args editor template: %v", err)
	}
	data.StoryArgsEditorHTML = template.HTML(argsEditorBuf.String())

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte("<!DOCTYPE html>\n<html lang=\"en\">")); err != nil {
		log.Printf("Home: Error writing initial HTML: %v", err)
		return
	}
	if err := h.Templates.ExecuteTemplate(w, "_document_head", data); err != nil {
		log.Printf("Home: Error executing _document_head template: %v", err)
		_, _ = w.Write([]byte("</html>")) // Best effort
		return
	}
	if err := h.Templates.ExecuteTemplate(w, "full-body-content", data); err != nil {
		log.Printf("Home: Error executing full-body-content template: %v", err)
		_, _ = w.Write([]byte("</html>")) // Best effort
		return
	}
	if _, err := w.Write([]byte("</html>")); err != nil {
		log.Printf("Home: Error writing closing HTML tag: %v", err)
	}
}

// ViewStory renders the page for a specific component story.
// It now always uses an iframe for content, whose src is determined by renderMode.
func (h *AppHandlers) ViewStory(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[0] != "sandbox" {
		http.NotFound(w, r)
		return
	}

	componentNameParam := pathParts[1]
	storyKeyParam := ""
	if len(pathParts) > 2 {
		storyKeyParam = pathParts[2]
	}

	var currentComponent *models.ComponentGroup
	var currentComponentIdx = -1
	var selectedStoryVariant *models.StoryVariant

	pageComponents := make([]models.ComponentGroup, len(h.Components))
	for i, comp := range h.Components {
		pageComponents[i] = comp
		pageComponents[i].Variants = make([]models.StoryVariant, len(comp.Variants))
		for j, variant := range comp.Variants {
			pageComponents[i].Variants[j] = variant
			if variant.Args != nil {
				newArgs := make(map[string]interface{})
				for k, v_arg := range variant.Args {
					newArgs[k] = v_arg
				}
				pageComponents[i].Variants[j].Args = newArgs
			}
		}
		if comp.Name == componentNameParam {
			currentComponent = &pageComponents[i]
			currentComponentIdx = i
		}
	}

	if currentComponent == nil {
		log.Printf("Component group not found: %s", componentNameParam)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	for i := range pageComponents {
		pageComponents[i].IsSelected = (i == currentComponentIdx)
		if i != currentComponentIdx {
			for j := range pageComponents[i].Variants {
				pageComponents[i].Variants[j].IsSelected = false
			}
		}
	}

	isFallbackScenario := false
	if storyKeyParam == "" && len(currentComponent.Variants) > 0 {
		storyKeyParam = currentComponent.Variants[0].Key
	} else if storyKeyParam == "" && len(currentComponent.Variants) == 0 {
		isFallbackScenario = true
	}

	validStoryKey := false
	if !isFallbackScenario && storyKeyParam != "" {
		for i, variant := range currentComponent.Variants {
			if variant.Key == storyKeyParam {
				currentComponent.Variants[i].IsSelected = true
				selectedStoryVariant = &currentComponent.Variants[i]
				validStoryKey = true
				break
			}
			currentComponent.Variants[i].IsSelected = false
		}
		if !validStoryKey {
			log.Printf("Story variant '%s' not found in component '%s'.", storyKeyParam, currentComponent.Title)
			if len(currentComponent.Variants) > 0 {
				firstStoryKey := currentComponent.Variants[0].Key
				redirectURL := "/sandbox/" + componentNameParam + "/" + firstStoryKey
				existingQuery := r.URL.Query()
				if len(existingQuery) > 0 {
					redirectURL += "?" + existingQuery.Encode()
				}
				http.Redirect(w, r, redirectURL, http.StatusFound)
				return
			}
			isFallbackScenario = true
		}
	}

	pageTitle := currentComponent.Title
	if selectedStoryVariant != nil {
		pageTitle += " - " + selectedStoryVariant.Title
	} else if isFallbackScenario {
		pageTitle += " - Info"
	}

	var availableModes []string
	if selectedStoryVariant != nil && selectedStoryVariant.HasCSR {
		availableModes = append(availableModes, "csr")
	}
	if selectedStoryVariant != nil && selectedStoryVariant.HasSSR && h.Templates.Lookup(selectedStoryVariant.Key) != nil {
		availableModes = append(availableModes, "ssr")
	} else if selectedStoryVariant != nil && selectedStoryVariant.HasSSR {
		log.Printf("Warning: Story %s/%s marked HasSSR, but Go template '%s' not found.", currentComponent.Name, selectedStoryVariant.Key, selectedStoryVariant.Key)
	}
	if isFallbackScenario && currentComponent.Path != "" {
		availableModes = []string{"csr"}
	}
	log.Printf("ViewStory: Initial availableModes: %v for %s/%s", availableModes, componentNameParam, storyKeyParam)

	// Comprehensive logic block for parameter determination and redirects:
	isMachRequest := r.Header.Get("X-Mach-Request") == "true"
	initialRequestedRenderMode := r.URL.Query().Get("renderMode")
	initialRequestedTheme := r.URL.Query().Get("theme")

	var effectiveRenderMode string
	var effectiveTheme string

	// --- Determine effectiveRenderMode based on request type and parameters ---
	ssrAvailable := slices.Contains(availableModes, "ssr") // Check actual SSR availability
	csrAvailable := slices.Contains(availableModes, "csr")
	isRequestedModeValid := false // Flag to track if the requested mode was valid

	if initialRequestedRenderMode != "" {
		for _, mode := range availableModes {
			if mode == initialRequestedRenderMode {
				isRequestedModeValid = true
				break
			}
		}
	}

	if isRequestedModeValid {
		// Requested mode is valid and available, use it regardless of Mach request status.
		effectiveRenderMode = initialRequestedRenderMode
		log.Printf("ViewStory: Honoring valid requested renderMode from query: '%s'.", effectiveRenderMode)
	} else {
		// Requested mode is missing, empty, or invalid for this story. Apply defaults based on request type.
		log.Printf("ViewStory: Requested renderMode ('%s') is missing or invalid for available modes %v. Determining default...", initialRequestedRenderMode, availableModes)
		if !isMachRequest { // Default for non-JS requests: Prioritize SSR
			effectiveRenderMode = "ssr" // Default to attempting SSR
			log.Printf("ViewStory: Defaulting non-Mach request to 'ssr'. Actual SSR availability: %t", ssrAvailable)
		} else { // Default for JS requests: Prioritize CSR
			if csrAvailable {
				effectiveRenderMode = "csr"
				log.Printf("ViewStory: Defaulting Mach request to 'csr' (available).")
			} else if ssrAvailable {
				effectiveRenderMode = "ssr"
				log.Printf("ViewStory: Defaulting Mach request to 'ssr' (CSR not available).")
			} else if len(availableModes) > 0 {
				effectiveRenderMode = availableModes[0]
				log.Printf("ViewStory: Defaulting Mach request to first available mode: '%s'.", availableModes[0])
			} else {
				effectiveRenderMode = "csr" // Ultimate fallback
				log.Printf("ViewStory: Defaulting Mach request to 'csr' (no modes detected).")
			}
		}
	}

	// Fallback scenario (no specific story selected) always forces CSR iframe content.
	if isFallbackScenario {
		if effectiveRenderMode != "csr" {
			log.Printf("ViewStory: Fallback scenario. Overriding effectiveRenderMode to 'csr' (was '%s')", effectiveRenderMode)
			effectiveRenderMode = "csr"
			// ssrAvailable = false // Uncomment if fallback should never be considered SSR available
		}
	}

	// Determine effective theme, defaulting to "light".
	effectiveTheme = initialRequestedTheme
	if effectiveTheme != "light" && effectiveTheme != "dark" {
		effectiveTheme = "light" // Default theme
	}

	// For non-Mach requests, redirect ONLY if parameters were defaulted (missing/invalid)
	// or if theme needed defaulting. If user explicitly provided valid params, don't redirect.
	if !isMachRequest {
		needsRedirect := false
		// Redirect if mode was defaulted (because requested was invalid/missing)
		// Note: Also check if the defaulted effectiveRenderMode is different from what was initially requested (even if invalid).
		if !isRequestedModeValid { // Mode was invalid or missing, default was applied
			if initialRequestedRenderMode != effectiveRenderMode { // Only redirect if the default is different from what was asked for (or absent)
				needsRedirect = true
				log.Printf("ViewStory: Redirect needed because renderMode was defaulted (requested '%s', effective '%s'. Default applied because request invalid/missing).", initialRequestedRenderMode, effectiveRenderMode)
			}
		}

		// Redirect if theme was defaulted (and wasn't just missing but defaulting to light)
		if initialRequestedTheme != effectiveTheme && !(initialRequestedTheme == "" && effectiveTheme == "light") {
			needsRedirect = true
			log.Printf("ViewStory: Redirect needed because theme was defaulted (requested '%s', effective '%s').", initialRequestedTheme, effectiveTheme)
		}

		if needsRedirect {
			canonicalQuery := r.URL.Query()
			canonicalQuery.Set("renderMode", effectiveRenderMode)
			canonicalQuery.Set("theme", effectiveTheme)
			redirectURL := url.URL{Path: r.URL.Path, RawQuery: canonicalQuery.Encode()}
			log.Printf("ViewStory: Redirecting non-Mach request to canonical URL: '%s'", redirectURL.String())
			http.Redirect(w, r, redirectURL.String(), http.StatusFound)
			return
		}
	}
	log.Printf("ViewStory: Final decision for request URL '%s': isMachRequest=%t, effectiveRenderMode='%s', effectiveTheme='%s', ssrAvailable=%t", r.URL.String(), isMachRequest, effectiveRenderMode, effectiveTheme, ssrAvailable)

	data := models.PageData{
		Title:                 pageTitle,
		Components:            pageComponents,
		SelectedComponent:     currentComponent,
		SelectedStoryKey:      storyKeyParam,
		StaticBaseURL:         "/static",
		RenderMode:            effectiveRenderMode,
		AvailableRenderModes:  availableModes,
		SSRAvailable:          ssrAvailable,
		Theme:                 effectiveTheme,
		CurrentPath:           r.URL.Path,
		CanClientSideNavigate: isMachRequest,
	}

	currentStoryArgs := make(map[string]interface{})
	// SelectedStoryArgs processing (from ViewStory logic, adapted)
	if selectedStoryVariant != nil && selectedStoryVariant.Args != nil {
		requestQueryParams := r.URL.Query()
		for argName, defaultValue := range selectedStoryVariant.Args {
			currentVal := defaultValue
			isDefaultBool := false
			if _, ok := defaultValue.(bool); ok {
				isDefaultBool = true
			}
			if queryVals, ok := requestQueryParams[argName]; ok && len(queryVals) > 0 {
				paramStrValue := queryVals[0]
				if isDefaultBool {
					currentVal = (strings.ToLower(paramStrValue) == "true")
				} else {
					switch defaultValue.(type) {
					case int, int64, float32, float64:
						if num, err := strconv.ParseFloat(paramStrValue, 64); err == nil {
							currentVal = num
						} else {
							currentVal = paramStrValue
						}
					default:
						currentVal = paramStrValue
					}
				}
			} else {
				if isDefaultBool {
					currentVal = false
				}
			}
			currentStoryArgs[argName] = currentVal
		}
		if defaultChildrenVal, ok := selectedStoryVariant.Args["Children"]; ok {
			if _, defaultIsHTML := defaultChildrenVal.(template.HTML); defaultIsHTML {
				if currentChildrenVal, currentOk := currentStoryArgs["Children"]; currentOk {
					if childrenStr, valIsString := currentChildrenVal.(string); valIsString {
						currentStoryArgs["Children"] = template.HTML(childrenStr)
					}
				}
			}
		}
		data.SelectedStoryArgs = currentStoryArgs
	}

	// --- Construct IframeSrcURL with dynamic path ---
	iframePath := "/sandbox-content/" + componentNameParam
	if storyKeyParam != "" {
		iframePath += "/" + storyKeyParam
	}

	iframeQuery := url.Values{}
	iframeQuery.Set("renderMode", data.RenderMode)
	iframeQuery.Set("theme", data.Theme)

	// Parse args for PageData (for editor) AND for iframe query
	if selectedStoryVariant != nil && selectedStoryVariant.Args != nil {
		requestQueryParams := r.URL.Query()
		for argName, defaultValue := range selectedStoryVariant.Args {
			currentVal := defaultValue
			stringValForQuery := fmt.Sprintf("%v", defaultValue)
			isDefaultBool := false
			if _, ok := defaultValue.(bool); ok {
				isDefaultBool = true
			}

			if queryVals, ok := requestQueryParams[argName]; ok && len(queryVals) > 0 {
				paramStrValue := queryVals[0]
				stringValForQuery = paramStrValue
				if isDefaultBool {
					currentVal = (strings.ToLower(paramStrValue) == "true")
				} else {
					switch defaultValue.(type) {
					case int, int64, float32, float64:
						if num, err := strconv.ParseFloat(paramStrValue, 64); err == nil {
							currentVal = num
						} else {
							currentVal = paramStrValue
						}
					default:
						currentVal = paramStrValue
					}
				}
			} else {
				if isDefaultBool {
					currentVal = false
					stringValForQuery = "false"
				}
			}
			currentStoryArgs[argName] = currentVal
			iframeQuery.Set(argName, stringValForQuery)
		}
		if defaultChildrenVal, ok := selectedStoryVariant.Args["Children"]; ok {
			if _, defaultIsHTML := defaultChildrenVal.(template.HTML); defaultIsHTML {
				if currentChildrenVal, currentOk := currentStoryArgs["Children"]; currentOk {
					if childrenStr, valIsString := currentChildrenVal.(string); valIsString {
						currentStoryArgs["Children"] = template.HTML(childrenStr)
					}
				}
			}
		}
		data.SelectedStoryArgs = currentStoryArgs
	} else {
		requestQueryParams := r.URL.Query()
		for key, vals := range requestQueryParams {
			if len(vals) > 0 && key != "renderMode" && key != "theme" && key != "componentName" && key != "storyKey" {
				iframeQuery.Set(key, vals[0])
			}
		}
	}
	data.IframeSrcURL = iframePath + "?" + iframeQuery.Encode() // Assign the final URL

	vsToggleThemeQuery := r.URL.Query()
	newThemeForStoryToggle := "dark"
	if data.Theme == "light" {
		newThemeForStoryToggle = "dark"
	} else {
		newThemeForStoryToggle = "light"
	}
	vsToggleThemeQuery.Set("theme", newThemeForStoryToggle)

	// Path for story view theme toggle should hit the body-swap handler with component/story context.
	bodySwapPathForStory := "/sandbox-body-swap/" + componentNameParam
	if storyKeyParam != "" {
		bodySwapPathForStory += "/" + storyKeyParam
	}
	data.ToggleThemeURL = (&url.URL{Path: bodySwapPathForStory, RawQuery: vsToggleThemeQuery.Encode()}).String()

	resetArgsQueryStory := r.URL.Query()
	if selectedStoryVariant != nil && selectedStoryVariant.Args != nil {
		for argName := range selectedStoryVariant.Args {
			resetArgsQueryStory.Del(argName)
		}
	}
	data.ResetArgsURL = (&url.URL{Path: bodySwapPathForStory, RawQuery: resetArgsQueryStory.Encode()}).String()

	var modeLinks []models.ModeSwitchLink
	for _, mode := range data.AvailableRenderModes {
		queryForModeSwitch := r.URL.Query()
		queryForModeSwitch.Set("renderMode", mode)
		modeURL := (&url.URL{Path: bodySwapPathForStory, RawQuery: queryForModeSwitch.Encode()}).String()
		isActive := (mode == data.RenderMode)
		var text string
		if isActive {
			text = strings.ToUpper(mode) + " Active"
		} else {
			text = "Switch to " + strings.ToUpper(mode)
		}
		modeLinks = append(modeLinks, models.ModeSwitchLink{ModeKey: mode, URL: modeURL, IsActive: isActive, Text: text})
	}
	data.ModeSwitchLinks = modeLinks

	var toolbarBuf bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&toolbarBuf, "toolbar-content", data); err == nil {
		data.ToolbarHTML = template.HTML(toolbarBuf.String())
	} else {
		log.Printf("Error rendering toolbar template: %v", err)
		data.ToolbarHTML = template.HTML("<!-- Error rendering toolbar -->")
	}

	var argsEditorBuf bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&argsEditorBuf, "story-args-editor-content", data); err == nil {
		data.StoryArgsEditorHTML = template.HTML(argsEditorBuf.String())
	} else {
		log.Printf("Error rendering story args editor template: %v", err)
		data.StoryArgsEditorHTML = template.HTML("<!-- Error rendering story args editor -->")
	}

	data.IsPartialRequest = isMachRequest
	if isMachRequest {
		log.Printf("Partial request for %s/%s. Target: '%s', Mode: %s, IframeSrc: %s", componentNameParam, storyKeyParam, data.ClientTargetSelector, data.RenderMode, data.IframeSrcURL)
		w.Header().Set("X-Mach-Title", pageTitle)

		var mainContentBuf bytes.Buffer
		if err := h.Templates.ExecuteTemplate(&mainContentBuf, "content", data); err != nil {
			log.Printf("Error executing partial template 'content': %v", err)
			http.Error(w, "Failed to render partial content", http.StatusInternalServerError)
			return
		}

		var navContentBuf bytes.Buffer
		if err := h.Templates.ExecuteTemplate(&navContentBuf, "navigation-content", data); err != nil {
			log.Printf("Error executing partial template 'navigation-content': %v", err)
			http.Error(w, "Failed to render partial navigation", http.StatusInternalServerError)
			return
		}
		var updatedToolbarBuf, updatedArgsEditorBuf bytes.Buffer
		_ = h.Templates.ExecuteTemplate(&updatedToolbarBuf, "toolbar-content", data)
		_ = h.Templates.ExecuteTemplate(&updatedArgsEditorBuf, "story-args-editor-content", data)

		responseBody := "<div data-mach-target-selector=\"#component-nav\">" + navContentBuf.String() + "</div>" +
			"<div data-mach-target-selector=\".sandbox-toolbar\">" + updatedToolbarBuf.String() + "</div>" +
			"<div data-mach-target-selector=\".story-args-editor\">" + updatedArgsEditorBuf.String() + "</div>" +
			mainContentBuf.String()

		w.Write([]byte(responseBody))
		return
	}

	// Fallthrough for non-Mach (full page) requests.
	// Render the full HTML structure using _document_head and full-body-content.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte("<!DOCTYPE html>\n<html lang=\"en\">")); err != nil {
		log.Printf("ViewStory: Error writing initial HTML: %v", err)
		// If we can't even write the initial HTML, sending an HTTP error might be futile or partial.
		return
	}
	if err := h.Templates.ExecuteTemplate(w, "_document_head", data); err != nil {
		log.Printf("ViewStory: Error executing _document_head template: %v", err)
		// Attempt to close HTML tag gracefully if head failed.
		_, _ = w.Write([]byte("</html>"))
		return
	}
	// Note: data.ToolbarHTML and data.StoryArgsEditorHTML were pre-rendered earlier in this function.
	// The 'full-body-content' template will use these.
	if err := h.Templates.ExecuteTemplate(w, "full-body-content", data); err != nil {
		log.Printf("ViewStory: Error executing full-body-content template: %v", err)
		// Attempt to close HTML tag gracefully if body failed.
		_, _ = w.Write([]byte("</html>"))
		return
	}
	if _, err := w.Write([]byte("</html>")); err != nil {
		log.Printf("ViewStory: Error writing closing HTML tag: %v", err)
	}
}

// ServeSandboxContent serves the content for the iframe, deciding between CSR bootstrap and SSR content.
func (h *AppHandlers) ServeSandboxContent(w http.ResponseWriter, r *http.Request) {
	// Extract component and story from path parameters
	componentName := r.PathValue("componentName")
	storyKey := r.PathValue("storyKey") // Will be empty if route was /sandbox-content/{componentName}

	query := r.URL.Query()
	renderMode := query.Get("renderMode")
	theme := query.Get("theme")
	// componentName := query.Get("componentName") // No longer from query
	// storyKey := query.Get("storyKey") // No longer from query

	if theme == "" {
		theme = "light"
	}
	if componentName == "" || componentName == "fallback" { // Handle fallback case explicitly if needed
		// If componentName is empty or 'fallback', maybe serve a specific fallback page or CSR frame?
		// For now, let's assume a valid componentName is required here.
		// The Home handler sets iframe src to ...componentName=fallback, maybe change that?
		// Let's treat 'fallback' as an indicator to serve the CSR fallback frame.
		if componentName == "fallback" {
			log.Printf("ServeSandboxContent: Handling fallback request.")
			h.serveCsrFallbackFrame(w, r, theme)
			return
		}
		http.Error(w, "Missing or invalid componentName in path", http.StatusBadRequest)
		return
	}

	// --- Handle CSR Mode ---
	if renderMode == "csr" {
		log.Printf("ServeSandboxContent: Handling CSR request for %s/%s", componentName, storyKey)

		// Prepare config for sandbox-config script tag
		config := make(map[string]interface{})
		currentArgsForCSR := make(map[string]interface{})
		var scriptToLoad string
		isFallback := false // Determine if this is a fallback scenario

		var targetComponentInDB *models.ComponentGroup
		var targetStoryVariantInDB *models.StoryVariant

		// Find component and story variant from DB to get default args and paths
		for i := range h.Components {
			if h.Components[i].Name == componentName {
				comp := h.Components[i]
				targetComponentInDB = &comp
				if storyKey != "" {
					for j := range comp.Variants {
						if comp.Variants[j].Key == storyKey {
							variant := comp.Variants[j]
							targetStoryVariantInDB = &variant
							break
						}
					}
				}
				if storyKey == "" || targetStoryVariantInDB == nil {
					isFallback = true
				}
				break
			}
		}

		if targetComponentInDB == nil {
			log.Printf("ServeSandboxContent (CSR): Component '%s' not found in DB.", componentName)
			http.Error(w, "Component not found", http.StatusNotFound)
			return
		}

		// Populate config based on whether it's a story or fallback/component view
		config["componentName"] = targetComponentInDB.Name
		config["renderMode"] = "csr"                      // Pass effective mode to client script
		if !isFallback && targetStoryVariantInDB != nil { // Specific Story
			config["storyKey"] = targetStoryVariantInDB.Key
			if targetComponentInDB.Path != "" {
				config["storyModulePath"] = "/static/" + targetComponentInDB.Path
			}
			scriptToLoad = "/static/modules/sandbox/iframe-client.js"

			if targetStoryVariantInDB.Args != nil {
				for k, v := range targetStoryVariantInDB.Args {
					currentArgsForCSR[k] = v
				}
			}
			for queryKey, queryValues := range query {
				if len(queryValues) == 0 {
					continue
				}
				if defaultValue, knownArg := currentArgsForCSR[queryKey]; knownArg {
					paramStrValue := queryValues[0]
					switch defaultValue.(type) {
					case bool:
						currentArgsForCSR[queryKey] = (strings.ToLower(paramStrValue) == "true")
					case int, int64, float32, float64:
						if num, err := strconv.ParseFloat(paramStrValue, 64); err == nil {
							currentArgsForCSR[queryKey] = num
						} else {
							currentArgsForCSR[queryKey] = paramStrValue
						}
					default:
						currentArgsForCSR[queryKey] = paramStrValue
					}
				} else if queryKey != "renderMode" && queryKey != "theme" && queryKey != "componentName" && queryKey != "storyKey" {
					currentArgsForCSR[queryKey] = queryValues[0]
				}
			}
			if targetStoryVariantInDB.Args != nil {
				for defaultArgName, defaultValue := range targetStoryVariantInDB.Args {
					if _, isBool := defaultValue.(bool); isBool {
						if _, presentInQuery := query[defaultArgName]; !presentInQuery {
							if _, setFromDefaults := currentArgsForCSR[defaultArgName]; !setFromDefaults {
								currentArgsForCSR[defaultArgName] = false
							}
						}
					}
				}
			}

		} else { // Fallback / Component View
			isFallback = true
			config["componentTitle"] = targetComponentInDB.Title
			if targetComponentInDB.Path != "" {
				config["componentPath"] = "/static/" + targetComponentInDB.Path
			}
			scriptToLoad = "/static/modules/sandbox/sandbox-fallback.js"
			for queryKey, queryValues := range query {
				if len(queryValues) > 0 && queryKey != "renderMode" && queryKey != "theme" && queryKey != "componentName" && queryKey != "storyKey" {
					currentArgsForCSR[queryKey] = queryValues[0]
				}
			}
		}

		if len(currentArgsForCSR) > 0 {
			config["currentArgs"] = currentArgsForCSR
		}

		configJSON, err := json.Marshal(config)
		if err != nil {
			log.Printf("ServeSandboxContent (CSR): Error marshalling config: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Render the CSR Frame HTML
		frameData := models.CSRFrameData{
			Theme:               theme,
			SandboxConfigJSON:   template.JS(configJSON),
			SandboxScriptToLoad: scriptToLoad,
			IsFallback:          isFallback,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte("<!DOCTYPE html>\n<html lang=\"en\">\n<head>")); err != nil {
			return
		}
		if err := h.Templates.ExecuteTemplate(w, "_csr_frame_head", frameData); err != nil {
			log.Printf("ServeSandboxContent (CSR): Error executing _csr_frame_head: %v", err)
			return
		}
		if _, err := w.Write([]byte("</head>\n<body class=\"" + template.HTMLEscapeString(frameData.Theme) + "-theme\">")); err != nil {
			return
		}
		if err := h.Templates.ExecuteTemplate(w, "_csr_frame_body_elements", frameData); err != nil {
			log.Printf("ServeSandboxContent (CSR): Error executing _csr_frame_body_elements: %v", err)
			return
		}
		if _, err := w.Write([]byte("</body>\n</html>")); err != nil {
			return
		}
		return // --- CSR handled ---
	}

	// --- Handle SSR Mode ---
	if renderMode == "ssr" {
		log.Printf("ServeSandboxContent: Handling SSR request for %s/%s", componentName, storyKey)

		if storyKey == "" {
			log.Printf("ServeSandboxContent (SSR): StoryKey is required for SSR mode.")
			h.serveSSRNotFoundErrorPage(w, componentName, "N/A", "StoryKey is required for SSR mode.")
			return
		}

		var selectedComponent *models.ComponentGroup
		var selectedStoryVariant *models.StoryVariant

		for i := range h.Components {
			if h.Components[i].Name == componentName {
				comp := h.Components[i]
				selectedComponent = &comp
				for j := range comp.Variants {
					if comp.Variants[j].Key == storyKey {
						variant := comp.Variants[j]
						selectedStoryVariant = &variant
						break
					}
				}
				break
			}
		}

		if selectedComponent == nil {
			log.Printf("ServeSandboxContent (SSR): Component '%s' not found.", componentName)
			h.serveSSRNotFoundErrorPage(w, componentName, storyKey, "Component not found.")
			return
		}
		if selectedStoryVariant == nil {
			log.Printf("ServeSandboxContent (SSR): Story '%s' for component '%s' not found.", storyKey, componentName)
			h.serveSSRNotFoundErrorPage(w, componentName, storyKey, "Story not found.")
			return
		}

		storyTemplate := h.Templates.Lookup(selectedStoryVariant.Key)
		if !selectedStoryVariant.HasSSR || storyTemplate == nil {
			errorMessage := fmt.Sprintf("SSR template definition '%s' not found or story not marked for SSR.", selectedStoryVariant.Key)
			log.Printf("ServeSandboxContent (SSR): Story %s/%s not SSR ready. HasSSR=%t, Template Found=%t", componentName, storyKey, selectedStoryVariant.HasSSR, storyTemplate != nil)
			h.serveSSRNotFoundErrorPage(w, componentName, storyKey, errorMessage)
			return
		}

		// --- SSR Possible: Render the story ---
		componentCSSPath := fmt.Sprintf("/static/components/%s/%s.css", selectedComponent.Name, selectedComponent.Name)
		storyArgsForTemplate := make(map[string]interface{})

		if selectedStoryVariant.Args != nil {
			for k, v := range selectedStoryVariant.Args {
				storyArgsForTemplate[k] = v
			}
		}
		for queryKey, queryValues := range query {
			if len(queryValues) == 0 {
				continue
			}
			if defaultValue, knownArg := storyArgsForTemplate[queryKey]; knownArg {
				paramStrValue := queryValues[0]
				switch defaultValue.(type) {
				case bool:
					storyArgsForTemplate[queryKey] = (strings.ToLower(paramStrValue) == "true")
				case int, int64, float32, float64:
					if num, err := strconv.ParseFloat(paramStrValue, 64); err == nil {
						storyArgsForTemplate[queryKey] = num
					} else {
						storyArgsForTemplate[queryKey] = paramStrValue
					}
				default:
					storyArgsForTemplate[queryKey] = paramStrValue
				}
			} else if queryKey != "renderMode" && queryKey != "theme" && queryKey != "componentName" && queryKey != "storyKey" {
				storyArgsForTemplate[queryKey] = queryValues[0] // Include other query params as potential args
			}
		}
		if selectedStoryVariant.Args != nil {
			for defaultArgName, defaultValue := range selectedStoryVariant.Args {
				if _, isBool := defaultValue.(bool); isBool {
					if _, presentInQuery := query[defaultArgName]; !presentInQuery {
						if _, setFromDefaults := storyArgsForTemplate[defaultArgName]; !setFromDefaults {
							storyArgsForTemplate[defaultArgName] = false
						}
					}
				}
			}
		}

		if children, ok := storyArgsForTemplate["Children"].(string); ok {
			storyArgsForTemplate["Children"] = template.HTML(children)
		}
		storyArgsForTemplate["Theme"] = theme

		var ssrOutput bytes.Buffer
		if err := storyTemplate.Execute(&ssrOutput, storyArgsForTemplate); err != nil {
			log.Printf("ServeSandboxContent (SSR): Error executing story template '%s': %v", storyKey, err)
			h.serveSSRExecutionErrorPage(w, componentName, storyKey, err)
			return
		}

		layoutData := struct {
			SSRContent       template.HTML
			Theme            string
			ComponentCSSPath string
		}{
			SSRContent:       template.HTML(ssrOutput.String()),
			Theme:            theme,
			ComponentCSSPath: componentCSSPath,
		}

		var finalOutput bytes.Buffer
		layoutTemplate := h.Templates.Lookup("_ssr_content_layout")
		if layoutTemplate == nil {
			log.Printf("ServeSandboxContent (SSR): Layout template '_ssr_content_layout' not found.")
			http.Error(w, "Internal Server Error: SSR Layout missing", http.StatusInternalServerError)
			return
		}
		if err := layoutTemplate.Execute(&finalOutput, layoutData); err != nil {
			log.Printf("ServeSandboxContent (SSR): Error executing layout template: %v", err)
			http.Error(w, "Internal Server Error: Failed to render SSR layout", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, err := w.Write(finalOutput.Bytes())
		if err != nil {
			log.Printf("ServeSandboxContent (SSR): Error writing final response: %v", err)
		}
		return // --- SSR handled ---
	}

	log.Printf("ServeSandboxContent: Invalid or missing renderMode '%s'", renderMode)
	http.Error(w, "Invalid renderMode specified", http.StatusBadRequest)
}

// NEW Helper function to specifically serve the CSR fallback frame
func (h *AppHandlers) serveCsrFallbackFrame(w http.ResponseWriter, r *http.Request, theme string) {
	query := r.URL.Query() // Get other potential query params
	config := make(map[string]interface{})
	config["componentName"] = "fallback"
	config["renderMode"] = "csr"
	scriptToLoad := "/static/modules/sandbox/sandbox-fallback.js"
	// Pass any other relevant query params to config if needed by fallback script
	for k, v := range query {
		if len(v) > 0 && k != "renderMode" && k != "theme" && k != "componentName" {
			config[k] = v[0]
		}
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		log.Printf("serveCsrFallbackFrame: Error marshalling config: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	frameData := models.CSRFrameData{
		Theme:               theme,
		SandboxConfigJSON:   template.JS(configJSON),
		SandboxScriptToLoad: scriptToLoad,
		IsFallback:          true, // Explicitly set
	}

	// Render the standard CSR frame structure
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte("<!DOCTYPE html>\n<html lang=\"en\">\n<head>")); err != nil {
		return
	}
	if err := h.Templates.ExecuteTemplate(w, "_csr_frame_head", frameData); err != nil {
		log.Printf("serveCsrFallbackFrame: Error executing _csr_frame_head: %v", err)
		return
	}
	if _, err := w.Write([]byte("</head>\n<body class=\"" + template.HTMLEscapeString(frameData.Theme) + "-theme\">")); err != nil {
		return
	}
	if err := h.Templates.ExecuteTemplate(w, "_csr_frame_body_elements", frameData); err != nil {
		log.Printf("serveCsrFallbackFrame: Error executing _csr_frame_body_elements: %v", err)
		return
	}
	if _, err := w.Write([]byte("</body>\n</html>")); err != nil {
		return
	}
}

// Helper function to serve a standard SSR "Not Found" error page
func (h *AppHandlers) serveSSRNotFoundErrorPage(w http.ResponseWriter, comp, story, reason string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusNotFound) // Not found is appropriate if template is missing
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>SSR Template Error</title>
    <link rel="stylesheet" href="/static/styles/global.css" />
    <style>body{padding: var(--space-5); background-color: var(--sage-1); color: var(--sage-12); font-family: var(--default-font-family);}</style>
</head>
<body>
    <h1>Server-Side Rendering Error</h1>
    <p>Could not render story <strong>%s / %s</strong> using SSR.</p>
    <p><strong>Reason:</strong> %s</p>
    <p>Check component name, story key, and ensure the corresponding Go template (<code>{{define "%s"}}</code>) exists and was parsed.</p>
</body>
</html>`,
		template.HTMLEscapeString(comp),
		template.HTMLEscapeString(story),
		template.HTMLEscapeString(reason),
		template.HTMLEscapeString(story))
}

// Helper function to serve a standard SSR "Execution Error" page
func (h *AppHandlers) serveSSRExecutionErrorPage(w http.ResponseWriter, comp, story string, execErr error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <title>SSR Execution Error</title>
    <link rel="stylesheet" href="/static/styles/global.css" />
    <style>body{padding: var(--space-5); background-color: var(--sage-1); color: var(--sage-12); font-family: var(--default-font-family);}</style>
</head>
<body>
    <h1>Server-Side Rendering Error</h1>
    <p>An error occurred while executing the Go template for story <strong>%s / %s</strong>.</p>
    <p><strong>Error:</strong> %s</p>
    <p>Check the Go template syntax and the data being passed to it. See server logs for more details.</p>
</body>
</html>`,
		template.HTMLEscapeString(comp),
		template.HTMLEscapeString(story),
		template.HTMLEscapeString(execErr.Error()))
}

// ServeFullBodyContent serves the entire body content
func (h *AppHandlers) ServeFullBodyContent(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	isMachRequest := r.Header.Get("X-Mach-Request") == "true"

	if len(pathParts) <= 1 || pathParts[1] == "" {
		h.serveBodyOrFullPageForHome(w, r, isMachRequest)
		return
	}

	componentNameParam := pathParts[1]
	storyKeyParam := ""
	if len(pathParts) > 2 {
		storyKeyParam = pathParts[2]
	}

	var currentComponent *models.ComponentGroup
	var currentComponentIdx = -1
	var selectedStoryVariant *models.StoryVariant

	pageComponents := make([]models.ComponentGroup, len(h.Components))
	for i, comp := range h.Components {
		pageComponents[i] = comp
		pageComponents[i].Variants = make([]models.StoryVariant, len(comp.Variants))
		copy(pageComponents[i].Variants, comp.Variants)
		if comp.Name == componentNameParam {
			currentComponent = &pageComponents[i]
			currentComponentIdx = i
		}
	}

	if currentComponent == nil {
		log.Printf("ServeFullBodyContent: Component group not found: %s. Serving 404.", componentNameParam)
		http.NotFound(w, r)
		return
	}

	for i := range pageComponents {
		pageComponents[i].IsSelected = (i == currentComponentIdx)
		if i != currentComponentIdx {
			for j := range pageComponents[i].Variants {
				pageComponents[i].Variants[j].IsSelected = false
			}
		}
	}

	isFallbackScenario := false
	if storyKeyParam == "" && len(currentComponent.Variants) > 0 {
		storyKeyParam = currentComponent.Variants[0].Key
	} else if storyKeyParam == "" && len(currentComponent.Variants) == 0 {
		isFallbackScenario = true
	}

	validStoryKey := false
	if !isFallbackScenario && storyKeyParam != "" {
		for i, variant := range currentComponent.Variants {
			if variant.Key == storyKeyParam {
				pageComponents[currentComponentIdx].Variants[i].IsSelected = true
				selectedStoryVariant = &pageComponents[currentComponentIdx].Variants[i]
				validStoryKey = true
				break
			}
			pageComponents[currentComponentIdx].Variants[i].IsSelected = false
		}
		if !validStoryKey {
			log.Printf("ServeFullBodyContent: Story '%s' not found in '%s'. Fallback to component view or first story.", storyKeyParam, currentComponent.Title)
			if len(currentComponent.Variants) > 0 {
				storyKeyParam = currentComponent.Variants[0].Key
				selectedStoryVariant = &pageComponents[currentComponentIdx].Variants[0]
				pageComponents[currentComponentIdx].Variants[0].IsSelected = true
			} else {
				isFallbackScenario = true
				selectedStoryVariant = nil
				storyKeyParam = ""
			}
		}
	}

	pageTitle := currentComponent.Title
	if selectedStoryVariant != nil {
		pageTitle += " - " + selectedStoryVariant.Title
	} else if isFallbackScenario {
		pageTitle += " - Info"
	}

	var availableModes []string
	if selectedStoryVariant != nil && selectedStoryVariant.HasCSR {
		availableModes = append(availableModes, "csr")
	}
	ssrAvailable := false
	if selectedStoryVariant != nil && selectedStoryVariant.HasSSR && h.Templates.Lookup(selectedStoryVariant.Key) != nil {
		availableModes = append(availableModes, "ssr")
		ssrAvailable = true
	}
	if isFallbackScenario && currentComponent.Path != "" {
		if !slices.Contains(availableModes, "csr") {
			availableModes = append(availableModes, "csr")
		}
	}

	effectiveTheme := r.URL.Query().Get("theme")
	if effectiveTheme != "light" && effectiveTheme != "dark" {
		effectiveTheme = "light"
	}

	effectiveRenderMode := r.URL.Query().Get("renderMode")
	if !slices.Contains(availableModes, effectiveRenderMode) {
		if slices.Contains(availableModes, "csr") {
			effectiveRenderMode = "csr"
		} else if slices.Contains(availableModes, "ssr") {
			effectiveRenderMode = "ssr"
		} else if len(availableModes) > 0 {
			effectiveRenderMode = availableModes[0]
		} else {
			effectiveRenderMode = "csr"
		}
	}
	if isFallbackScenario && effectiveRenderMode != "csr" {
		effectiveRenderMode = "csr"
	}

	viewStoryPath := "/sandbox/" + componentNameParam
	if storyKeyParam != "" {
		viewStoryPath += "/" + storyKeyParam
	}

	handlerPath := r.URL.Path

	data := models.PageData{
		Title:                 pageTitle,
		Components:            pageComponents,
		SelectedComponent:     currentComponent,
		SelectedStoryKey:      storyKeyParam,
		StaticBaseURL:         "/static",
		RenderMode:            effectiveRenderMode,
		AvailableRenderModes:  availableModes,
		SSRAvailable:          ssrAvailable,
		Theme:                 effectiveTheme,
		CurrentPath:           viewStoryPath,
		CanClientSideNavigate: true,
	}

	// --- Construct IframeSrcURL with dynamic path ---
	iframePath := "/sandbox-content/" + componentNameParam
	if storyKeyParam != "" {
		iframePath += "/" + storyKeyParam
	}

	iframeQuery := url.Values{}
	iframeQuery.Set("renderMode", data.RenderMode)
	iframeQuery.Set("theme", data.Theme)

	// Parse args for PageData (for editor) AND for iframe query
	currentStoryArgs := make(map[string]interface{})

	if selectedStoryVariant != nil && selectedStoryVariant.Args != nil {
		requestQueryParams := r.URL.Query()
		for argName, defaultValue := range selectedStoryVariant.Args {
			currentVal := defaultValue
			stringValForQuery := fmt.Sprintf("%v", defaultValue)
			isDefaultBool := false
			if _, ok := defaultValue.(bool); ok {
				isDefaultBool = true
			}
			if queryVals, ok := requestQueryParams[argName]; ok && len(queryVals) > 0 {
				paramStrValue := queryVals[0]
				stringValForQuery = paramStrValue
				if isDefaultBool {
					currentVal = (strings.ToLower(paramStrValue) == "true")
				} else {
					switch defaultValue.(type) {
					case int, int64, float32, float64:
						if num, err := strconv.ParseFloat(paramStrValue, 64); err == nil {
							currentVal = num
						} else {
							currentVal = paramStrValue
						}
					default:
						currentVal = paramStrValue
					}
				}
			} else {
				if isDefaultBool {
					currentVal = false
					stringValForQuery = "false"
				}
			}
			currentStoryArgs[argName] = currentVal
			iframeQuery.Set(argName, stringValForQuery)
		}
		if defaultChildrenVal, ok := selectedStoryVariant.Args["Children"]; ok {
			if _, defaultIsHTML := defaultChildrenVal.(template.HTML); defaultIsHTML {
				if currentChildrenVal, currentOk := currentStoryArgs["Children"]; currentOk {
					if childrenStr, valIsString := currentChildrenVal.(string); valIsString {
						currentStoryArgs["Children"] = template.HTML(childrenStr)
					}
				}
			}
		}
	} else {
		// Still parse query params even if no defaults defined
		requestQueryParams := r.URL.Query()
		for key, vals := range requestQueryParams {
			if len(vals) > 0 && key != "renderMode" && key != "theme" && key != "componentName" && key != "storyKey" {
				iframeQuery.Set(key, vals[0])
			}
		}
	}
	data.SelectedStoryArgs = currentStoryArgs
	data.IframeSrcURL = iframePath + "?" + iframeQuery.Encode()
	// --- End IframeSrcURL construction ---

	toggleThemeQuery := r.URL.Query()
	newThemeForToggle := "dark"
	if effectiveTheme == "light" {
		newThemeForToggle = "dark"
	} else {
		newThemeForToggle = "light"
	}
	toggleThemeQuery.Set("theme", newThemeForToggle)
	data.ToggleThemeURL = (&url.URL{Path: handlerPath, RawQuery: toggleThemeQuery.Encode()}).String()

	resetArgsQuery := r.URL.Query()
	if selectedStoryVariant != nil && selectedStoryVariant.Args != nil {
		for argName := range selectedStoryVariant.Args {
			resetArgsQuery.Del(argName)
		}
	}
	data.ResetArgsURL = (&url.URL{Path: handlerPath, RawQuery: resetArgsQuery.Encode()}).String()

	var modeLinks []models.ModeSwitchLink
	for _, mode := range data.AvailableRenderModes {
		queryForModeSwitch := r.URL.Query()
		queryForModeSwitch.Set("renderMode", mode)
		modeURL := (&url.URL{Path: handlerPath, RawQuery: queryForModeSwitch.Encode()}).String()
		isActive := (mode == data.RenderMode)
		var text string
		if isActive {
			text = strings.ToUpper(mode) + " Active"
		} else {
			text = "Switch to " + strings.ToUpper(mode)
		}
		modeLinks = append(modeLinks, models.ModeSwitchLink{ModeKey: mode, URL: modeURL, IsActive: isActive, Text: text})
	}
	data.ModeSwitchLinks = modeLinks

	var toolbarBuf bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&toolbarBuf, "toolbar-content", data); err != nil {
		log.Printf("ServeFullBodyContent: Error rendering toolbar: %v", err)
	}
	data.ToolbarHTML = template.HTML(toolbarBuf.String())

	var argsEditorBuf bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&argsEditorBuf, "story-args-editor-content", data); err != nil {
		log.Printf("ServeFullBodyContent: Error rendering args editor: %v", err)
	}
	data.StoryArgsEditorHTML = template.HTML(argsEditorBuf.String())

	w.Header().Set("X-Mach-Title", pageTitle)

	if isMachRequest {
		log.Printf("ServeFullBodyContent: Mach request for %s. Serving BODY content.", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.Templates.ExecuteTemplate(w, "full-body-content", data); err != nil {
			log.Printf("ServeFullBodyContent: Error executing template 'full-body-content': %v", err)
			http.Error(w, "Failed to render body content fragment", http.StatusInternalServerError)
		}
	} else {
		log.Printf("ServeFullBodyContent: Non-Mach request for %s. Serving FULL page for HOME using _document_head and full-body-content.", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte("<!DOCTYPE html>\n<html lang=\"en\">")); err != nil {
			log.Printf("serveBodyOrFullPageForHome: Error writing initial HTML: %v", err)
			return
		}
		if err := h.Templates.ExecuteTemplate(w, "_document_head", data); err != nil {
			log.Printf("serveBodyOrFullPageForHome: Error executing _document_head for HOME: %v", err)
			_, _ = w.Write([]byte("</html>")) // Best effort
			return
		}
		if err := h.Templates.ExecuteTemplate(w, "full-body-content", data); err != nil {
			log.Printf("serveBodyOrFullPageForHome: Error executing full-body-content for HOME: %v", err)
			_, _ = w.Write([]byte("</html>")) // Best effort
			return
		}
		if _, err := w.Write([]byte("</html>")); err != nil {
			log.Printf("serveBodyOrFullPageForHome: Error writing closing HTML tag for HOME: %v", err)
		}
	}
}

// serveBodyOrFullPageForHome handles requests to /sandbox-body-swap/ (home context)
func (h *AppHandlers) serveBodyOrFullPageForHome(w http.ResponseWriter, r *http.Request, isMachRequest bool) {
	currentTheme := r.URL.Query().Get("theme")
	if currentTheme != "light" && currentTheme != "dark" {
		currentTheme = "light"
	}

	pageComponents := make([]models.ComponentGroup, len(h.Components))
	copy(pageComponents, h.Components) // Use a copy
	for i := range pageComponents {
		pageComponents[i].IsSelected = false
		for j := range pageComponents[i].Variants {
			pageComponents[i].Variants[j].IsSelected = false
		}
	}

	handlerPath := "/sandbox-body-swap/" // Base path for home context of this handler

	homeToggleThemeQuery := r.URL.Query()
	newThemeForToggle := "dark"
	if currentTheme == "light" {
		newThemeForToggle = "dark"
	} else {
		newThemeForToggle = "light"
	}
	homeToggleThemeQuery.Set("theme", newThemeForToggle)
	homeToggleThemeURL := url.URL{Path: handlerPath, RawQuery: homeToggleThemeQuery.Encode()}

	homeResetArgsQuery := r.URL.Query()
	homeResetArgsURL := url.URL{Path: handlerPath, RawQuery: homeResetArgsQuery.Encode()}

	data := models.PageData{
		Title:                 "Component Playground",
		Components:            pageComponents,
		StaticBaseURL:         "/static",
		Theme:                 currentTheme,
		ToggleThemeURL:        homeToggleThemeURL.String(),
		ResetArgsURL:          homeResetArgsURL.String(),
		CurrentPath:           "/",
		RenderMode:            "csr",
		IframeSrcURL:          "/sandbox-content/fallback?renderMode=csr&theme=" + currentTheme,
		CanClientSideNavigate: true,
	}

	var toolbarBuf bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&toolbarBuf, "toolbar-content", data); err != nil {
		log.Printf("serveBodyOrFullPageForHome: Error rendering toolbar: %v", err)
	}
	data.ToolbarHTML = template.HTML(toolbarBuf.String())

	var argsEditorBuf bytes.Buffer
	if err := h.Templates.ExecuteTemplate(&argsEditorBuf, "story-args-editor-content", data); err != nil {
		log.Printf("serveBodyOrFullPageForHome: Error rendering args editor: %v", err)
	}
	data.StoryArgsEditorHTML = template.HTML(argsEditorBuf.String())

	w.Header().Set("X-Mach-Title", data.Title)

	if isMachRequest {
		log.Printf("serveBodyOrFullPageForHome: Mach request for %s. Serving BODY content for HOME.", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := h.Templates.ExecuteTemplate(w, "full-body-content", data); err != nil {
			log.Printf("serveBodyOrFullPageForHome: Error executing 'full-body-content': %v", err)
			http.Error(w, "Failed to render home body fragment", http.StatusInternalServerError)
		}
	} else {
		log.Printf("serveBodyOrFullPageForHome: Non-Mach request for %s. Serving FULL page for HOME using _document_head and full-body-content.", r.URL.Path)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if _, err := w.Write([]byte("<!DOCTYPE html>\n<html lang=\"en\">")); err != nil {
			log.Printf("serveBodyOrFullPageForHome: Error writing initial HTML: %v", err)
			return
		}
		if err := h.Templates.ExecuteTemplate(w, "_document_head", data); err != nil {
			log.Printf("serveBodyOrFullPageForHome: Error executing _document_head for HOME: %v", err)
			_, _ = w.Write([]byte("</html>")) // Best effort
			return
		}
		if err := h.Templates.ExecuteTemplate(w, "full-body-content", data); err != nil {
			log.Printf("serveBodyOrFullPageForHome: Error executing full-body-content for HOME: %v", err)
			_, _ = w.Write([]byte("</html>")) // Best effort
			return
		}
		if _, err := w.Write([]byte("</html>")); err != nil {
			log.Printf("serveBodyOrFullPageForHome: Error writing closing HTML tag for HOME: %v", err)
		}
	}
}
