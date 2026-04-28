package jobs

import "sync"

type Service struct {
	Dispatcher Dispatcher
	Monitor    Monitor
	Registry   *MemoryRegistry
}

var (
	defaultService *Service
	serviceMu      sync.RWMutex
)

func SetDefault(service *Service) {
	serviceMu.Lock()
	defer serviceMu.Unlock()
	defaultService = service
}

func Default() *Service {
	serviceMu.RLock()
	defer serviceMu.RUnlock()
	return defaultService
}
