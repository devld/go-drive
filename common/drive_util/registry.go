package drive_util

type DrivesRegistry map[string]DriveFactoryConfig

var registry DrivesRegistry = make(map[string]DriveFactoryConfig)

func RegisterDrive(factory DriveFactoryConfig) {
	registry[factory.Type] = factory
}

func GetDrive(typeName string) *DriveFactoryConfig {
	d, ok := registry[typeName]
	if !ok {
		return nil
	}
	return &d
}

func GetRegisteredDrives() []DriveFactoryConfig {
	fc := make([]DriveFactoryConfig, 0, len(registry))
	for _, v := range registry {
		fc = append(fc, v)
	}
	return fc
}
