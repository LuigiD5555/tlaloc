package compiler

type Section struct {
	ID       string `json:"id"`
	Priority int    `json:"priority"`
	Text     string `json:"text"`
}

type PromptIR struct {
	Version  string    `json:"version"`
	Target   string    `json:"target"`
	Sections []Section `json:"sections"`
}
