package script

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	s "go-drive/script"
	"io"
	"maps"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

//go:embed helper.js
var helperScript []byte
var baseVM *s.VM

const (
	DefaultPoolMaxTotal = 100
	DefaultPoolMaxIdle  = 50
	DefaultPoolMinIdle  = 10
	DefaultPoolIdleTime = time.Duration(30 * time.Minute)
)

func init() {
	vm, e := s.NewVM()
	if e != nil {
		panic(e)
	}

	_, e = vm.RunNamed(context.Background(), "helper.js", helperScript)
	if e != nil {
		panic(e)
	}

	baseVM = vm
}

var t = i18n.TPrefix("drive.script.")

const (
	scriptConfigField = "__script"
	poolConfigField   = "__pool"
)

func withScriptName(name string, config types.SM) types.SM {
	result := make(types.SM, len(config)+1)
	maps.Copy(result, config)
	result[scriptConfigField] = name
	return result
}

func scriptFileName(name string) (string, error) {
	if name == "" {
		return "", err.NewNotAllowedMessageError(i18n.T("drive.not_configured"))
	}
	if name == "." || name == ".." || strings.ContainsAny(name, `/\\`) {
		return "", err.NewBadRequestError("invalid script drive name")
	}
	return name + ".js", nil
}

func validateScriptForm(form []types.FormItem) error {
	for _, item := range form {
		if strings.HasPrefix(item.Field, "_") {
			return err.NewBadRequestError("script form fields must not start with '_': " + item.Field)
		}
	}
	return nil
}

// GetDriveScriptConfigForm returns the static form declared by a script.
func GetDriveScriptConfigForm(ctx context.Context, config common.Config, name string) ([]types.FormItem, error) {
	file, e := scriptFileName(name)
	if e != nil {
		return nil, e
	}
	vm, e := createVm(ctx, config, file)
	if e != nil {
		return nil, e
	}
	defer func() { _ = vm.Dispose() }()

	formValue, e := vm.GetValue("__driveConfigForm")
	if e != nil {
		return nil, e
	}
	form := make([]types.FormItem, 0)
	if formValue != nil && !formValue.IsNil() {
		formValue.ParseInto(&form)
	}
	if e := validateScriptForm(form); e != nil {
		return nil, e
	}
	return form, nil
}

func newScriptDrive(ctx context.Context, config types.SM, driveUtils driveutil.DriveUtils) (types.IDrive, error) {
	selectedScript, e := scriptFileName(config[scriptConfigField])
	if e != nil {
		return nil, e
	}

	poolConfig, e := parsePoolConfig(config[poolConfigField])
	if e != nil {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.script.invalid_pool_config", e.Error()))
	}

	vm, e := createVm(ctx, driveUtils.Config, selectedScript)
	if e != nil {
		return nil, e
	}

	d := &ScriptDrive{
		baseVM:   vm,
		data:     make(map[string]json.RawMessage),
		writable: true,
	}
	d.cache = driveUtils.CreateCache(d.deserializeEntry)

	vm.Set("__setData", s.WrapVmCall(vm, d.setData))
	vm.Set("__getData", s.WrapVmCall(vm, d.getData))

	scriptUtils := newScriptDriveUtils(driveUtils)
	scriptUtils.cache = &scriptDriveCache{d.cache}

	createdVal, e := vm.Call(ctx, "__driveCreate", s.NewContext(vm, ctx), config, scriptUtils)

	if e != nil {
		_ = d.Dispose()
		return nil, e
	}
	var created struct {
		Writable      bool
		EntryCacheTTL string
	}
	if createdVal != nil && !createdVal.IsNil() {
		createdVal.ParseInto(&created)
		d.writable = created.Writable
		ttl := types.SV(created.EntryCacheTTL).Duration(0)
		if ttl > 0 {
			d.cacheTTL = ttl
		}
	}
	d.inspectMethods(vm)
	vm.Set("selfDrive", s.NewDrive(d))
	d.pool = s.NewVMPool(vm, poolConfig)

	return d, nil
}

func (sd *ScriptDrive) hasMethod(vm *s.VM, name string) bool {
	v, e := vm.GetValue("__drive_" + name)
	return e == nil && v != nil && !v.IsNil()
}

func (sd *ScriptDrive) inspectMethods(vm *s.VM) {
	sd.has.meta = sd.hasMethod(vm, "meta")
	sd.has.save = sd.hasMethod(vm, "save")
	sd.has.makeDir = sd.hasMethod(vm, "makeDir")
	sd.has.copy = sd.hasMethod(vm, "copy")
	sd.has.move = sd.hasMethod(vm, "move")
	sd.has.delete = sd.hasMethod(vm, "delete")
	sd.has.upload = sd.hasMethod(vm, "upload")
	sd.has.getReader = sd.hasMethod(vm, "getReader")
	sd.has.getURL = sd.hasMethod(vm, "getURL")
	sd.has.getThumbnail = sd.hasMethod(vm, "getThumbnail")
}

