package script

import (
	"bufio"
	"context"
	_ "embed"
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
	"regexp"
	"strings"

	"golang.org/x/oauth2"
)

//go:embed helper.js
var helperScript []byte
var baseVM *s.VM

func init() {
	vm, e := s.NewVM()
	if e != nil {
		panic(e)
	}

	_, e = vm.Run(helperScript)
	if e != nil {
		panic(e)
	}

	baseVM = vm
}

const scriptDir = "script-drives"

var t = i18n.TPrefix("drive.script.")

func newScriptDrive(ctx context.Context, config types.SM, driveUtils drive_util.DriveUtils) (types.IDrive, error) {
	cfg, e := driveUtils.Data.Load("_script")
	if e != nil {
		return nil, e
	}

	if cfg["_script"] == "" {
		return nil, err.NewNotAllowedMessageError(i18n.T("drive.not_configured"))
	}

	vm, e := createVm(driveUtils.Config, cfg["_script"])
	if e != nil {
		return nil, e
	}

	d := &ScriptDrive{
		vm: vm,
	}

	_, e = vm.Call("__driveCreate", s.NewContext(vm, ctx), config, newScriptDriveUtils(vm, driveUtils))

	if e != nil {
		_ = d.Dispose()
		return nil, e
	}

	return d, nil
}

func initConfig(ctx context.Context, config types.SM, driveUtils drive_util.DriveUtils) (*drive_util.DriveInitConfig, error) {
	cfg, e := driveUtils.Data.Load("_script")
	if e != nil {
		return nil, e
	}

	scriptsPath, _ := driveUtils.Config.GetDir(scriptDir, false)
	scripts, _ := readAvailableScripts(scriptsPath)

	initForm := make([]types.FormItem, 1)
	initForm[0] = types.FormItem{
		Field:       "_script",
		Label:       t("form.script.label"),
		Description: t("form.script.description"),
		Type:        "select",
		Options:     &scripts,
		Required:    true,
		Disabled:    cfg["_script"] != "",
	}
	values := make(types.SM, 1)
	values["_script"] = cfg["_script"]

	retCfg := &drive_util.DriveInitConfig{
		Configured: false,
		Form:       initForm,
		Value:      values,
	}

	if cfg["_script"] == "" {
		return retCfg, nil
	}

	vm, e := createVm(driveUtils.Config, cfg["_script"])
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

	v, e := vm.Call("__driveInitConfig", s.NewContext(vm, ctx), config, newScriptDriveUtils(vm, driveUtils))
	if e != nil {
		return nil, e
	}

	vmCfg := &drive_util.DriveInitConfig{}
	v.ParseInto(vmCfg)

	retCfg.Configured = vmCfg.Configured
	retCfg.OAuth = vmCfg.OAuth
	retCfg.Form = append(retCfg.Form, vmCfg.Form...)
	utils.CopyMap(vmCfg.Value, retCfg.Value)

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

	vm, e := createVm(driveUtils.Config, cfg["_script"])
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

	_, e = vm.Call("__driveInit", s.NewContext(vm, ctx), data, config, newScriptDriveUtils(vm, driveUtils))
	return e
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

func createVm(config common.Config, script string) (*s.VM, error) {
	scriptsPath, _ := config.GetDir(scriptDir, false)
	scriptBytes, e := os.ReadFile(filepath.Join(scriptsPath, script))
	if e != nil {
		return nil, e
	}

	vm := baseVM.Fork()

	_, e = vm.Run(scriptBytes)
	return vm, e
}

func readAvailableScripts(p string) ([]types.FormItemOption, error) {
	entries, e := os.ReadDir(p)
	if e != nil {
		return []types.FormItemOption{}, e
	}
	result := make([]types.FormItemOption, 0)

	for _, entry := range entries {
		n := strings.ToLower(entry.Name())
		if !strings.HasSuffix(n, ".js") {
			continue
		}

		scriptFile, e := os.Open(filepath.Join(p, entry.Name()))
		if e != nil {
			continue
		}
		r := bufio.NewReader(scriptFile)
		name := readMetaValue(r, entry.Name())
		description := readMetaValue(r, "")
		_ = scriptFile.Close()

		result = append(result, types.FormItemOption{
			Name:  name,
			Value: entry.Name(),
			Title: description,
		})
	}

	return result, nil
}

var metaPrefixRegexp = regexp.MustCompile(`^\s*//\s*`)

func readMetaValue(r *bufio.Reader, def string) string {
	line, e := r.ReadBytes('\n')
	if e != nil {
		return def
	}
	temp := strings.TrimSpace(string(metaPrefixRegexp.ReplaceAll(line, []byte{})))
	if temp != "" {
		return temp
	}
	return def
}

func wrapReader(reader io.Reader) io.ReadCloser {
	var r io.ReadCloser
	if rc, ok := reader.(io.ReadCloser); ok {
		r = rc
	} else {
		r = fakeCloseReader{reader}
	}
	return r
}

type fakeCloseReader struct {
	io.Reader
}

func (fcr fakeCloseReader) Close() error {
	return nil
}

func wrapContentURL(reader io.Reader) types.IContentReader {
	return fakeContentReader{reader}
}

type fakeContentReader struct {
	reader io.Reader
}

func (fcr fakeContentReader) GetReader(ctx context.Context) (io.ReadCloser, error) {
	return wrapReader(fcr.reader), nil
}

func (fcr fakeContentReader) GetURL(_ context.Context) (*types.ContentURL, error) {
	return nil, err.NewUnsupportedError()
}
