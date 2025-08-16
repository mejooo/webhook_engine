package events

type Raw struct {
	Source string `json:"source"`
	Format string `json:"format"`
	Body   []byte `json:"body"`
}
type Valid struct {
	Raw Raw `json:"raw"`
}
