package script

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/logging"
	httpreq "go-drive/common/req"
	"go-drive/common/task"
	"go-drive/common/types"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	maxRepositoryResponseSize int64 = 1 << 20
	maxScriptSize             int64 = 1 << 20

	availableScriptsDir = ".repo"
	repositoryIndexFile = "index.json"
)

type driveRepositoryListResp struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
}

type AvailableDriveScript struct {
	Name             string `json:"name"`
	DisplayName      string `json:"displayName,omitempty"`
	Description      string `json:"description,omitempty"`
	Version          string `json:"version,omitempty"`
	DriveURL         string `json:"driveUrl"`
	DriveUploaderURL string `json:"driveUploaderUrl,omitempty"`
}

type DriveScriptRepository struct {
	Scripts []AvailableDriveScript `json:"scripts"`
	Ready   bool                   `json:"ready"`
}

type DriveScriptList struct {
	Scripts []DriveScriptListItem `json:"scripts"`
	Ready   bool                  `json:"ready"`
}

type DriveScriptListItem struct {
	AvailableDriveScript
	Installed       *DriveScript `json:"installed"`
	UpdateAvailable bool         `json:"updateAvailable"`
}

type repositoryCache struct {
	Scripts []repositoryCachedScript `json:"scripts"`
}

type repositoryCachedScript struct {
	AvailableDriveScript
	DriveFile    string `json:"driveFile"`
	UploaderFile string `json:"uploaderFile,omitempty"`
}