func initConfig(ctx context.Context, config types.SM, driveUtils driveutil.DriveUtils) (*driveutil.DriveInitConfig, error) {
	selectedScript, e := scriptFileName(config[scriptConfigField])
	if e != nil {
		return nil, e
	}

	vm, e := createVm(ctx, driveUtils.Config, selectedScript)
	if e != nil {
		return nil, e
	}
	defer func() { _ = vm.Dispose() }()

	initConfigVal, e := vm.GetValue("__driveInitConfig")
	if e != nil {
		return nil, e
	}
	if initConfigVal == nil || initConfigVal.IsNil() {
		return nil, nil
	}

	v, e := vm.Call(ctx, "__driveInitConfig", s.NewContext(vm, ctx), config, newScriptDriveUtils(driveUtils))
	if e != nil {
		return nil, e
	}

	if v == nil || v.IsNil() {
		return nil, nil
	}
	vmCfg := &driveutil.DriveInitConfig{}
	v.ParseInto(vmCfg)
	if e := validateScriptForm(vmCfg.Form); e != nil {
		return nil, e
	}
	return vmCfg, nil
}

func init_(ctx context.Context, data, config types.SM, driveUtils driveutil.DriveUtils) error {
	selectedScript, e := scriptFileName(config[scriptConfigField])
	if e != nil {
		return e
	}
	vm, e := createVm(ctx, driveUtils.Config, selectedScript)
	if e != nil {
		return e
	}
	defer func() { _ = vm.Dispose() }()

	initConfigVal, e := vm.GetValue("__driveInit")
	if e != nil {
		return e
	}
	if initConfigVal == nil || initConfigVal.IsNil() {
		return nil
	}

	_, e = vm.Call(ctx, "__driveInit", s.NewContext(vm, ctx), data, config, newScriptDriveUtils(driveUtils))
	return e
}

// parsePoolConfig parses config like this: MaxTotal,MaxIdle,MinIdle,IdleTime
func parsePoolConfig(arg string) (*s.VMPoolConfig, error) {
	args := strings.Split(strings.ReplaceAll(arg, " ", ""), ",")
	c := &s.VMPoolConfig{
		MaxTotal: DefaultPoolMaxTotal,
		MaxIdle:  DefaultPoolMaxIdle,
		MinIdle:  DefaultPoolMinIdle,
		IdleTime: DefaultPoolIdleTime,
	}

	if len(args) > 0 {
		c.MaxTotal = types.SV(args[0]).Int(DefaultPoolMaxTotal)
	}
	if len(args) > 1 {
		c.MaxIdle = types.SV(args[1]).Int(DefaultPoolMaxIdle)
	}
	if len(args) > 2 {
		c.MinIdle = types.SV(args[2]).Int(DefaultPoolMinIdle)
	}
	if len(args) > 3 {
		c.IdleTime = types.SV(args[3]).Duration(DefaultPoolIdleTime)
	}

	if c.MaxIdle < c.MinIdle {
		return nil, errors.New("MaxIdle must be greater than or equal to MinIdle")
	}
	if c.MaxTotal <= 0 {
		return nil, errors.New("MaxTotal must be greater than zero")
	}
	if c.MaxIdle < 0 {
		return nil, errors.New("MaxIdle must not be negative")
	}
	if c.MinIdle < 0 {
		return nil, errors.New("MinIdle must not be negative")
	}
	if c.MaxTotal < c.MinIdle {
		return nil, errors.New("MaxTotal must be greater than or equal to MinIdle")
	}
	return c, nil
}

func newScriptDriveUtils(utils driveutil.DriveUtils) *scriptDriveUtils {
	return &scriptDriveUtils{utils.CreateCache, nil, driveDataStore{utils.Data}, utils.Config}
}

type scriptDriveUtils struct {
	createCache driveutil.DriveCacheFactory
	cache       *scriptDriveCache

	Data   driveDataStore
	Config common.Config
}

func (sdu *scriptDriveUtils) CreateCache() *scriptDriveCache {
	if sdu.cache != nil {
		return sdu.cache
	}
	return &scriptDriveCache{sdu.createCache(nil)}
}

func (sdu *scriptDriveUtils) OAuthInitConfig(or driveutil.OAuthRequest,
	cred driveutil.OAuthCredentials) *oauthInitConfigResult {
	c, oauthHolder, e := driveutil.OAuthInitConfig(or, cred, sdu.Data.data)
	if e != nil {
		s.ThrowDetachedError(e)
	}
	var wrapped *oauthHolderWrapper
	if oauthHolder != nil {
		wrapped = &oauthHolderWrapper{oauthHolder}
	}
	return &oauthInitConfigResult{c, wrapped}
}

