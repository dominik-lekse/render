package render

// WatcherFS is an FS which supports notification for changes in the FS
type WatcherFS interface {
	FS

	// Watch returns a channel through which Event are announced
	Watch() (chan Event, error)
}

// Event represents a single file system notification.
type Event struct {
	Name string // Relative path to the file or directory.
	Op   Op     // File operation that triggered the event.
}

// Op describes a set of file operations.
type Op uint32

// TODO implement localWatcherFS which satisfies WatcherFS using fsnotifier module
type localWatcherFS struct {
}
