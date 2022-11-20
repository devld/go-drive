package script

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"go-drive/common"
	err "go-drive/common/errors"
	"go-drive/common/utils"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type driveRepositoryListResp struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
}

type AvailableDriveScript struct {
	Name             string `json:"name"`
	DriveURL         string `json:"driveUrl"`
	DriveUploaderURL string `json:"driveUploaderUrl,omitempty"`
}

func ListAvailableScriptsFromRepository(ctx context.Context, repoURL string) ([]AvailableDriveScript, error) {
	req, e := http.NewRequestWithContext(ctx, "GET", repoURL, nil)
	if e != nil {
		return nil, e
	}
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		return nil, e
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, err.NewRemoteApiError(resp.StatusCode, "failed to fetch data")
	}
	respData, e := io.ReadAll(resp.Body)
	if e != nil {
		return nil, e
	}
	items := make([]driveRepositoryListResp, 0)
	if e := json.Unmarshal(respData, &items); e != nil {
		return nil, e
	}
	itemsMap := utils.ArrayKeyBy(items, func(t driveRepositoryListResp, _ int) string { return t.Name })
	result := make([]AvailableDriveScript, 0)
	for _, item := range items {
		if !strings.HasSuffix(item.Name, ".js") || strings.HasSuffix(item.Name, "-uploader.js") {
			continue
		}
		name := strings.TrimRight(item.Name, ".js")
		resultItem := AvailableDriveScript{
			Name:     name,
			DriveURL: item.DownloadURL,
		}
		if uploaderItem, ok := itemsMap[name+"-uploader.js"]; ok {
			resultItem.DriveUploaderURL = uploaderItem.DownloadURL
		}
		result = append(result, resultItem)
	}
	return result, nil
}

type DriveScript struct {
	// Name is the script name without `.js`` suffix
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

func InstallDriveScript(ctx context.Context, config common.Config, s AvailableDriveScript) error {
	if s.Name == "" || s.DriveURL == "" {
		return err.NewBadRequestError("invalid installation request")
	}

	drivesDir, e := config.GetDir(config.DrivesDir, true)
	if e != nil {
		return e
	}
	e = downloadFile(ctx, s.DriveURL, filepath.Join(drivesDir, s.Name+".js"))
	if e != nil {
		return e
	}
	if s.DriveUploaderURL != "" {
		driveUploadersDir, e := config.GetDir(config.DriveUploadersDir, true)
		if e != nil {
			return e
		}
		e = downloadFile(ctx, s.DriveUploaderURL, filepath.Join(driveUploadersDir, s.Name+".js"))
		if e != nil {
			return e
		}
	}
	return nil
}

func UninstallDriveScript(config common.Config, name string) error {
	if name == "" {
		return err.NewBadRequestError("")
	}

	drivesDir, _ := config.GetDir(config.DrivesDir, false)
	driveUploadersDir, _ := config.GetDir(config.DriveUploadersDir, true)
	driveFile := filepath.Join(drivesDir, name+".js")
	driveUploaderFile := filepath.Join(driveUploadersDir, name+".js")
	if exists, _ := utils.FileExists(driveFile); exists {
		e := os.Remove(driveFile)
		if e != nil {
			return e
		}
	} else {
		return err.NewNotFoundError()
	}
	if exists, _ := utils.FileExists(driveUploaderFile); exists {
		e := os.Remove(driveUploaderFile)
		if e != nil {
			return e
		}
	}
	return nil
}

func downloadFile(ctx context.Context, url string, name string) error {
	req, e := http.NewRequestWithContext(ctx, "GET", url, nil)
	if e != nil {
		return e
	}
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		return e
	}
	defer func() { _ = resp.Body.Close() }()
	f, e := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if e != nil {
		return e
	}
	defer func() { _ = f.Close() }()
	_, e = io.Copy(f, resp.Body)
	return e
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

		scriptFile, e := os.Open(filepath.Join(scriptsPath, entry.Name()))
		if e != nil {
			continue
		}
		r := bufio.NewReader(scriptFile)
		name := readMetaValue(r, true, entry.Name())
		description := readMetaValue(r, false, "")
		_ = scriptFile.Close()

		result = append(result, DriveScript{
			Name:        strings.TrimRight(entry.Name(), ".js"),
			DisplayName: name,
			Description: description,
		})
	}

	return result, nil
}

var metaPrefixRegexp = regexp.MustCompile(`^\s*//\s*`)

func readMetaValue(r *bufio.Reader, oneLine bool, def string) string {
	sb := strings.Builder{}
	for {
		line, e := r.ReadBytes('\n')
		if e != nil {
			break
		}
		if !bytes.HasPrefix(line, []byte("//")) {
			break
		}

		temp := strings.TrimSpace(string(metaPrefixRegexp.ReplaceAll(line, []byte{})))
		sb.WriteString(temp)

		if oneLine {
			break
		}

		if len(bytes.TrimSpace(line)) == 0 {
			break
		}

		sb.WriteRune('\n')
	}
	if sb.Len() == 0 {
		return def
	}
	return sb.String()
}
