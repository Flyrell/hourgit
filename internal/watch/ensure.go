package watch

import (
	"github.com/Flyrell/hourgit/internal/project"
)

// EnsureWatcherService checks if the watcher service should be installed or removed
// based on whether any project has precise mode enabled.
// binPath is the path to the hourgit binary.
func EnsureWatcherService(homeDir, binPath string) error {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	sm, err := NewServiceManager(homeDir)
	if err != nil {
		return nil // unsupported platform, silently skip
	}

	anyPrecise := project.AnyPreciseProject(cfg)

	if anyPrecise && !sm.IsInstalled() {
		if err := sm.Install(binPath); err != nil {
			return err
		}
		return sm.Start()
	}

	if !anyPrecise && sm.IsInstalled() {
		return sm.Remove()
	}

	return nil
}
