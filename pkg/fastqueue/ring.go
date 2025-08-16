package fastqueue

type Event struct {
	Body []byte
	Sig  []byte
	TS   []byte
}

type Ring struct{ ch chan Event }
func NewRing(capacity int) *Ring { return &Ring{ch: make(chan Event, capacity)} }
func (r *Ring) TryPush(e Event) bool {
	select { case r.ch <- e: return true; default: return false }
}
func (r *Ring) Pop() Event { return <-r.ch }
func (r *Ring) Len() int { return len(r.ch) }
func (r *Ring) C() <-chan Event { return r.ch }
