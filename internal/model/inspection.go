package model

type Inspection struct {
	PresentationPart string         `json:"presentation_part"`
	PartCount        int            `json:"part_count"`
	SlideCount       int            `json:"slide_count"`
	Metadata         Metadata       `json:"metadata"`
	Slides           []Slide        `json:"slides"`
	RepeatedText     []RepeatedText `json:"repeated_text"`
}

type Metadata struct {
	Title   string `json:"title,omitempty"`
	Author  string `json:"author,omitempty"`
	Subject string `json:"subject,omitempty"`
}

type Slide struct {
	Number      int         `json:"number"`
	ID          string      `json:"id"`
	Part        string      `json:"part"`
	Layout      string      `json:"layout,omitempty"`
	LayoutName  string      `json:"layout_name,omitempty"`
	Master      string      `json:"master,omitempty"`
	MasterName  string      `json:"master_name,omitempty"`
	Title       string      `json:"title,omitempty"`
	VisibleText []TextBlock `json:"visible_text"`
	Notes       []TextBlock `json:"notes"`
	Images      []MediaRef  `json:"images"`
	Media       []MediaRef  `json:"media"`
	Warnings    []Warning   `json:"warnings"`
}

type TextBlock struct {
	ObjectID string   `json:"object_id"`
	Text     string   `json:"text"`
	Runs     []string `json:"runs"`
}

type MediaRef struct {
	ObjectID         string `json:"object_id"`
	Kind             string `json:"kind"`
	Relationship     string `json:"relationship"`
	Target           string `json:"target"`
	ContentType      string `json:"content_type,omitempty"`
	Extension        string `json:"extension,omitempty"`
	RelationshipType string `json:"relationship_type,omitempty"`
}

type RepeatedText struct {
	Text  string `json:"text"`
	Count int    `json:"count"`
}
