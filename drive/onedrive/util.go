package onedrive

import (
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/task"
	"go-drive/common/types"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var httpApi, _ = common.NewHttpClient("", nil, ifApiCallError, nil)

func pathURL(path string) string {
	if common.IsRootPath(path) {
		return "/root"
	}
	return common.BuildURL("/root:/{}:", path)
}

func idURL(id string) string {
	return common.BuildURL("/items/{}", id)
}

func itemPath(path string) string {
	if common.IsRootPath(path) {
		return "/drive/root:"
	}
	return common.BuildURL("/drive/root:/{}", path)
}

func InitConfig(config drive_util.DriveConfig, utils drive_util.DriveUtils) (drive_util.DriveInitConfig, error) {
	clientId := config["client_id"]
	state := common.RandString(5)
	if e := utils.Data.Save(types.SM{"state": state}); e != nil {
		return drive_util.DriveInitConfig{}, e
	}
	oauthUrl, _ := url.Parse("https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize?" +
		"response_type=code&response_mode=query")
	q := oauthUrl.Query()
	q.Set("client_id", clientId)
	q.Set("scope", scope)
	q.Set("state", state)
	q.Set("redirect_uri", redirectUri)
	oauthUrl.RawQuery = q.Encode()

	params, e := utils.Data.Load("token", "expires_at", "drive_id", "refresh_token")
	if e != nil {
		return drive_util.DriveInitConfig{}, e
	}
	expiresAt := time.Unix(common.ToInt64(params["expires_at"], -1), 0)

	configured := !expiresAt.Before(time.Now().Add(-30 * 24 * time.Hour))
	principal := ""

	if configured && expiresAt.Before(time.Now()) {
		// token expired
		r, e := getToken(true, params["refresh_token"], clientId, config["client_secret"], utils.Data)
		configured = e == nil
		if e == nil {
			params["token"] = r.AccessToken
		}
	}

	if configured {
		user, e := getUser(params["token"])
		configured = e == nil
		if e == nil {
			principal = fmt.Sprintf("%s <%s>", user.DisplayName, user.UserPrincipalName)
		}
	}

	var form []types.FormItem = nil
	var formValue types.SM = nil

	if configured {
		drives, e := getDrives(params["token"])
		configured = e == nil
		if e == nil {
			opts := make([]types.FormItemOption, len(drives))
			for i, d := range drives {
				used := "-"
				if d.Quota.Total != 0 {
					used = fmt.Sprintf("%.1f%%", float64(d.Quota.Used)/float64(d.Quota.Total)*100)
				}
				opts[i] = types.FormItemOption{
					Name: fmt.Sprintf("%s %d", d.DriveType, i+1),
					Title: fmt.Sprintf(
						"%s / %s | %s used",
						common.FormatBytes(uint64(d.Quota.Used), 1),
						common.FormatBytes(uint64(d.Quota.Total), 1),
						used,
					),
					Value: d.Id,
				}
			}
			form = []types.FormItem{
				{Label: "Drive", Type: "select", Field: "drive_id", Required: true, Options: opts},
			}
			formValue = types.SM{"drive_id": params["drive_id"]}
		}
	}

	if configured {
		configured = params["drive_id"] != ""
	}

	return drive_util.DriveInitConfig{
		Configured: configured,
		OAuth: &drive_util.OAuthInitConfig{
			Url:       oauthUrl.String(),
			Text:      "Connect to OneDrive",
			Principal: principal,
		},
		Form:  form,
		Value: formValue,
	}, nil
}

func Init(data types.SM, config drive_util.DriveConfig, utils drive_util.DriveUtils) error {
	code := data["code"]
	state := data["state"]
	driveId := data["drive_id"]

	params, e := utils.Data.Load("state", "drive_id")
	if e != nil {
		return e
	}
	if code != "" {
		if state != params["state"] {
			return common.NewNotAllowedMessageError("state does not match")
		}
		_, e = getToken(false, code, config["client_id"], config["client_secret"], utils.Data)
		return e
	}
	if driveId != "" {
		return utils.Data.Save(types.SM{"drive_id": driveId})
	}

	return common.NewBadRequestError("bad request")
}

func getToken(refresh bool, codeOrRefreshToken, clientId, clientSecret string, ds drive_util.DriveDataStore) (getTokenResp, error) {
	key := "code"
	grantType := "authorization_code"
	if refresh {
		key = "refresh_token"
		grantType = "refresh_token"
	}

	resp, e := httpApi.Post(
		"https://login.microsoftonline.com/consumers/oauth2/v2.0/token", nil,
		common.NewURLEncodedBody(types.SM{
			"client_id":     clientId,
			"scope":         scope,
			"redirect_uri":  redirectUri,
			"grant_type":    grantType,
			"client_secret": clientSecret,
			key:             codeOrRefreshToken,
		}),
	)
	r := getTokenResp{}
	if e != nil {
		return r, e
	}
	if e := resp.Json(&r); e != nil {
		return r, e
	}

	return r, ds.Save(types.SM{
		"state":         "",
		"token":         r.TokenType + " " + r.AccessToken,
		"refresh_token": r.RefreshToken,
		"expires_at":    strconv.FormatInt(time.Now().Add(time.Duration(r.ExpiresIn)*time.Second).Unix(), 10),
	})
}

func getUser(token string) (userProfile, error) {
	user := userProfile{}
	resp, e := httpApi.Get(
		"https://graph.microsoft.com/v1.0/me",
		types.SM{"Authorization": token},
	)
	if e != nil {
		return user, e
	}
	if e := resp.Json(&user); e != nil {
		return user, e
	}
	return user, nil
}

func getDrives(token string) ([]driveInfo, error) {
	o := userDrives{}
	resp, e := httpApi.Get(
		"https://graph.microsoft.com/v1.0/me/drives",
		types.SM{"Authorization": token},
	)
	if e != nil {
		return nil, e
	}
	if e := resp.Json(&o); e != nil {
		return nil, e
	}
	return o.Drives, nil
}

func (o *OneDrive) addToken(req *http.Request) error {
	req.Header.Add("Authorization", o.accessToken)
	return nil
}

func (o *OneDrive) refreshToken() {
	params, e := o.ds.Load("refresh_token")
	if e != nil {
		log.Printf("load params error: %v", e)
		return
	}
	refreshToken := params["refresh_token"]

	r, e := getToken(true, refreshToken, o.clientId, o.clientSecret, o.ds)
	if e != nil {
		log.Printf("failed refresh access token: %v", e)
	} else {
		o.accessToken = r.TokenType + " " + r.AccessToken
	}
}

// uploadSmallFile uploads a new file that less than 4Mb
func (o *OneDrive) uploadSmallFile(parentId, filename string, size int64,
	reader io.Reader, ctx types.TaskCtx) (*oneDriveEntry, error) {
	ctx.Total(size, true)
	resp, e := o.c.Request("PUT",
		idURL(parentId)+":"+common.BuildURL("/{}:/content", filename),
		types.SM{"Content-Type": "application/octet-stream"},
		common.NewReadBody(reader, size),
	)
	if e != nil {
		return nil, e
	}
	ctx.Progress(size, true)
	return o.toEntry(resp)
}

// uploadSmallFile uploads file that less than 4Mb, override if exists
func (o *OneDrive) uploadSmallFileOverride(id string, size int64,
	reader io.Reader, ctx types.TaskCtx) (*oneDriveEntry, error) {
	ctx.Total(size, true)
	resp, e := o.c.Request("PUT",
		idURL(id)+"/content",
		types.SM{"Content-Type": "application/octet-stream"},
		common.NewReadBody(reader, size),
	)
	if e != nil {
		return nil, e
	}
	ctx.Progress(size, true)
	return o.toEntry(resp)
}

func (o *OneDrive) uploadLargeFile(parentId, filename string, size int64, override bool,
	reader io.Reader, ctx types.TaskCtx) (*oneDriveEntry, error) {
	ctx.Total(size, true)
	sessionUrl, e := o.createUploadSession(parentId, filename, override)
	if e != nil {
		return nil, e
	}
	chunkSize := int64(uploadChunkSize)
	var finalResp common.HttpResponse = nil
	for s := int64(0); s < size; s += chunkSize {
		if ctx.Canceled() {
			_ = deleteUploadSession(sessionUrl)
			return nil, task.ErrorCanceled
		}
		end := s + chunkSize
		if end > size {
			end = size
		}
		contentRange := fmt.Sprintf("bytes %d-%d/%d", s, end-1, size)
		resp, e := httpApi.RequestWithContext(
			"PUT", sessionUrl,
			types.SM{
				"Content-Range": contentRange,
				"Content-Type":  "application/octet-stream",
			},
			common.NewReadBody(io.LimitReader(reader, chunkSize), end-s),
			ctx,
		)
		if e != nil {
			_ = deleteUploadSession(sessionUrl)
			return nil, e
		}
		if end == size {
			finalResp = resp
		} else {
			_ = resp.Dispose()
		}
		ctx.Progress(end-s+1, false)
	}
	if finalResp == nil {
		panic("expect finalResp is not nil")
	}
	if finalResp.Status() != 201 {
		_ = deleteUploadSession(sessionUrl)
		return nil, errors.New(fmt.Sprintf("unexpected status code %d", finalResp.Status()))
	}
	return o.toEntry(finalResp)
}

func (o *OneDrive) createUploadSession(parentId, filename string, override bool) (string, error) {
	conflictBehavior := "fail"
	if override {
		conflictBehavior = "replace"
	}
	resp, e := o.c.Post(
		idURL(parentId)+":"+common.BuildURL("/{}:/createUploadSession", filename), nil,
		common.NewJsonBody(types.M{"item": types.M{"@microsoft.graph.conflictBehavior": conflictBehavior}}),
	)
	if e != nil {
		return "", e
	}
	createdUploadSession := createUploadSessionResp{}
	if e = resp.Json(&createdUploadSession); e != nil {
		return "", e
	}
	return createdUploadSession.UploadURL, nil
}

func deleteUploadSession(sessionUrl string) error {
	_, e := httpApi.Request("DELETE", sessionUrl, nil, nil)
	return e
}

func waitLongRunningAction(waitUrl string) error {
	for {
		resp, e := httpApi.Get(waitUrl, nil)
		if e != nil {
			return e
		}
		s := actionProgress{}
		if e := resp.Json(&s); e != nil {
			return e
		}
		if s.Status != "inProgress" {
			if s.Status != "completed" {
				return errors.New(fmt.Sprintf("unknown action status: %s", s.Status))
			}
			return nil
		}
		time.Sleep(2 * time.Second)
	}
}

func (o *OneDrive) toEntry(resp common.HttpResponse) (*oneDriveEntry, error) {
	item := driveItem{}
	if e := resp.Json(&item); e != nil {
		return nil, e
	}
	entry := o.newEntry(item)
	return entry, nil
}

func (o *OneDrive) deserializeEntry(dat string) (types.IEntry, error) {
	ec, e := drive_util.DeserializeEntry(dat)
	if e != nil {
		return nil, e
	}
	ed := ec.Data
	if ed == nil || ed["id"] == "" {
		return nil, errors.New("invalid cache")
	}
	return &oneDriveEntry{
		d: o, id: ed["id"],
		path: ec.Path, size: ec.Size, modTime: ec.ModTime, isDir: ec.Type.IsDir(),
		downloadUrl:          ed["du"],
		downloadUrlExpiresAt: common.ToInt64(ed["de"], -1),
	}, nil
}

func ifApiCallError(resp common.HttpResponse) error {
	if resp.Status() < 200 || resp.Status() >= 400 {
		err := apiError{}
		if e := resp.Json(&err); e != nil {
			return e
		}
		if err.Err.Code == "itemNotFound" {
			return common.NewNotFoundMessageError(err.Err.Message)
		}
		return err
	}
	return nil
}
