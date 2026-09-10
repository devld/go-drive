package script

import (
	"context"
	_ "embed"
	"errors"
	"io"
	"maps"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-drive/common"
	"go-drive/common/driveutil"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	s "go-drive/script"

	"golang.org/x/oauth2"
)

//go:embed helper.js
var helperScript []byte
var helperProgram = s.MustCompile("helper.js", helperScript)

const (
	DefaultPoolMaxTotal = 100
	DefaultPoolMaxIdle  = 50
	DefaultPoolMinIdle  = 10
	DefaultPoolIdleTime = time.Duration(30 * time.Minute)
)

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
	if e := validateScriptName(name); e != nil {
		return "", e
	}
	return name + ".js", nil
}

func validateScriptName(name string) error {
	if name == "." || name == ".." || strings.ContainsRune(name, 0) || strings.ContainsAny(name, `/\`) {
		return err.NewBadRequestError("invalid script drive name")
	}
	if filepath.Base(name) != name {
		return err.NewBadRequestError("invalid script drive name")
	}
	return nil
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

	form := make([]types.FormItem, 0)
	e = vm.Do(ctx, func() error {
		formValue, e := vm.GetValue("__driveConfigForm")
		if e != nil {
			return e
		}
		if formValue != nil && !formValue.IsNil() {
			if e := formValue.ParseInto(&form); e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		return nil, e
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

	compiled, e := compileDriveScript(driveUtils.Config, selectedScript)
	if e != nil {
		return nil, e
	}

	d := &ScriptDrive{
		name:     config[scriptConfigField],
		data:     make(map[string]any),
		writable: true,
	}
	d.cache = driveUtils.CreateCache(d.deserializeEntry)

	var initializeOnce sync.Once
	var initializeErr error
	initializer := func(vmCtx context.Context, vm *s.VM) error {
		if e := initializeDriveScriptVM(vmCtx, vm, compiled, d); e != nil {
			return e
		}

		runtimeConfig := vm.ToJSValue(config)
		scriptUtils := newScriptDriveUtils(vm, driveUtils, &d.oauth, newScriptDriveCache(vm, d.cache))
		return vm.Do(vmCtx, func() error {
			createdVal, e := vm.Call(vmCtx, "__driveCreate", runtimeConfig, scriptUtils)
			if e != nil {
				return e
			}
			initializeOnce.Do(func() {
				initializeErr = d.applyCreated(createdVal)
				if initializeErr == nil {
					d.inspectMethods(vm)
				}
			})
			if initializeErr != nil {
				return initializeErr
			}
			return vm.DefineGlobal("selfDrive", d)
		})
	}

	d.pool, e = s.NewVMPool(ctx, initializer, poolConfig)
	if e != nil {
		_ = d.Dispose()
		return nil, e
	}
	// MinIdle may be zero; force one initialization so metadata and supported
	// methods are known before the Drive becomes visible.
	vm, e := d.pool.Get(ctx)
	if e != nil {
		_ = d.Dispose()
		return nil, e
	}
	if e := d.pool.Return(context.Background(), vm); e != nil {
		_ = d.Dispose()
		return nil, e
	}
	if e := d.startIntervals(); e != nil {
		_ = d.Dispose()
		return nil, e
	}

	return d, nil
}

func (sd *ScriptDrive) jsFunInitData(vm *s.VM, args s.Values) any {
	data := args.Get(0)
	keys := data.Keys()
	sd.mu.Lock()
	defer sd.mu.Unlock()
	for _, key := range keys {
		if _, exists := sd.data[key]; exists {
			continue
		}
		cloned, e := vm.FromJSValue(data.Get(key))
		if e != nil {
			vm.ThrowTypeError("shared state must be JSON serializable: " + e.Error())
		}
		sd.data[key] = cloned
	}
	return nil
}

func (sd *ScriptDrive) applyCreated(createdVal *s.Value) error {
	if createdVal == nil || createdVal.IsNil() {
		return nil
	}
	sd.writable = true
	if v := createdVal.Get("writable"); !v.IsNil() {
		sd.writable = v.Bool()
	}
	if v := createdVal.Get("entryCacheTTL"); !v.IsNil() {
		ttl, ok := s.ParseDuration(v)
		if !ok {
			return err.NewNotAllowedMessageError("entryCacheTTL requires a Duration or duration string")
		}
		if ttl > 0 {
			sd.cacheTTL = ttl
		}
	}
	return sd.prepareIntervals(createdVal.Get("intervals"))
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
	sd.has.onInterval = sd.hasMethod(vm, "onInterval")
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

	var vmCfg *driveutil.DriveInitConfig
	e = vm.Do(ctx, func() error {
		entry, e := vm.GetValue("__driveInitConfig")
		if e != nil || entry.IsNil() {
			return e
		}
		v, e := vm.Call(ctx, "__driveInitConfig", vm.ToJSValue(config), newScriptDriveUtils(vm, driveUtils, nil, nil))
		if e != nil {
			return e
		}
		if v == nil || v.IsNil() {
			return nil
		}
		cfg := &driveutil.DriveInitConfig{}
		if e := v.ParseInto(cfg); e != nil {
			return e
		}
		if e := validateScriptForm(cfg.Form); e != nil {
			return e
		}
		vmCfg = cfg
		return nil
	})
	if e != nil {
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

	return vm.Do(ctx, func() error {
		entry, e := vm.GetValue("__driveInit")
		if e != nil || entry.IsNil() {
			return e
		}
		_, e = vm.Call(ctx, "__driveInit", vm.ToJSValue(data), vm.ToJSValue(config), newScriptDriveUtils(vm, driveUtils, nil, nil))
		return e
	})
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

func newScriptDriveUtils(vm *s.VM, utils driveutil.DriveUtils, oauth *oauthHolderShare, cache *scriptDriveCache) *scriptDriveUtils {
	return &scriptDriveUtils{
		vm:          vm,
		createCache: utils.CreateCache,
		cache:       cache,
		oauth:       oauth,
		Data:        &driveDataStore{vm, utils.Data},
		Config: rootConfig{
			OAuthRedirectURI: utils.Config.OAuthRedirectURI,
			Version:          utils.Config.Version,
			RevHash:          utils.Config.RevHash,
			BuildAt:          utils.Config.BuildAt,
		},
	}
}

// rootConfig is the script-facing subset of common.Config (RootConfig in drive.d.ts).
// It is a value type so each VM gets an independent copy without JSON cloning, and
// scripts cannot read unrelated server settings such as database credentials.
type rootConfig struct {
	OAuthRedirectURI string
	Version          string
	RevHash          string
	BuildAt          string
}

type scriptDriveUtils struct {
	vm          *s.VM
	createCache driveutil.DriveCacheFactory
	cache       *scriptDriveCache
	oauth       *oauthHolderShare

	Data   *driveDataStore
	Config rootConfig
}

// oauthHolderShare keeps one *driveutil.OAuthHolder per credentials/endpoint
// for a Drive. Pooled VMs only wrap it for their own Runtime.
type oauthHolderShare struct {
	mu      sync.Mutex
	holders map[string]*driveutil.OAuthHolder
}

func oauthShareKey(o driveutil.OAuthRequest, cred driveutil.OAuthCredentials) string {
	return strings.Join([]string{
		cred.ClientID,
		cred.ClientSecret,
		o.Endpoint.AuthURL,
		o.Endpoint.TokenURL,
		strconv.Itoa(int(o.Endpoint.AuthStyle)),
		o.RedirectURL,
		strings.Join(o.Scopes, " "),
	}, "\x00")
}

func (c *oauthHolderShare) get(key string, load func() (*driveutil.OAuthHolder, error)) (*driveutil.OAuthHolder, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if h, ok := c.holders[key]; ok {
		return h, nil
	}
	h, e := load()
	if e != nil || h == nil {
		return h, e
	}
	if c.holders == nil {
		c.holders = make(map[string]*driveutil.OAuthHolder)
	}
	c.holders[key] = h
	return h, nil
}

func (c *oauthHolderShare) put(key string, h *driveutil.OAuthHolder) {
	if h == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.holders == nil {
		c.holders = make(map[string]*driveutil.OAuthHolder)
	}
	c.holders[key] = h
}

func (sdu *scriptDriveUtils) CreateCache(_ *s.VM, _ s.Values) any {
	if sdu.cache != nil {
		return sdu.cache
	}
	return newScriptDriveCache(sdu.vm, sdu.createCache(nil))
}

func (d *driveDataStore) Save(_ *s.VM, args s.Values) any {
	if e := d.data.Save(args.Get(0).SM()); e != nil {
		d.vm.ThrowError(e)
	}
	return nil
}

func (d *driveDataStore) Load(_ *s.VM, args s.Values) any {
	if args.Len() == 0 {
		d.vm.ThrowTypeError("data.load requires a key")
	}
	extra := make([]string, 0, args.Len()-1)
	for i := 1; i < args.Len(); i++ {
		extra = append(extra, args.Get(i).String())
	}
	r, e := d.data.Load(args.Get(0).String(), extra...)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return r
}

func (sdu *scriptDriveUtils) OAuthInitConfig(_ *s.VM, args s.Values) any {
	req := s.Parse[driveutil.OAuthRequest](args.Get(0))
	cred := s.Parse[driveutil.OAuthCredentials](args.Get(1))
	c, holder, e := driveutil.OAuthInitConfig(req, cred, sdu.Data.data)
	if e != nil {
		sdu.vm.ThrowError(e)
	}
	if sdu.oauth != nil {
		holder, e = sdu.oauth.get(oauthShareKey(req, cred), func() (*driveutil.OAuthHolder, error) {
			return holder, nil
		})
		if e != nil {
			sdu.vm.ThrowError(e)
		}
	}
	result := map[string]any{"config": c}
	if holder != nil {
		result["oauthHolder"] = &oauthHolderWrapper{sdu.vm, holder}
	}
	return result
}

func (sdu *scriptDriveUtils) OAuthInit(_ *s.VM, args s.Values) any {
	req := s.Parse[driveutil.OAuthRequest](args.Get(1))
	cred := s.Parse[driveutil.OAuthCredentials](args.Get(2))
	holder, e := driveutil.OAuthInit(sdu.vm.ExecutionContext(), req, args.Get(0).SM(), cred, sdu.Data.data)
	if e != nil {
		sdu.vm.ThrowError(e)
	}
	if sdu.oauth != nil {
		sdu.oauth.put(oauthShareKey(req, cred), holder)
	}
	if holder == nil {
		return nil
	}
	return &oauthHolderWrapper{sdu.vm, holder}
}

func (sdu *scriptDriveUtils) OAuthLoad(_ *s.VM, args s.Values) any {
	req := s.Parse[driveutil.OAuthRequest](args.Get(0))
	cred := s.Parse[driveutil.OAuthCredentials](args.Get(1))
	load := func() (*driveutil.OAuthHolder, error) {
		return driveutil.OAuthLoad(req, cred, sdu.Data.data)
	}
	var holder *driveutil.OAuthHolder
	var e error
	if sdu.oauth != nil {
		holder, e = sdu.oauth.get(oauthShareKey(req, cred), load)
	} else {
		holder, e = load()
	}
	if e != nil {
		sdu.vm.ThrowError(e)
	}
	if holder == nil {
		return nil
	}
	return &oauthHolderWrapper{sdu.vm, holder}
}

type driveDataStore struct {
	vm   *s.VM
	data driveutil.DriveDataStore
}

type oauthHolderWrapper struct {
	vm          *s.VM
	oauthHolder *driveutil.OAuthHolder
}

func (or *oauthHolderWrapper) Token(_ *s.VM, _ s.Values) any {
	c := or.vm.ExecutionContext()
	t, e := or.oauthHolder.Token(c)
	if e != nil {
		or.vm.ThrowError(e)
	}
	return oauthTokenJS(t)
}

func (or *oauthHolderWrapper) Refresh(_ *s.VM, _ s.Values) any {
	c := or.vm.ExecutionContext()
	t, e := or.oauthHolder.Refresh(c)
	if e != nil {
		or.vm.ThrowError(e)
	}
	return oauthTokenJS(t)
}

func oauthTokenJS(t *oauth2.Token) map[string]any {
	if t == nil {
		return nil
	}
	return map[string]any{
		"accessToken":  t.AccessToken,
		"tokenType":    t.TokenType,
		"refreshToken": t.RefreshToken,
		"expiry":       t.Expiry,
	}
}

type compiledDriveScript struct {
	program *s.Program
	version string
	name    string
}

func compileDriveScript(config common.Config, script string) (*compiledDriveScript, error) {
	scriptBytes, e := readDriveScriptFile(script, config)
	if e != nil {
		return nil, e
	}

	meta, ok, e := parseDriveScriptMeta(scriptBytes, script)
	if e != nil {
		return nil, e
	}
	version := ""
	if ok {
		version = meta.Version
	}
	program, e := s.Compile(script, scriptBytes)
	if e != nil {
		return nil, e
	}
	return &compiledDriveScript{
		program: program,
		version: version,
		name:    strings.TrimSuffix(script, ".js"),
	}, nil
}

func initializeDriveScriptVM(ctx context.Context, vm *s.VM, compiled *compiledDriveScript, drive *ScriptDrive) error {
	bridge := map[string]any{
		"version": compiled.version,
		"name":    compiled.name,
	}
	if drive != nil {
		bridge["initData"] = s.NativeFunction(drive.jsFunInitData)
		bridge["setData"] = s.NativeFunction(drive.jsFunSetData)
		bridge["getData"] = s.NativeFunction(drive.jsFunGetData)
	}
	if e := vm.WithBridge(bridge, func() error {
		_, e := vm.Run(ctx, helperProgram, "helper.js")
		return e
	}); e != nil {
		return e
	}
	_, e := vm.Run(ctx, compiled.program, compiled.name+".js")
	return e
}

func createVm(ctx context.Context, config common.Config, script string) (*s.VM, error) {
	compiled, e := compileDriveScript(config, script)
	if e != nil {
		return nil, e
	}
	vm, e := s.NewVM()
	if e != nil {
		return nil, e
	}
	e = initializeDriveScriptVM(ctx, vm, compiled, nil)
	if e != nil {
		_ = vm.Dispose()
		return nil, e
	}
	return vm, nil
}

// wrapContentReader adapts an io.ReadCloser (already removed from the VM
// disposables, so the caller owns closing it) into an IContentReader.
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
