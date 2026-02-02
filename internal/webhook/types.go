package webhook

import "time"

// SentryWebhook represents the incoming Sentry webhook payload.
type SentryWebhook struct {
	Action string       `json:"action"`
	Data   WebhookData  `json:"data"`
	Actor  Actor        `json:"actor"`
}

// WebhookData contains the issue and event data.
type WebhookData struct {
	Issue *Issue `json:"issue,omitempty"`
	Event *Event `json:"event,omitempty"`
}

// Issue represents a Sentry issue.
type Issue struct {
	ID          string    `json:"id"`
	ShortID     string    `json:"shortId"`
	Title       string    `json:"title"`
	Culprit     string    `json:"culprit"`
	Permalink   string    `json:"permalink"`
	Logger      string    `json:"logger"`
	Level       string    `json:"level"`
	Status      string    `json:"status"`
	Platform    string    `json:"platform"`
	Project     Project   `json:"project"`
	Metadata    Metadata  `json:"metadata"`
	FirstSeen   time.Time `json:"firstSeen"`
	LastSeen    time.Time `json:"lastSeen"`
	Count       string    `json:"count"`
	UserCount   int       `json:"userCount"`
}

// Project represents a Sentry project.
type Project struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Platform string `json:"platform"`
}

// Metadata contains error metadata.
type Metadata struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	Filename string `json:"filename"`
	Function string `json:"function"`
}

// Event represents a Sentry event.
type Event struct {
	EventID    string      `json:"eventID"`
	Context    interface{} `json:"context"`
	Contexts   Contexts    `json:"contexts"`
	DateCreated time.Time  `json:"dateCreated"`
	Entries    []Entry     `json:"entries"`
	Message    string      `json:"message"`
	Platform   string      `json:"platform"`
	SDK        SDK         `json:"sdk"`
	Tags       []Tag       `json:"tags"`
	Title      string      `json:"title"`
	Type       string      `json:"type"`
	User       *User       `json:"user,omitempty"`
}

// Contexts contains runtime context information.
type Contexts struct {
	Browser  map[string]interface{} `json:"browser,omitempty"`
	OS       map[string]interface{} `json:"os,omitempty"`
	Runtime  map[string]interface{} `json:"runtime,omitempty"`
	Device   map[string]interface{} `json:"device,omitempty"`
}

// Entry represents an event entry (exception, breadcrumbs, etc.).
type Entry struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ExceptionData represents exception entry data.
type ExceptionData struct {
	Values []ExceptionValue `json:"values"`
}

// ExceptionValue represents a single exception.
type ExceptionValue struct {
	Type       string     `json:"type"`
	Value      string     `json:"value"`
	Module     string     `json:"module"`
	Stacktrace Stacktrace `json:"stacktrace"`
	Mechanism  *Mechanism `json:"mechanism,omitempty"`
}

// Stacktrace represents an exception stacktrace.
type Stacktrace struct {
	Frames []Frame `json:"frames"`
}

// Frame represents a single stack frame.
type Frame struct {
	Filename    string                 `json:"filename"`
	AbsPath     string                 `json:"absPath"`
	Module      string                 `json:"module"`
	Package     string                 `json:"package"`
	Platform    string                 `json:"platform"`
	Function    string                 `json:"function"`
	InApp       bool                   `json:"inApp"`
	LineNo      int                    `json:"lineNo"`
	ColNo       int                    `json:"colNo"`
	Context     [][]interface{}        `json:"context"`
	Vars        map[string]interface{} `json:"vars"`
	PreContext  []string               `json:"preContext"`
	PostContext []string               `json:"postContext"`
}

// Mechanism represents error mechanism info.
type Mechanism struct {
	Type    string `json:"type"`
	Handled bool   `json:"handled"`
}

// SDK represents the Sentry SDK info.
type SDK struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Tag represents a Sentry tag.
type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// User represents user information.
type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	IPAddress string `json:"ip_address"`
}

// Actor represents who triggered the action.
type Actor struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ParsedError extracts key error information for the agent.
type ParsedError struct {
	IssueID      string
	ProjectSlug  string
	Title        string
	ErrorType    string
	ErrorMessage string
	Level        string
	Platform     string
	Culprit      string
	Frames       []Frame
	Permalink    string
}

// ParseWebhook extracts error information from the webhook payload.
func ParseWebhook(wh *SentryWebhook) *ParsedError {
	if wh.Data.Issue == nil {
		return nil
	}

	parsed := &ParsedError{
		IssueID:      wh.Data.Issue.ID,
		ProjectSlug:  wh.Data.Issue.Project.Slug,
		Title:        wh.Data.Issue.Title,
		ErrorType:    wh.Data.Issue.Metadata.Type,
		ErrorMessage: wh.Data.Issue.Metadata.Value,
		Level:        wh.Data.Issue.Level,
		Platform:     wh.Data.Issue.Platform,
		Culprit:      wh.Data.Issue.Culprit,
		Permalink:    wh.Data.Issue.Permalink,
		Frames:       make([]Frame, 0),
	}

	// Extract frames from event if available
	if wh.Data.Event != nil {
		for _, entry := range wh.Data.Event.Entries {
			if entry.Type == "exception" {
				if data, ok := entry.Data.(map[string]interface{}); ok {
					if values, ok := data["values"].([]interface{}); ok {
						for _, v := range values {
							if val, ok := v.(map[string]interface{}); ok {
								if st, ok := val["stacktrace"].(map[string]interface{}); ok {
									if frames, ok := st["frames"].([]interface{}); ok {
										for _, f := range frames {
											if fm, ok := f.(map[string]interface{}); ok {
												frame := Frame{
													InApp: getBool(fm, "inApp"),
												}
												if filename, ok := fm["filename"].(string); ok {
													frame.Filename = filename
												}
												if absPath, ok := fm["absPath"].(string); ok {
													frame.AbsPath = absPath
												}
												if function, ok := fm["function"].(string); ok {
													frame.Function = function
												}
												if lineNo, ok := fm["lineNo"].(float64); ok {
													frame.LineNo = int(lineNo)
												}
												if colNo, ok := fm["colNo"].(float64); ok {
													frame.ColNo = int(colNo)
												}
												if module, ok := fm["module"].(string); ok {
													frame.Module = module
												}
												parsed.Frames = append(parsed.Frames, frame)
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return parsed
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}
