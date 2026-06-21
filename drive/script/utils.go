package script

import (
	"context"
	_ "embed"
	"errors"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/types"
	"go-drive/common/utils"
	s "go-drive/script"
	"io"
	"os"
	"path/filepath"
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

	_, e = vm.Run(context.Background(), helperScript)
	if e != nil {
		panic(e)
	}

	baseVM = vm
}

var t = i18n.TPrefix("drive.script.")

func newScriptDrive(ctx context.Context, config types.SM, driveUtils drive_util.DriveUtils) (types.IDrive, error) {
	cfg, e := driveUtils.Data.Load("_script")
	if e != nil {
		return nil, e
	}

	if cfg["_script"] == "" {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.not_configured"))
	}

	poolConfig, e := parsePoolConfig(config["pool"])
	if e != nil {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.script.invalid_pool_config", e.Error()))
	}

	vm, e := createVm(ctx, driveUtils.Config, cfg["_script"])
	if e != nil {
		return nil, e
	}

	d := &ScriptDrive{
		baseVM: vm,
		data:   make(map[string]*s.Value),
	}

	vm.Set("setData", s.WrapVmCall(vm, d.setData))
	vm.Set("getData", s.WrapVmCall(vm, d.getData))

	_, e = vm.Call(ctx, "__driveCreate", s.NewContext(vm, ctx), config, newScriptDriveUtils(vm, driveUtils))

	if e != nil {
		_ = d.Dispose()
		return nil, e
	}
	vm.Set("selfDrive", s.NewDrive(vm, d))
	d.pool = s.NewVMPool(vm, poolConfig)

	return d, nil
}

func initConfig(ctx context.Context, config types.SM, driveUtils drive_util.DriveUtils) (*drive_util.DriveInitConfig, error) {
	selectedScript := config["script"]
	if selectedScript == "" {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.not_configured"))
	}
	selectedScript += ".js"

	cfg, e := driveUtils.Data.Load("_script")
	if e != nil {
		return nil, e
	}
	if cfg["_script"] != selectedScript {
		if e := driveUtils.Data.Clear(); e != nil {
			return nil, e
		}
	}
	if e := driveUtils.Data.Save(types.SM{"_script": selectedScript}); e != nil {
		return nil, e
	}

	initForm := make([]types.FormItem, 0, 1)
	values := make(types.SM)

	ds, e := readDriveScriptMeta(selectedScript, driveUtils.Config)
	if e != nil {
		return nil, e
	}
	initForm = append(initForm, types.FormItem{Type: "md", Description: ds.Description})

	retCfg := &drive_util.DriveInitConfig{
		Configured: false,
		Form:       initForm,
		Value:      values,
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
	if initConfigVal.IsNil() {
		return retCfg, nil
	}

	v, e := vm.Call(ctx, "__driveInitConfig", s.NewContext(vm, ctx), config, newScriptDriveUtils(vm, driveUtils))
	if e != nil {
		return nil, e
	}

	vmCfg := &drive_util.DriveInitConfig{}
	v.ParseInto(vmCfg)

	retCfg.Configured = vmCfg.Configured
	retCfg.OAuth = vmCfg.OAuth
	retCfg.Form = append(retCfg.Form, vmCfg.Form...)
	utils.MapCopy(vmCfg.Value, retCfg.Value)

	return retCfg, nil
}

func init_(ctx context.Context, data, config types.SM, driveUtils drive_util.DriveUtils) error {
	cfg, e := driveUtils.Data.Load("_script")
	if e != nil {
		return e
	}
	if cfg["_script"] != "" {
		// _script is not modifiable
		delete(data, "_script")
	} else if data["_script"] != "" {
		cfg["_script"] = data["_script"]
		if e := driveUtils.Data.Save(types.SM{"_script": data["_script"]}); e != nil {
			return e
		}
	}

	vm, e := createVm(ctx, driveUtils.Config, cfg["_script"])
	if e != nil {
		return e
	}
	defer func() { _ = vm.Dispose() }()

	initConfigVal, e := vm.GetValue("__driveInit")
	if e != nil {
		return e
	}
	if initConfigVal.IsNil() {
		return nil
	}

	_, e = vm.Call(ctx, "__driveInit", s.NewContext(vm, ctx), data, config, newScriptDriveUtils(vm, driveUtils))
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

func newScriptDriveUtils(vm *s.VM, utils drive_util.DriveUtils) *scriptDriveUtils {
	return &scriptDriveUtils{vm, utils.CreateCache, driveDataStore{vm, utils.Data}, utils.Config}
}

type scriptDriveUtils struct {
	vm *s.VM

	createCache drive_util.DriveCacheFactory

	Data   driveDataStore
	Config common.Config
}

func (sdu *scriptDriveUtils) CreateCache() *scriptDriveCache {
	return &scriptDriveCache{sdu.vm, sdu.createCache(nil)}
}

func (sdu *scriptDriveUtils) OAuthInitConfig(or drive_util.OAuthRequest,
	cred drive_util.OAuthCredentials) *oauthInitConfigResp {
	c, r, e := drive_util.OAuthInitConfig(or, cred, sdu.Data.data)
	if e != nil {
		sdu.vm.ThrowError(e)
	}
	var resp *oauthRespWrapper
	if r != nil {
		resp = &oauthRespWrapper{sdu.vm, r}
	}
	return &oauthInitConfigResp{c, resp}
}

func (sdu *scriptDriveUtils) OAuthInit(ctx s.Context,
	data types.SM, or drive_util.OAuthRequest,
	cred drive_util.OAuthCredentials) *oauthRespWrapper {
	resp, e := drive_util.OAuthInit(s.GetContext(ctx), or, data, cred, sdu.Data.data)
	if e != nil {
		sdu.vm.ThrowError(e)
	}
	var r *oauthRespWrapper
	if resp != nil {
		r = &oauthRespWrapper{sdu.vm, resp}
	}
	return r
}

func (sdu *scriptDriveUtils) OAuthGet(o drive_util.OAuthRequest,
	cred drive_util.OAuthCredentials) *oauthRespWrapper {
	resp, e := drive_util.OAuthGet(o, cred, sdu.Data.data)
	if e != nil {
		sdu.vm.ThrowError(e)
	}
	var r *oauthRespWrapper
	if resp != nil {
		r = &oauthRespWrapper{sdu.vm, resp}
	}
	return r
}

type driveDataStore struct {
	vm   *s.VM
	data drive_util.DriveDataStore
}

func (d driveDataStore) Save(s types.SM) {
	if e := d.data.Save(s); e != nil {
		d.vm.ThrowError(e)
	}
}

func (d driveDataStore) Load(keys ...string) types.SM {
	r, e := d.data.Load(keys...)
	if e != nil {
		d.vm.ThrowError(e)
	}
	return r
}

type oauthInitConfigResp struct {
	Config   *drive_util.DriveInitConfig
	Response *oauthRespWrapper
}

type oauthRespWrapper struct {
	vm   *s.VM
	resp *drive_util.OAuthResponse
}

func (or *oauthRespWrapper) Token() *oauth2.Token {
	t, e := or.resp.Token()
	if e != nil {
		or.vm.ThrowError(e)
	}
	return t
}

func createVm(ctx context.Context, config common.Config, script string) (*s.VM, error) {
	scriptsPath, _ := config.GetDir(config.DrivesDir, false)
	scriptBytes, e := os.ReadFile(filepath.Join(scriptsPath, script))
	if e != nil {
		return nil, e
	}

	vm := baseVM.Fork()

	_, e = vm.Run(ctx, scriptBytes)
	return vm, e
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
