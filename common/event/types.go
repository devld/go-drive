package event

const (
	// EntryAccessed fires when an entry is accessed.
	// The args is (types.DriveListenerContext, path).
	EntryAccessed = "drive:entry_accessed"
	// EntryUpdated fires when an entry is added or updated.
	// The args is (types.DriveListenerContext, path, includeDescendants bool).
	EntryUpdated = "drive:entry_updated"
	// EntryDeleted fires when an entry is deleted.
	// The args is (types.DriveListenerContext, path).
	EntryDeleted = "drive:entry_deleted"
)
