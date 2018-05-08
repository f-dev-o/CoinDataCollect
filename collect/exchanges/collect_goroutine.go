package exchanges

// CollectGoroutine exchanges task interface
type CollectGoroutine interface {
	Start() <-chan struct{}
	Stop()
}
