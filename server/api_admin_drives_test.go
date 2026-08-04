package server

import (
	"errors"
	"go-drive/common/driveutil"
	"go-drive/common/types"
	"reflect"
	"testing"
)

func TestEscapeDriveInitConfigSecrets(t *testing.T) {
	config := &driveutil.DriveInitConfig{
		Form: []types.FormItem{
			{Field: "password", Type: "password"},
			{Field: "privateKey", Type: "textarea", Secret: "custom marker"},
			{Field: "empty", Type: "password"},
		},
		Value: types.SM{
			"password":   "password value",
			"privateKey": "private key value",
			"empty":      "",
		},
	}

	escapeDriveInitConfigSecrets(config)

	want := types.SM{
		"password":   secretPlaceholder,
		"privateKey": secretPlaceholder,
		"empty":      "",
	}
	if !reflect.DeepEqual(config.Value, want) {
		t.Fatalf("unexpected masked values: got %#v, want %#v", config.Value, want)
	}
}

func TestRestoreDriveInitSecrets(t *testing.T) {
	store := &testDriveDataStore{data: types.SM{"password": "saved password"}}
	data := types.SM{
		"password": secretPlaceholder,
		"name":     "new name",
	}

	if e := restoreDriveInitSecrets(data, store); e != nil {
		t.Fatalf("restoreDriveInitSecrets: %v", e)
	}

	if got, want := data["password"], "saved password"; got != want {
		t.Fatalf("password: got %q, want %q", got, want)
	}
	if got, want := data["name"], "new name"; got != want {
		t.Fatalf("name: got %q, want %q", got, want)
	}
	if got, want := store.loaded, []string{"password"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("loaded keys: got %#v, want %#v", got, want)
	}
}

func TestRestoreDriveInitSecretsLeavesNewValuesUntouched(t *testing.T) {
	store := &testDriveDataStore{data: types.SM{"password": "saved password"}}
	data := types.SM{"password": "new password"}

	if e := restoreDriveInitSecrets(data, store); e != nil {
		t.Fatalf("restoreDriveInitSecrets: %v", e)
	}

	if got, want := data["password"], "new password"; got != want {
		t.Fatalf("password: got %q, want %q", got, want)
	}
	if len(store.loaded) != 0 {
		t.Fatalf("unexpected store load: %#v", store.loaded)
	}
}

func TestRestoreDriveInitSecretsPropagatesStoreError(t *testing.T) {
	wantErr := errors.New("load failed")
	store := &testDriveDataStore{err: wantErr}
	data := types.SM{"password": secretPlaceholder}

	if e := restoreDriveInitSecrets(data, store); !errors.Is(e, wantErr) {
		t.Fatalf("restoreDriveInitSecrets error: got %v, want %v", e, wantErr)
	}
}

type testDriveDataStore struct {
	data   types.SM
	loaded []string
	err    error
}

func (s *testDriveDataStore) Save(types.SM) error { return nil }

func (s *testDriveDataStore) Load(keys ...string) (types.SM, error) {
	s.loaded = append([]string(nil), keys...)
	if s.err != nil {
		return nil, s.err
	}
	result := make(types.SM, len(keys))
	for _, key := range keys {
		result[key] = s.data[key]
	}
	return result, nil
}

func (s *testDriveDataStore) Clear() error { return nil }
