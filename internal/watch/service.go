package watch

// ServiceManager manages the daemon as an OS service.
type ServiceManager interface {
	Install(binPath string) error
	Remove() error
	Start() error
	Stop() error
	IsInstalled() bool
	IsRunning() bool
}

// NewServiceManager returns the platform-specific ServiceManager.
// Implemented per-platform in service_<os>.go files.
// Returns nil, error for unsupported platforms.
