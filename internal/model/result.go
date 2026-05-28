package model

const SchemaVersion = "puppt.v1"

// CommandResult is the shared top-level JSON envelope for agent-facing output.
type CommandResult struct {
	SchemaVersion string       `json:"schema_version"`
	Command       string       `json:"command"`
	Status        string       `json:"status"`
	Input         string       `json:"input"`
	Output        *string      `json:"output"`
	Warnings      []Warning    `json:"warnings"`
	Errors        []ErrorItem  `json:"errors"`
	Summary       Summary      `json:"summary"`
	Inspection    *Inspection  `json:"inspection,omitempty"`
	Plan          *EditPlan    `json:"plan,omitempty"`
	Changes       []ChangeItem `json:"changes,omitempty"`
	Skipped       []SkipItem   `json:"skipped,omitempty"`
	Ambiguous     []SkipItem   `json:"ambiguous,omitempty"`
	Unsupported   []SkipItem   `json:"unsupported,omitempty"`
	Validation    *Validation  `json:"validation,omitempty"`
}

type Summary struct {
	Human string `json:"human"`
}

type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Part    string `json:"part,omitempty"`
}

type ErrorItem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Part    string `json:"part,omitempty"`
}

type ChangeItem struct {
	SlideNumber int    `json:"slide_number,omitempty"`
	ObjectID    string `json:"object_id,omitempty"`
	Message     string `json:"message"`
}

type SkipItem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Part    string `json:"part,omitempty"`
}

type Validation struct {
	Valid    bool        `json:"valid"`
	Warnings []Warning   `json:"warnings"`
	Errors   []ErrorItem `json:"errors"`
}