type DriveScript struct {
	// Name is the script name without `.js`` suffix
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type DriveScriptContent struct {
	Drive    string `json:"drive"`
	Uploader string `json:"uploader,omitempty"`
}

func LoadDriveScriptRepository(config common.Config) (DriveScriptRepository, error) {
	cache, ready, e := loadRepositoryCache(config)
	if e != nil {
		return DriveScriptRepository{}, e
	}
	if !ready {
		return DriveScriptRepository{Scripts: []AvailableDriveScript{}, Ready: false}, nil
	}
	scripts := make([]AvailableDriveScript, 0, len(cache.Scripts))
	for _, item := range cache.Scripts {
		scripts = append(scripts, item.AvailableDriveScript)
	}
	return DriveScriptRepository{Scripts: scripts, Ready: true}, nil
}

func ListAllDriveScripts(config common.Config) (DriveScriptList, error) {
	repo, e := LoadDriveScriptRepository(config)
	if e != nil {
		return DriveScriptList{}, e
	}
	installed, e := ListDriveScripts(config)
	if e != nil {
		return DriveScriptList{}, e
	}

	availableByName := make(map[string]AvailableDriveScript, len(repo.Scripts))
	for _, item := range repo.Scripts {
		availableByName[item.Name] = item
	}

	scripts := make([]DriveScriptListItem, 0, len(installed)+len(repo.Scripts))
	seen := make(map[string]struct{}, len(installed)+len(repo.Scripts))
	for _, local := range installed {
		local := local
		item := DriveScriptListItem{Installed: &local}
		if available, ok := availableByName[local.Name]; ok {
			item.AvailableDriveScript = available
		} else {
			item.AvailableDriveScript = AvailableDriveScript{
				Name:        local.Name,
				DisplayName: local.DisplayName,
				Description: local.Description,
				Version:     local.Version,
			}
		}
		item.UpdateAvailable = driveScriptUpdateAvailable(item.Version, local.Version)
		scripts = append(scripts, item)
		seen[local.Name] = struct{}{}
	}
	for _, available := range repo.Scripts {
		if _, ok := seen[available.Name]; ok {
			continue
		}
		scripts = append(scripts, DriveScriptListItem{AvailableDriveScript: available})
	}
	return DriveScriptList{Scripts: scripts, Ready: repo.Ready}, nil
}

func driveScriptUpdateAvailable(repositoryVersion, installedVersion string) bool {
	return repositoryVersion != "" && installedVersion != "" && repositoryVersion != installedVersion
}

func SyncDriveScriptsFromRepository(ctx types.TaskCtx, config common.Config, repoURL string) (DriveScriptRepository, error) {
	started := time.Now()
	log := logging.For("scr-drv")
	if ctx == nil {
		ctx = task.DummyContext()
	}
	if e := ctx.Err(); e != nil {
		return DriveScriptRepository{}, e
	}

	_, filesDir, indexPath, e := repositoryCachePaths(config, true)
	if e != nil {
		return DriveScriptRepository{}, e
	}

	ctx.Total(1, true)
	ctx.Progress(0, true)

	resp, e := httpreq.NewDefaultClient().Get(ctx, repoURL, nil)
	if e != nil {
		return DriveScriptRepository{}, e
	}
	defer func() { _ = resp.Dispose() }()
	if resp.Status() != http.StatusOK {
		return DriveScriptRepository{}, err.NewRemoteApiError(resp.Status(), "failed to fetch data")
	}
	respData, e := readLimitedContent(resp.Response().Body, maxRepositoryResponseSize, "script repository response is too large")
	if e != nil {
		return DriveScriptRepository{}, e
	}
	items := make([]driveRepositoryListResp, 0)
	if e := json.Unmarshal(respData, &items); e != nil {
		return DriveScriptRepository{}, e
	}
	ctx.Progress(1, true)

	itemsMap := make(map[string]driveRepositoryListResp, len(items))
	jsItems := make([]driveRepositoryListResp, 0)
	invalidItems := 0
	for _, item := range items {
		name, ok := repositoryFileName(item.Name)
		if !ok || item.DownloadURL == "" {
			invalidItems++
			log.Warnf("ignored invalid script repository item name=%s", logging.Sanitize(item.Name))
			continue
		}
		item.Name = name
		itemsMap[name] = item
		jsItems = append(jsItems, item)
	}

	ctx.Total(int64(len(jsItems)), false)
	contents := make(map[string][]byte, len(jsItems))
	failedDownloads := 0
	for _, item := range jsItems {
		if e := ctx.Err(); e != nil {
			return DriveScriptRepository{}, e
		}
		content, e := syncRepositoryFile(ctx, filesDir, item)
		ctx.Progress(1, false)
		if e != nil {
			failedDownloads++
			log.Warnf("script repository file download failed file=%s: %v", logging.Sanitize(item.Name), e)
			continue
		}
		contents[item.Name] = content
	}

	cached := make([]repositoryCachedScript, 0, len(jsItems))
	keepFiles := make(map[string]struct{}, len(jsItems))
	invalidScripts := 0
	for _, item := range jsItems {
		content, ok := contents[item.Name]
		if !ok {
			continue
		}
		meta, ok, e := parseDriveScriptMeta(content, item.Name)
		if e != nil || !ok {
			invalidScripts++
			log.Warnf("ignored invalid script repository file=%s: %v", logging.Sanitize(item.Name), e)
			continue
		}

		resultItem := repositoryCachedScript{
			AvailableDriveScript: AvailableDriveScript{
				Name:        meta.Name,
				DisplayName: meta.DisplayName,
				Description: meta.Description,
				Version:     meta.Version,
				DriveURL:    item.DownloadURL,
			},
			DriveFile: item.Name,
		}
		keepFiles[item.Name] = struct{}{}

		if meta.Uploader != "" {
			if uploaderItem, ok := itemsMap[meta.Uploader]; ok {
				if _, downloaded := contents[uploaderItem.Name]; downloaded {
					resultItem.DriveUploaderURL = uploaderItem.DownloadURL
					resultItem.UploaderFile = uploaderItem.Name
					keepFiles[uploaderItem.Name] = struct{}{}
				}
			}
		}
		cached = append(cached, resultItem)
	}

	if e := removeUnusedRepositoryFiles(filesDir, keepFiles); e != nil {
		return DriveScriptRepository{}, e
	}
	if e := saveRepositoryCache(indexPath, repositoryCache{Scripts: cached}); e != nil {
		return DriveScriptRepository{}, e
	}
	log.Debugf("script repository sync completed files=%d scripts=%d invalid_items=%d failed_downloads=%d invalid_scripts=%d duration=%s",
		len(jsItems), len(cached), invalidItems, failedDownloads, invalidScripts, time.Since(started))

	scripts := make([]AvailableDriveScript, 0, len(cached))
	for _, item := range cached {
		scripts = append(scripts, item.AvailableDriveScript)
	}
	return DriveScriptRepository{Scripts: scripts, Ready: true}, nil
}

func GetDriveScript(config common.Config, name string) (DriveScriptContent, error) {
	file, e := driveScriptFile(name)
	if e != nil {
		return DriveScriptContent{}, e
	}

	drivesDir, e := config.GetDir(config.DrivesDir, false)
	if e != nil {
		return DriveScriptContent{}, err.NewNotFoundError()
	}

	r := DriveScriptContent{}
	driveBytes, e := readFileInDir(drivesDir, file)
	if e != nil {
		if os.IsNotExist(e) {
			return r, err.NewNotFoundError()
		}
		return r, e
	}
	r.Drive = string(driveBytes)

	driveUploadersDir, e := config.GetDir(config.DriveUploadersDir, false)
	if e != nil {
		return r, nil
	}
	uploaderBytes, e := readFileInDir(driveUploadersDir, file)
	if e != nil {
		if os.IsNotExist(e) {
			return r, nil
		}
		return r, e
	}
	r.Uploader = string(uploaderBytes)
	return r, nil
}

func SaveDriveScript(config common.Config, name string, content DriveScriptContent) error {
	file, e := driveScriptFile(name)
	if e != nil {
		return e
	}
	if content.Drive != "" {
		drivesDir, e := config.GetDir(config.DrivesDir, true)
		if e != nil {
			return e
		}
		if e = writeFileAtomically(drivesDir, file, []byte(content.Drive)); e != nil {
			return e
		}
	}

	if content.Uploader != "" {
		driveUploadersDir, e := config.GetDir(config.DriveUploadersDir, true)
		if e != nil {
			return e
		}
		if e = writeFileAtomically(driveUploadersDir, file, []byte(content.Uploader)); e != nil {
			return e
		}
	}

	return nil
}

func InstallDriveScript(config common.Config, name string) error {
	file, e := driveScriptFile(name)
	if e != nil {
		return e
	}
	cache, ready, e := loadRepositoryCache(config)
	if e != nil {
		return e
	}
	if !ready {
		return err.NewNotFoundError()
	}
	var cached *repositoryCachedScript
	for i := range cache.Scripts {
		if cache.Scripts[i].Name == name {
			cached = &cache.Scripts[i]
			break
		}
	}
	if cached == nil || cached.DriveFile == "" {
		return err.NewNotFoundError()
	}

	driveFile, ok := repositoryFileName(cached.DriveFile)
	if !ok {
		return err.NewNotFoundError()
	}
	destFile, e := driveScriptFile(cached.Name)
	if e != nil || destFile != file {
		return err.NewNotFoundError()
	}

	_, filesDir, _, e := repositoryCachePaths(config, false)
	if e != nil {
		return e
	}

	drivesDir, e := config.GetDir(config.DrivesDir, true)
	if e != nil {
		return e
	}
	if exists, _ := fileExistsInDir(filesDir, driveFile); !exists {
		return err.NewNotFoundError()
	}

	var uploaderFile string
	var driveUploadersDir string
	if cached.UploaderFile != "" {
		uploaderFile, ok = repositoryFileName(cached.UploaderFile)
		if !ok {
			return err.NewNotFoundError()
		}
		if exists, _ := fileExistsInDir(filesDir, uploaderFile); !exists {
			return err.NewNotFoundError()
		}
		driveUploadersDir, e = config.GetDir(config.DriveUploadersDir, true)
		if e != nil {
			return e
		}
	}

	if e = copyFileInDir(filesDir, driveFile, drivesDir, destFile); e != nil {
		return e
	}
	if driveUploadersDir != "" {
		return copyFileInDir(filesDir, uploaderFile, driveUploadersDir, destFile)
	}

	driveUploadersDir, e = config.GetDir(config.DriveUploadersDir, false)
	if e != nil {
		return nil
	}
	if exists, _ := fileExistsInDir(driveUploadersDir, destFile); exists {
		_ = removeFileInDir(driveUploadersDir, destFile)
	}
	return nil
}

func UninstallDriveScript(config common.Config, name string) error {
	file, e := driveScriptFile(name)
	if e != nil {
		return e
	}

	drivesDir, e := config.GetDir(config.DrivesDir, false)
	if e != nil {
		return err.NewNotFoundError()
	}
	if e := removeFileInDir(drivesDir, file); e != nil {
		if os.IsNotExist(e) {
			return err.NewNotFoundError()
		}
		return e
	}

	driveUploadersDir, e := config.GetDir(config.DriveUploadersDir, false)
	if e != nil {
		return nil
	}
	if e := removeFileInDir(driveUploadersDir, file); e != nil && !os.IsNotExist(e) {
		return e
	}
	return nil
}

func ListDriveScripts(config common.Config) ([]DriveScript, error) {
	scriptsPath, _ := config.GetDir(config.DrivesDir, false)
	entries, e := os.ReadDir(scriptsPath)
	if e != nil {
		return []DriveScript{}, nil
	}
	result := make([]DriveScript, 0)
	for _, entry := range entries {
		n := strings.ToLower(entry.Name())
		if !strings.HasSuffix(n, ".js") {
			continue
		}
		ds, e := readDriveScriptMeta(entry.Name(), config)
		if e != nil {
			continue
		}
		result = append(result, ds)
	}
	return result, nil
}

func driveScriptFile(name string) (string, error) {
	if name == "" {
		return "", err.NewBadRequestError("")
	}
	return scriptFileName(name)
}

func readDriveScriptFile(file string, config common.Config) ([]byte, error) {
	scriptsPath, e := config.GetDir(config.DrivesDir, false)
	if e != nil {
		return nil, e
	}
	return readFileInDir(scriptsPath, file)
}

func readFileInDir(dir, name string) ([]byte, error) {
	root, e := os.OpenRoot(dir)
	if e != nil {
		return nil, e
	}
	defer func() { _ = root.Close() }()
	return root.ReadFile(name)
}

func fileExistsInDir(dir, name string) (bool, error) {
	root, e := os.OpenRoot(dir)
	if e != nil {
		if os.IsNotExist(e) {
			return false, nil
		}
		return false, e
	}
	defer func() { _ = root.Close() }()
	_, e = root.Stat(name)
	if e != nil {
		if os.IsNotExist(e) {
			return false, nil
		}
		return false, e
	}
	return true, nil
}

func removeFileInDir(dir, name string) error {
	root, e := os.OpenRoot(dir)
	if e != nil {
		return e
	}
	defer func() { _ = root.Close() }()
	return root.Remove(name)
}

func readDriveScriptMeta(file string, config common.Config) (DriveScript, error) {
	content, e := readDriveScriptFile(file, config)
	if e != nil {
		return DriveScript{}, e
	}
	meta, ok, e := parseDriveScriptMeta(content, file)
	if e != nil {
		return DriveScript{}, e
	}
	if !ok {
		return DriveScript{}, errInvalidDriveScriptMeta
	}
	return meta.DriveScript, nil
}

var (
	errInvalidDriveScriptMeta = errors.New("invalid drive script metadata")
	metaPrefixRegexp          = regexp.MustCompile(`^\s*//\s?`)
	metaDirectiveRegexp       = regexp.MustCompile(`(?i)^@([a-z][a-z0-9_-]*)(?:\s+(.*))?$`)
)

type parsedDriveScriptMeta struct {
	DriveScript
	Uploader string
}

func parseDriveScriptMeta(content []byte, file string) (parsedDriveScriptMeta, bool, error) {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lines := make([]string, 0, 8)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "//") {
			break
		}
		lines = append(lines, strings.TrimSpace(metaPrefixRegexp.ReplaceAllString(line, "")))
	}
	if e := scanner.Err(); e != nil {
		return parsedDriveScriptMeta{}, false, e
	}

	name := strings.TrimSuffix(filepath.Base(file), ".js")
	meta := parsedDriveScriptMeta{DriveScript: DriveScript{Name: name}}
	description := make([]string, 0, len(lines))
	inDescription := false
	for _, line := range lines {
		if match := metaDirectiveRegexp.FindStringSubmatch(line); match != nil {
			key := strings.ToLower(match[1])
			value := strings.TrimSpace(match[2])
			switch key {
			case "name":
				meta.DisplayName = value
			case "version":
				meta.Version = value
			case "uploader":
				if uploader := repositoryUploaderName(value); uploader != "" {
					meta.Uploader = uploader
				}
			case "description":
				inDescription = true
				if value != "" {
					description = append(description, value)
				}
			}
			continue
		}
		if inDescription {
			description = append(description, line)
		}
	}
	meta.Description = strings.TrimSpace(strings.Join(description, "\n"))
	return meta, meta.DisplayName != "" && meta.Version != "", nil
}

