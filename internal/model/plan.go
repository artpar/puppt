package model

type EditSpec struct {
	Operation   string     `json:"operation"`
	Target      TargetSpec `json:"target"`
	Replacement string     `json:"replacement,omitempty"`
}

type TargetSpec struct {
	Type        string `json:"type"`
	Scope       string `json:"scope,omitempty"`
	SlideNumber int    `json:"slide_number,omitempty"`
	Text        string `json:"text,omitempty"`
	ObjectID    string `json:"object_id,omitempty"`
	Property    string `json:"property,omitempty"`
}

type EditPlan struct {
	Operation string        `json:"operation"`
	Target    TargetSpec    `json:"target"`
	Matches   []TargetMatch `json:"matches"`
	Status    string        `json:"status"`
	Message   string        `json:"message"`
}

type TargetMatch struct {
	SlideNumber int    `json:"slide_number,omitempty"`
	SlideID     string `json:"slide_id,omitempty"`
	ObjectID    string `json:"object_id,omitempty"`
	Kind        string `json:"kind"`
	Text        string `json:"text,omitempty"`
	Property    string `json:"property,omitempty"`
}
