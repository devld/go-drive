package driveutil

import (
	"go-drive/common/registry"
	"reflect"
	"testing"
)

func TestReplaceDriveGroupAndUnregisterDrive(t *testing.T) {
	driveRegistry := NewDriveRegistry(registry.NewComponentHolder())
	prefix := "test/registry/"
	t.Cleanup(func() { _ = driveRegistry.ReplaceDriveGroup(prefix, nil) })

	first := DriveFactoryConfig{Type: prefix + "first"}
	second := DriveFactoryConfig{Type: prefix + "second"}
	if e := driveRegistry.ReplaceDriveGroup(prefix, []DriveFactoryConfig{first, second}); e != nil {
		t.Fatalf("ReplaceDriveGroup() error = %v", e)
	}
	got := driveRegistry.GetDrive(first.Type)
	if got == nil || got.Type != first.Type {
		if got == nil {
			t.Fatalf("GetDrive() = nil, want %q", first.Type)
		}
		t.Fatalf("GetDrive() type = %q, want %q", got.Type, first.Type)
	}
	if !driveRegistry.UnregisterDrive(first.Type) {
		t.Fatal("UnregisterDrive() = false, want true")
	}
	if driveRegistry.GetDrive(first.Type) != nil {
		t.Fatal("unregistered drive factory is still available")
	}
	if driveRegistry.GetDrive(second.Type) == nil {
		t.Fatal("unrelated drive factory was removed")
	}
}

func TestGetRegisteredDrivesPreservesRegistrationOrder(t *testing.T) {
	driveRegistry := NewDriveRegistry(registry.NewComponentHolder())
	prefix := "test/order/"
	t.Cleanup(func() { _ = driveRegistry.ReplaceDriveGroup(prefix, nil) })

	fs := DriveFactoryConfig{Type: "fs"}
	s3 := DriveFactoryConfig{Type: "s3"}
	driveRegistry.RegisterDrive(fs)
	driveRegistry.RegisterDrive(s3)

	first := DriveFactoryConfig{Type: prefix + "dropbox"}
	second := DriveFactoryConfig{Type: prefix + "github"}
	third := DriveFactoryConfig{Type: prefix + "qiniu"}
	if e := driveRegistry.ReplaceDriveGroup(prefix, []DriveFactoryConfig{first, second, third}); e != nil {
		t.Fatalf("ReplaceDriveGroup() error = %v", e)
	}

	assertRegisteredTypes(t, driveRegistry, []string{"fs", "s3", first.Type, second.Type, third.Type})

	if !driveRegistry.UnregisterDrive("s3") {
		t.Fatal("UnregisterDrive() = false, want true")
	}
	assertRegisteredTypes(t, driveRegistry, []string{"fs", first.Type, second.Type, third.Type})

	if e := driveRegistry.ReplaceDriveGroup(prefix, []DriveFactoryConfig{third, first}); e != nil {
		t.Fatalf("ReplaceDriveGroup() error = %v", e)
	}
	assertRegisteredTypes(t, driveRegistry, []string{"fs", third.Type, first.Type})
}

func assertRegisteredTypes(t *testing.T, driveRegistry *DriveRegistry, want []string) {
	t.Helper()
	got := driveRegistry.GetRegisteredDrives()
	types := make([]string, 0, len(got))
	for _, factory := range got {
		types = append(types, factory.Type)
	}
	if !reflect.DeepEqual(types, want) {
		t.Fatalf("GetRegisteredDrives() types = %v, want %v", types, want)
	}
}