func repositoryCachePaths(config common.Config, create bool) (cacheDir, filesDir, indexPath string, e error) {
	drivesDir, e := config.GetDir(config.DrivesDir, create)
	if e != nil {
		return "", "", "", e
	}
	cacheDir = filepath.Join(drivesDir, availableScriptsDir)
	if create {
		if e = os.MkdirAll(cacheDir, 0755); e != nil {
			return "", "", "", e
		}
	}
	return cacheDir, cacheDir, filepath.Join(cacheDir, repositoryIndexFile), nil
}

func loadRepositoryCache(config common.Config) (repositoryCache, bool, error) {
	_, _, indexPath, e := repositoryCachePaths(config, false)
	if e != nil {
		return repositoryCache{}, false, e
	}
	data, e := os.ReadFile(indexPath)
	if e != nil {
		if os.IsNotExist(e) {
			return repositoryCache{}, false, nil
		}
		return repositoryCache{}, false, e
	}
	cache := repositoryCache{}
	if e := json.Unmarshal(data, &cache); e != nil {
		return repositoryCache{}, false, nil
	}
	if cache.Scripts == nil {
		cache.Scripts = []repositoryCachedScript{}
	}
	return cache, true, nil
}

func saveRepositoryCache(indexPath string, cache repositoryCache) error {
	data, e := json.Marshal(cache)
	if e != nil {
		return e
	}
	return writeFileAtomically(filepath.Dir(indexPath), filepath.Base(indexPath), data)
}

