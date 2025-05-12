package models

import (
	"html/template"
)

// StoryVariant holds information about a specific story variant.
type StoryVariant struct {
	Key            string                 // e.g., "Default", "Ghost", serves as ID
	Title          string                 // e.g., "Default Button", "Ghost Button"
	IsSelected     bool                   // True if this is the currently selected story variant
	Args           map[string]interface{} // Parsed arguments from the story.js export
	ArgTypes       map[string]ArgTypeInfo // Added ArgTypes map to store type information
	HasCSR         bool                   // True if client-side rendering is available (always true if discovered from JS)
	HasSSR         bool                   // True if server-side rendering via Go template is available
	HasPendingText bool                   // Flag to indicate if the story has pendingText support
}

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

// ArgType represents the data type of an argument
type ArgType string

const (
	ArgTypeString  ArgType = "string"
	ArgTypeBoolean ArgType = "boolean"
	ArgTypeNumber  ArgType = "number"
	ArgTypeHTML    ArgType = "html"
)

// ComponentGroup holds information about a component and its story variants.
type ComponentGroup struct {
	Name                string         // e.g., "button"
	Title               string         // e.g., "Button"
	Path                string         // Path to the .stories.js file, relative to "static"
	StoryContent        string         // JavaScript content of the .stories.js file (may not be needed in PageData if only path is used by template)
	Variants            []StoryVariant // List of story variants
	IsSelected          bool           // True if this component group is currently selected
	SSRGoHTMLPath       string         // Path to the .stories.gohtml file, relative to "static"
	ComponentGoHTMLPath string         // Path to the component's .gohtml file (e.g. button.gohtml)
	CanSSR              bool           // True if this component has associated Go templates for SSR
}

// ModeSwitchLink holds data for rendering a mode switch button in the toolbar.
type ModeSwitchLink struct {
	ModeKey  string // e.g., "csr", "ssr"
	URL      string // The full URL for this mode switch
	IsActive bool   // True if this mode is the currently active one
	Text     string // Display text for the button, e.g., "Switch to SSR" or "SSR Active"
}

// PageData holds the data to be passed to the HTML templates.
type PageData struct {
	Title                string
	Components           []ComponentGroup
	SelectedComponent    *ComponentGroup
	SelectedStoryKey     string
	StaticBaseURL        string
	IsPartialRequest     bool   // True if the request is for a partial update (e.g., via X-Mach-Request)
	ClientTargetSelector string // Optional: The selector client intends to update, for logging/debug
	Theme                string // e.g., "light", "dark"
	ToggleThemeURL       string // URL for the theme toggle button
	ResetArgsURL         string // URL for the Reset Args button

	// SSR Specific Fields
	RenderMode            string                 // "csr" or "ssr"
	SSRContent            template.HTML          // Rendered HTML content for the story if mode is SSR
	SelectedStoryArgs     map[string]interface{} // Arguments for the selected story, for SSR
	ToolbarHTML           template.HTML          // Rendered HTML for the toolbar, if mode is SSR
	StoryArgsEditorHTML   template.HTML          // Rendered HTML for the story args editor, if mode is SSR
	AvailableRenderModes  []string               // e.g., ["csr", "ssr"]
	ModeSwitchLinks       []ModeSwitchLink       // New: For toolbar mode buttons
	IframeSrcURL          string                 // New: Source URL for the content iframe
	CurrentPath           string                 // New: The current request path, for form actions
	CanClientSideNavigate bool                   // New: True if client is JS-enabled (for mode switching UI)
	SSRAvailable          bool                   // New: True if the selected story has a valid SSR template
}

// CSRFrameData holds data for the sandbox_csr_frame.gohtml template.
type CSRFrameData struct {
	Theme                 string
	SandboxConfigJSON     template.JS      // Marshalled JSON for sandbox config
	SandboxScriptToLoad   string           // Path to sandbox-app.js or sandbox-fallback.js
	IsFallback            bool             // To show different loading text
	ModeSwitchLinks       []ModeSwitchLink // New: For toolbar mode buttons
	IframeSrcURL          string           // New: Source URL for the content iframe
	CurrentPath           string           // New: The current request path, for form actions
	CanClientSideNavigate bool             // New: True if client is JS-enabled (for mode switching UI)
}
