package drive_util

import "go-drive/common"

type DriveDynamicRegistration func(common.Config) *DriveFactoryConfig

var registry = make(map[string]interface{})

func RegisterDrive(factory DriveFactoryConfig) {
	if _, exists := registry[factory.Type]; exists {
		panic(factory.Type + " already registered")
	}
	registry[factory.Type] = factory
}

func RegisterDynamicDrive(typeName string, factory DriveDynamicRegistration) {
	if _, exists := registry[typeName]; exists {
		panic(typeName + " already registered")
	}
	registry[typeName] = factory
}

func toDriveFactory(typeName string, v interface{}, config common.Config) *DriveFactoryConfig {
	if f, ok := v.(DriveFactoryConfig); ok {
		return &f
	}
	fn := v.(DriveDynamicRegistration)
	f := fn(config)
	if f == nil {
		return nil
	}
	factory := *f
	factory.Type = typeName
	return &factory
}

func GetDrive(typeName string, config common.Config) *DriveFactoryConfig {
	d, ok := registry[typeName]
	if !ok {
		return nil
	}
	return toDriveFactory(typeName, d, config)
}

func GetRegisteredDrives(config common.Config) []DriveFactoryConfig {
	fc := make([]DriveFactoryConfig, 0, len(registry))
	for n, v := range registry {
		f := toDriveFactory(n, v, config)
		if f != nil {
			fc = append(fc, *f)
		}
	}
	return fc
}