func syncRepositoryFile(ctx context.Context, filesDir string, item driveRepositoryListResp) ([]byte, error) {
	content, e := downloadScriptContent(ctx, item.DownloadURL)
	if e != nil {
		return nil, e
	}
	if e := writeFileAtomically(filesDir, item.Name, content); e != nil {
		return nil, e
	}
	return content, nil
}

func removeUnusedRepositoryFiles(filesDir string, keep map[string]struct{}) error {
	entries, e := os.ReadDir(filesDir)
	if e != nil {
		if os.IsNotExist(e) {
			return nil
		}
		return e
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == repositoryIndexFile {
			continue
		}
		if _, ok := keep[entry.Name()]; ok {
			continue
		}
		if e := os.Remove(filepath.Join(filesDir, entry.Name())); e != nil {
			logging.For("scr-drv").Warnf("failed to remove stale repository file file=%s: %v",
				logging.Sanitize(entry.Name()), e)
		}
	}
	return nil
}

func repositoryFileName(name string) (string, bool) {
	cleaned := strings.ReplaceAll(strings.TrimSpace(name), "\\", "/")
	base := filepath.Base(cleaned)
	if base == "" || base == "." || base == ".." || strings.ContainsAny(base, `/\`) {
		return "", false
	}
	if !strings.HasSuffix(strings.ToLower(base), ".js") {
		return "", false
	}
	return base, true
}

func repositoryUploaderName(value string) string {
	name, ok := repositoryFileName(value)
	if ok {
		return name
	}
	cleaned := strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	base := filepath.Base(cleaned)
	if base == "" || base == "." || base == ".." || strings.ContainsAny(base, `/\`) {
		return ""
	}
	return base + ".js"
}

func openScriptResponse(ctx context.Context, url string) (httpreq.Response, error) {
	resp, e := httpreq.NewDefaultClient().Get(ctx, url, nil)
	if e != nil {
		return nil, e
	}
	if resp.Status() < http.StatusOK || resp.Status() >= http.StatusMultipleChoices {
		_ = resp.Dispose()
		return nil, err.NewRemoteApiError(resp.Status(), "failed to download script")
	}
	return resp, nil
}

func downloadScriptContent(ctx context.Context, url string) ([]byte, error) {
	resp, e := openScriptResponse(ctx, url)
	if e != nil {
		return nil, e
	}
	defer func() { _ = resp.Dispose() }()
	return readLimitedContent(resp.Response().Body, maxScriptSize, "script is too large")
}

func readLimitedContent(reader io.Reader, maxSize int64, message string) ([]byte, error) {
	content, e := io.ReadAll(io.LimitReader(reader, maxSize+1))
	if e != nil {
		return nil, e
	}
	if int64(len(content)) > maxSize {
		return nil, err.NewRemoteApiError(http.StatusRequestEntityTooLarge, message)
	}
	return content, nil
}

func writeFileAtomically(dir, name string, content []byte) error {
	if e := os.MkdirAll(dir, 0755); e != nil {
		return e
	}
	root, e := os.OpenRoot(dir)
	if e != nil {
		return e
	}
	defer func() { _ = root.Close() }()

	tempFile, e := os.CreateTemp(dir, ".script-*")
	if e != nil {
		return e
	}
	tempName := filepath.Base(tempFile.Name())
	cleanup := func() {
		_ = tempFile.Close()
		_ = root.Remove(tempName)
	}
	if _, e = tempFile.Write(content); e != nil {
		cleanup()
		return e
	}
	if e = tempFile.Chmod(0644); e != nil {
		cleanup()
		return e
	}
	if e = tempFile.Close(); e != nil {
		_ = root.Remove(tempName)
		return e
	}
	_ = root.Remove(name)
	if e = root.Rename(tempName, name); e != nil {
		_ = root.Remove(tempName)
		return e
	}
	return nil
}

func copyFileInDir(srcDir, srcName, destDir, destName string) error {
	content, e := readFileInDir(srcDir, srcName)
	if e != nil {
		return e
	}
	return writeFileAtomically(destDir, destName, content)
}
