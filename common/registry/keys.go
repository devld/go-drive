package registry

type componentKey struct{ k string }

func (c componentKey) String() string { return c.k }

var (
	KeyConfig           = componentKey{k: "config"}
	KeyVersionSysConfig = componentKey{k: "versionSysConfig"}
	KeyRuntimeStat      = componentKey{k: "runtimeStat"}

	KeyDB = componentKey{k: "db"}

	KeyDriveAccess = componentKey{k: "driveAccess"}
	KeyRootDrive   = componentKey{k: "rootDrive"}

	KeyEventBus       = componentKey{k: "eventBus"}
	KeyTaskRunner     = componentKey{k: "taskRunner"}
	KeyJobExecutor    = componentKey{k: "jobExecutor"}
	KeyFailBanGroup   = componentKey{k: "failBanGroup"}
	KeyThumbnail      = componentKey{k: "thumbnail"}
	KeySearchService  = componentKey{k: "searchService"}
	KeyFileTokenStore = componentKey{k: "fileTokenStore"}

	KeyUserDAO           = componentKey{k: "userDAO"}
	KeyPathMetaDAO       = componentKey{k: "pathMetaDAO"}
	KeyJobDAO            = componentKey{k: "jobDAO"}
	KeyOptionsDAO        = componentKey{k: "optionsDAO"}
	KeyDriveDataDAO      = componentKey{k: "driveDataDAO"}
	KeyFileBucketDAO     = componentKey{k: "fileBucketDAO"}
	KeyDriveCacheDAO     = componentKey{k: "driveCacheDAO"}
	KeyDrivesDAO         = componentKey{k: "drivesDAO"}
	KeyPathPermissionDAO = componentKey{k: "pathPermissionDAO"}
	KeyPathMountDAO      = componentKey{k: "pathMountDAO"}
	KeyGroupDAO          = componentKey{k: "groupDAO"}
)