func (sdu *scriptDriveUtils) OAuthInit(ctx s.Context,
	data types.SM, or driveutil.OAuthRequest,
	cred driveutil.OAuthCredentials) *oauthHolderWrapper {
	oauthHolder, e := driveutil.OAuthInit(s.GetContext(ctx), or, data, cred, sdu.Data.data)
	if e != nil {
		s.ThrowDetachedError(e)
	}
	if oauthHolder == nil {
		return nil
	}
	return &oauthHolderWrapper{oauthHolder}
}

func (sdu *scriptDriveUtils) OAuthLoad(o driveutil.OAuthRequest,
	cred driveutil.OAuthCredentials) *oauthHolderWrapper {
	oauthHolder, e := driveutil.OAuthLoad(o, cred, sdu.Data.data)
	if e != nil {
		s.ThrowDetachedError(e)
	}
	if oauthHolder == nil {
		return nil
	}
	return &oauthHolderWrapper{oauthHolder}
}

type driveDataStore struct {
	data driveutil.DriveDataStore
}

func (d driveDataStore) Save(data types.SM) {
	if e := d.data.Save(data); e != nil {
		s.ThrowDetachedError(e)
	}
}

func (d driveDataStore) Load(key string, keys ...string) types.SM {
	r, e := d.data.Load(key, keys...)
	if e != nil {
		s.ThrowDetachedError(e)
	}
	return r
}

type oauthInitConfigResult struct {
	Config      *driveutil.DriveInitConfig
	OAuthHolder *oauthHolderWrapper
}

type oauthHolderWrapper struct {
	oauthHolder *driveutil.OAuthHolder
}

func (or *oauthHolderWrapper) Token(ctx s.Context) *oauth2.Token {
	c := s.GetContext(ctx)
	if c == nil {
		s.ThrowDetachedError(errors.New("OAuthHolder.Token requires a context"))
	}
	t, e := or.oauthHolder.Token(c)
	if e != nil {
		s.ThrowDetachedError(e)
	}
	return t
}

func createVm(ctx context.Context, config common.Config, script string) (*s.VM, error) {
	scriptBytes, e := readDriveScriptFile(script, config)
	if e != nil {
		return nil, e
	}

	vm := baseVM.Fork()
	meta, ok, e := parseDriveScriptMeta(scriptBytes, script)
	if e != nil {
		_ = vm.Dispose()
		return nil, e
	}
	if ok {
		vm.Set("__driveScriptVersion", meta.Version)
	} else {
		vm.Set("__driveScriptVersion", "")
	}
	vm.Set("__driveUploaderName", strings.TrimSuffix(script, ".js"))
	vm.Set("__ownEntry", s.WrapVmCall(vm, ownEntry))

	_, e = vm.RunNamed(ctx, script, scriptBytes)
	if e != nil {
		_ = vm.Dispose()
		return nil, e
	}
	return vm, nil
}

func ownEntry(vm *s.VM, args s.Values) any {
	from := s.GetEntry(args.Get(0).Raw())
	if from == nil {
		return nil
	}
	selfVal, e := vm.GetValue("selfDrive")
	if e != nil || selfVal.IsNil() {
		return nil
	}
	self := s.GetDrive(selfVal.Raw())
	if self == nil {
		return nil
	}
	owned := driveutil.GetSelfEntry(self, from)
	if owned == nil {
		return nil
	}
	result := map[string]any{
		"Path":    owned.Path(),
		"IsDir":   owned.Type().IsDir(),
		"Size":    owned.Size(),
		"ModTime": owned.ModTime(),
	}
	if ce, ok := owned.(driveutil.CacheableEntry); ok {
		result["Data"] = ce.EntryData()
	}
	return result
}

// wrapReader adapts an io.Reader into an io.ReadCloser. If reader already is a
// ReadCloser it is returned as-is, otherwise a no-op Close is added.
func wrapReader(reader io.Reader) io.ReadCloser {
	if rc, ok := reader.(io.ReadCloser); ok {
		return rc
	}
	return fakeCloseReader{reader}
}

type fakeCloseReader struct {
	io.Reader
}

func (fcr fakeCloseReader) Close() error {
	return nil
}

// wrapContentReader adapts an io.ReadCloser (already detached from the VM, so
// the caller owns closing it) into an IContentReader for thumbnail responses.
func wrapContentReader(rc io.ReadCloser) types.IContentReader {
	return readCloserContentReader{rc}
}

type readCloserContentReader struct {
	rc io.ReadCloser
}

func (r readCloserContentReader) GetReader(_ context.Context, start, size int64) (io.ReadCloser, error) {
	// The underlying value is a single-shot stream; range requests are not
	// supported. start < 0 / size < 0 means "the whole content".
	if start > 0 || size > 0 {
		return nil, err.NewUnsupportedError()
	}
	return r.rc, nil
}

func (r readCloserContentReader) GetURL(_ context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}
