package onedrive

import (
	"context"
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/drive_util"
	err "go-drive/common/errors"
	"go-drive/common/i18n"
	"go-drive/common/req"
	"go-drive/common/types"
	"go-drive/common/utils"
	"io"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

var t = i18n.TPrefix("drive.onedrive.")

type apiConfig struct {
	AuthorizeURL string
	TokenURL     string
	ApiBase      string
}

var sites = map[string]apiConfig{
	"global": {
		AuthorizeURL: "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/authorize",
		TokenURL:     "https://login.microsoftonline.com/{tenant}/oauth2/v2.0/token",
		ApiBase:      "https://graph.microsoft.com/v1.0",
	},
	"china": {
		AuthorizeURL: "https://login.chinacloudapi.cn/{tenant}/oauth2/v2.0/authorize",
		TokenURL:     "https://login.chinacloudapi.cn/{tenant}/oauth2/v2.0/token",
		ApiBase:      "https://microsoftgraph.chinacloudapi.cn/v1.0",
	},
}

func getSiteConfig(site string) apiConfig {
	s, ok := sites[site]
	if !ok {
		s = sites["global"]
	}
	return s
}

func oauthReq(c common.Config, config types.SM) *drive_util.OAuthRequest {
	tenant := config["tenant"]
	site := getSiteConfig(config["site"])

	filesScope := "Files.ReadWrite"
	if config["share_point"] != "" {
		filesScope = "Files.ReadWrite.All"
	}

	return &drive_util.OAuthRequest{
		Endpoint: oauth2.Endpoint{
			AuthURL:   strings.ReplaceAll(site.AuthorizeURL, "{tenant}", tenant),
			TokenURL:  strings.ReplaceAll(site.TokenURL, "{tenant}", tenant),
			AuthStyle: oauth2.AuthStyleInParams,
		},
		RedirectURL: c.OAuthRedirectURI,
		Scopes:      []string{filesScope, "offline_access", "User.Read"},
		Text:        i18n.T("drive.onedrive.oauth_text"),
	}
}

var httpApi, _ = req.NewClient("", nil, ifApiCallError, nil)

var thumbnailExtensions = make(map[string]bool)

func init() {
	// https://support.microsoft.com/en-us/office/file-types-supported-for-previewing-files-in-onedrive-sharepoint-and-teams-e054cd0f-8ef2-4ccb-937e-26e37419c5e4
	for _, ext := range strings.Split("3mf cool glb gltf obj stl movie pages pict sketch ai pdf psb psd 3g2 3gp asf bmp hevc m2ts m4v mov mp3 mp4 mp4v mts ts wmv dwg fbx erf zip z dcm dcm30 dicm dicom ply hcp gif heic heif jpeg jpg jpe mef mrw nef nrw orf pano pef png rw2 spm tif tiff xbm xcf key log csv dic doc docm docx dotm dotx pot potm potx pps ppsm ppsx ppt pptm pptx xd xls xlsb xlsx sltx eml msg vsd vsdx cur ico icon epub odp ods odt arw cr2 crw dng rtf abap ada adp ahk as as3 asc ascx asm asp awk bas bash bash_login bash_logout bash_profile bashrc bat bib bsh build builder c c++ capfile cbk cc cfc cfm cfml cl clj cmake cmd coffee cpp cpt cpy cs cshtml cson csproj css ctp cxx d ddl di.dif diff disco dml dtd dtml el emake erb erl f90 f95 fs fsi fsscript fsx gemfile gemspec gitconfig go groovy gvy h h++ haml handlebars hbs hrl hs htc html hxx idl iim inc inf ini inl ipp irbrc jade jav java js json jsp jsx l less lhs lisp log lst ltx lua m make markdn markdown md mdown mkdn ml mli mll mly mm mud nfo opml osascript out p pas patch php php2 php3 php4 php5 pl plist pm pod pp profile properties ps ps1 pt py pyw r rake rb rbx rc re readme reg rest resw resx rhtml rjs rprofile rpy rss rst rxml s sass scala scm sconscript sconstruct script scss sgml sh shtml sml sql sty tcl tex text textile tld tli tmpl tpl txt vb vi vim wsdl xaml xhtml xoml xml xsd xsl xslt yaml yaws yml zsh htm html markdown md url", " ") {
		thumbnailExtensions[ext] = true
	}
}

func supportThumbnail(item driveItem) bool {
	return item.Folder != nil || thumbnailExtensions[utils.PathExt(item.Name)]
}

func pathURL(path string) string {
	if utils.IsRootPath(path) {
		return "/root"
	}
	return utils.BuildURL("/root:/{}:", path)
}

func idURL(id string) string {
	return utils.BuildURL("/items/{}", id)
}

func itemPath(path string) string {
	if utils.IsRootPath(path) {
		return "/drive/root:"
	}
	return "/drive/root:/" + path
}

func InitConfig(ctx context.Context, config types.SM,
	driveUtils drive_util.DriveUtils) (*drive_util.DriveInitConfig, error) {
	initConfig, resp, e := drive_util.OAuthInitConfig(*oauthReq(driveUtils.Config, config),
		drive_util.OAuthCredentials{
			ClientID:     config["client_id"],
			ClientSecret: config["client_secret"],
		}, driveUtils.Data)
	if e != nil {
		return nil, e
	}
	if resp == nil {
		return initConfig, nil
	}
	reqClient, e := req.NewClient("", nil, ifApiCallError, resp.Client())
	if e != nil {
		return nil, e
	}

	// get user
	user, e := getUser(ctx, reqClient, config["site"])
	initConfig.Configured = e == nil

	if e != nil {
		log.Println("[OneDrive] Failed to get user info", e)
	}

	if e == nil {
		initConfig.OAuth.Principal = fmt.Sprintf("%s <%s>", user.DisplayName, user.UserPrincipalName)
	}

	params, e := driveUtils.Data.Load("drive_id", "share_point_id")
	if e != nil {
		return nil, e
	}

	sharePointURL := config["share_point"]

	// get drives
	if initConfig.Configured && sharePointURL == "" {
		_ = generateDrivesForm(ctx, reqClient, config, params, initConfig)
	}

	// if no SharePoint site provided, get the user's drives
	// otherwise, treat the SharePoint site as a drive
	if initConfig.Configured {
		if sharePointURL == "" {
			initConfig.Configured = params["drive_id"] != ""
		} else {
			initConfig.Configured = params["share_point_id"] != ""
		}
	}

	return initConfig, nil
}

func Init(ctx context.Context, data types.SM, config types.SM, utils drive_util.DriveUtils) error {
	oReq := *oauthReq(utils.Config, config)
	resp, e := drive_util.OAuthInit(ctx, oReq, data,
		drive_util.OAuthCredentials{
			ClientID:     config["client_id"],
			ClientSecret: config["client_secret"],
		}, utils.Data)
	if e != nil {
		return e
	}

	var paramsData types.SM

	sharePointURL := config["share_point"]
	if sharePointURL != "" {
		if resp == nil {
			resp, e = drive_util.OAuthGet(oReq,
				drive_util.OAuthCredentials{
					ClientID:     config["client_id"],
					ClientSecret: config["client_secret"],
				}, utils.Data)
			if e != nil {
				return e
			}
		}

		reqClient, e := req.NewClient("", nil, ifApiCallError, resp.Client())
		if e != nil {
			return e
		}
		info, e := getSharePointInfo(ctx, reqClient, config["site"], sharePointURL)
		if e != nil {
			return e
		}

		paramsData = types.SM{"share_point_id": info.Id, "drive_id": ""}
	} else {
		paramsData = types.SM{"drive_id": data["drive_id"], "share_point_id": ""}
	}

	return utils.Data.Save(paramsData)
}

func generateDrivesForm(ctx context.Context, reqClient *req.Client,
	config types.SM, params types.SM, initConfig *drive_util.DriveInitConfig) error {
	drives, e := getDrives(ctx, reqClient, config["site"])
	initConfig.Configured = e == nil
	if e != nil {
		log.Println("[OneDrive] Error getting drives:", e)
		return e
	}
	opts := make([]types.FormItemOption, len(drives))
	for i, d := range drives {
		used := "-"
		if d.Quota.Total != 0 {
			used = fmt.Sprintf("%.1f%%", float64(d.Quota.Used)/float64(d.Quota.Total)*100)
		}
		opts[i] = types.FormItemOption{
			Name: fmt.Sprintf("%s %d", d.DriveType, i+1),
			Title: i18n.T("drive.onedrive.drive_used",
				utils.FormatBytes(uint64(d.Quota.Used), 1),
				utils.FormatBytes(uint64(d.Quota.Total), 1),
				used),
			Value: d.Id,
		}
	}
	initConfig.Form = []types.FormItem{
		{Label: i18n.T("drive.onedrive.drive_select"), Type: "select", Field: "drive_id", Required: true, Options: &opts},
	}
	initConfig.Value = types.SM{"drive_id": params["drive_id"]}
	return nil
}

func getUser(ctx context.Context, req *req.Client, site string) (userProfile, error) {
	s := getSiteConfig(site)
	user := userProfile{}
	resp, e := req.Get(ctx, s.ApiBase+"/me", nil)
	if e != nil {
		return user, e
	}
	e = resp.Json(&user)
	return user, e
}

func getDrives(ctx context.Context, req *req.Client, site string) ([]driveInfo, error) {
	s := getSiteConfig(site)
	o := userDrives{}
	resp, e := req.Get(ctx, s.ApiBase+"/me/drives", nil)
	if e != nil {
		return nil, e
	}
	e = resp.Json(&o)
	return o.Drives, e
}

func getSharePointInfo(ctx context.Context, req *req.Client, site string, sharePointURL string) (sharePointInfo, error) {
	parsedURL, e := url.Parse(sharePointURL)
	if e != nil {
		return sharePointInfo{}, e
	}
	s := getSiteConfig(site)
	o := sharePointInfo{}
	resp, e := req.Get(ctx, s.ApiBase+utils.BuildURL(
		"/sites/{}:/{}",
		parsedURL.Hostname(),
		strings.Trim(parsedURL.Path, "/"),
	), nil)
	if e != nil {
		return o, e
	}
	e = resp.Json(&o)
	return o, e
}

// uploadSmallFile uploads a new file that less than 4Mb
func (o *OneDrive) uploadSmallFile(ctx types.TaskCtx,
	parentId, filename string, size int64, reader io.Reader) (*oneDriveEntry, error) {
	ctx.Total(size, true)
	resp, e := o.c.Request(ctx, "PUT", idURL(parentId)+":"+utils.BuildURL("/{}:/content", filename),
		types.SM{"Content-Type": "application/octet-stream"}, req.NewReaderBody(reader, size))
	if e != nil {
		return nil, e
	}
	ctx.Progress(size, false)
	return o.toEntry(resp)
}

// uploadSmallFile uploads file that less than 4Mb, override if exists
func (o *OneDrive) uploadSmallFileOverride(ctx types.TaskCtx,
	id string, size int64, reader io.Reader) (*oneDriveEntry, error) {
	ctx.Total(size, true)
	resp, e := o.c.Request(ctx, "PUT", idURL(id)+"/content", types.SM{"Content-Type": "application/octet-stream"},
		req.NewReaderBody(drive_util.ProgressReader(reader, ctx), size))
	if e != nil {
		return nil, e
	}
	return o.toEntry(resp)
}

func (o *OneDrive) uploadLargeFile(ctx types.TaskCtx,
	parentId, filename string, size int64, override bool, reader io.Reader) (*oneDriveEntry, error) {
	ctx.Total(size, true)
	sessionUrl, e := o.createUploadSession(ctx, parentId, filename, override)
	if e != nil {
		return nil, e
	}
	chunkSize := int64(uploadChunkSize)
	var finalResp req.Response = nil
	for s := int64(0); s < size; s += chunkSize {
		if e := ctx.Err(); e != nil {
			_ = deleteUploadSession(ctx, sessionUrl)
			return nil, e
		}
		end := s + chunkSize
		if end > size {
			end = size
		}
		contentRange := fmt.Sprintf("bytes %d-%d/%d", s, end-1, size)
		resp, e := httpApi.Request(ctx, "PUT", sessionUrl, types.SM{
			"Content-Range": contentRange,
			"Content-Type":  "application/octet-stream",
		}, req.NewReaderBody(drive_util.ProgressReader(io.LimitReader(reader, chunkSize), ctx), end-s))
		if e != nil {
			_ = deleteUploadSession(ctx, sessionUrl)
			return nil, e
		}
		if end == size {
			finalResp = resp
		} else {
			_ = resp.Dispose()
		}
	}
	if finalResp == nil {
		panic("expect finalResp is not nil")
	}
	if finalResp.Status() != 201 && finalResp.Status() != 200 {
		_ = deleteUploadSession(ctx, sessionUrl)
		return nil, errors.New(i18n.T("drive.onedrive.unexpected_status", strconv.Itoa(finalResp.Status())))
	}
	return o.toEntry(finalResp)
}

func (o *OneDrive) createUploadSession(ctx context.Context, parentId, filename string, override bool) (string, error) {
	conflictBehavior := "fail"
	if override {
		conflictBehavior = "replace"
	}
	resp, e := o.c.Post(ctx, idURL(parentId)+":"+utils.BuildURL("/{}:/createUploadSession", filename),
		nil, req.NewJsonBody(types.M{"item": types.M{"@microsoft.graph.conflictBehavior": conflictBehavior}}))
	if e != nil {
		return "", e
	}
	createdUploadSession := createUploadSessionResp{}
	if e = resp.Json(&createdUploadSession); e != nil {
		return "", e
	}
	return createdUploadSession.UploadURL, nil
}

func deleteUploadSession(ctx context.Context, sessionUrl string) error {
	_, e := httpApi.Request(ctx, "DELETE", sessionUrl, nil, nil)
	return e
}

func waitLongRunningAction(ctx context.Context, waitUrl string) error {
	for {
		resp, e := httpApi.Get(ctx, waitUrl, nil)
		if e != nil {
			return e
		}
		s := actionProgress{}
		if e := resp.Json(&s); e != nil {
			return e
		}
		if s.Status != "inProgress" && s.Status != "notStarted" {
			if s.Status != "completed" {
				return errors.New(i18n.T("drive.onedrive.unknown_action_status", s.Status))
			}
			return nil
		}
		time.Sleep(2 * time.Second)
	}
}

func (o *OneDrive) toEntry(resp req.Response) (*oneDriveEntry, error) {
	item := driveItem{}
	if e := resp.Json(&item); e != nil {
		return nil, e
	}
	entry := o.newEntry(item)
	return entry, nil
}

func (o *OneDrive) deserializeEntry(ec drive_util.EntryCacheItem) (types.IEntry, error) {
	ed := ec.Data
	if ed == nil || ed["id"] == "" {
		return nil, errors.New("invalid cache")
	}
	return &oneDriveEntry{
		d: o, id: ed["id"],
		path: ec.Path, size: ec.Size, modTime: ec.ModTime, isDir: ec.Type.IsDir(),
		downloadUrl:          ed["du"],
		downloadUrlExpiresAt: ed.GetInt64("de", -1),
		thumbnail:            ed["th"],
	}, nil
}

func ifApiCallError(resp req.Response) error {
	if resp.Status() < 200 || resp.Status() >= 400 {
		ee := apiError{}
		if e := resp.Json(&ee); e != nil {
			return e
		}
		if ee.Err.Code == "itemNotFound" {
			return err.NewNotFoundMessageError(ee.Err.Message)
		}
		return ee
	}
	return nil
}
