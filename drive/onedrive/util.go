package onedrive

import (
	"errors"
	"fmt"
	"go-drive/common"
	"go-drive/common/drive_util"
	"go-drive/common/req"
	"go-drive/common/task"
	"go-drive/common/types"
	"golang.org/x/oauth2"
	"io"
	path2 "path"
	"strings"
	"time"
)

var oauth = drive_util.OAuthRequest{
	Endpoint: oauth2.Endpoint{
		AuthURL:   "https://login.microsoftonline.com/consumers/oauth2/v2.0/authorize",
		TokenURL:  "https://login.microsoftonline.com/consumers/oauth2/v2.0/token",
		AuthStyle: oauth2.AuthStyleInParams,
	},
	RedirectURL: drive_util.CommonRedirectURL,
	Scopes:      []string{"Files.ReadWrite", "offline_access", "User.Read"},
	Text:        "Connect to OneDrive",
}

var httpApi, _ = req.NewClient("", nil, ifApiCallError, nil)

var thumbnailExtensions = make(map[string]bool)

func init() {
	// https://support.microsoft.com/en-us/office/file-types-supported-for-previewing-files-in-onedrive-sharepoint-and-teams-e054cd0f-8ef2-4ccb-937e-26e37419c5e4
	for _, ext := range strings.Split("3mf cool glb gltf obj stl movie pages pict sketch ai pdf psb psd 3g2 3gp asf bmp hevc m2ts m4v mov mp3 mp4 mp4v mts ts wmv dwg fbx erf zip z dcm dcm30 dicm dicom ply hcp gif heic heif jpeg jpg jpe mef mrw nef nrw orf pano pef png rw2 spm tif tiff xbm xcf key log csv dic doc docm docx dotm dotx pot potm potx pps ppsm ppsx ppt pptm pptx xd xls xlsb xlsx sltx eml msg vsd vsdx cur ico icon epub odp ods odt arw cr2 crw dng rtf abap ada adp ahk as as3 asc ascx asm asp awk bas bash bash_login bash_logout bash_profile bashrc bat bib bsh build builder c c++ capfile cbk cc cfc cfm cfml cl clj cmake cmd coffee cpp cpt cpy cs cshtml cson csproj css ctp cxx d ddl di.dif diff disco dml dtd dtml el emake erb erl f90 f95 fs fsi fsscript fsx gemfile gemspec gitconfig go groovy gvy h h++ haml handlebars hbs hrl hs htc html hxx idl iim inc inf ini inl ipp irbrc jade jav java js json jsp jsx l less lhs lisp log lst ltx lua m make markdn markdown md mdown mkdn ml mli mll mly mm mud nfo opml osascript out p pas patch php php2 php3 php4 php5 pl plist pm pod pp profile properties ps ps1 pt py pyw r rake rb rbx rc re readme reg rest resw resx rhtml rjs rprofile rpy rss rst rxml s sass scala scm sconscript sconstruct script scss sgml sh shtml sml sql sty tcl tex text textile tld tli tmpl tpl txt vb vi vim wsdl xaml xhtml xoml xml xsd xsl xslt yaml yaws yml zsh htm html markdown md url", " ") {
		thumbnailExtensions["."+ext] = true
	}
}

func supportThumbnail(item driveItem) bool {
	return item.Folder != nil || thumbnailExtensions[strings.ToLower(path2.Ext(item.Name))]
}

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
	return "/drive/root:/" + path
}

func InitConfig(config drive_util.DriveConfig, utils drive_util.DriveUtils) (*drive_util.DriveInitConfig, error) {
	initConfig, resp, e := drive_util.OAuthInitConfig(oauth, config, utils.Data)
	if e != nil {
		return nil, e
	}
	if resp == nil {
		return initConfig, nil
	}
	reqClient, e := req.NewClient("", nil, ifApiCallError, resp.Client(nil))
	if e != nil {
		return nil, e
	}

	// get user
	user, e := getUser(reqClient)
	initConfig.Configured = e == nil
	if e == nil {
		initConfig.OAuth.Principal = fmt.Sprintf("%s <%s>", user.DisplayName, user.UserPrincipalName)
	}

	params, e := utils.Data.Load("drive_id")
	if e != nil {
		return nil, e
	}

	// get drives
	if initConfig.Configured {
		drives, e := getDrives(reqClient)
		initConfig.Configured = e == nil
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
			initConfig.Form = []types.FormItem{
				{Label: "Drive", Type: "select", Field: "drive_id", Required: true, Options: opts},
			}
			initConfig.Value = types.SM{"drive_id": params["drive_id"]}
		}
	}

	if initConfig.Configured {
		initConfig.Configured = params["drive_id"] != ""
	}

	return initConfig, nil
}

func Init(data types.SM, config drive_util.DriveConfig, utils drive_util.DriveUtils) error {
	_, e := drive_util.OAuthInit(oauth, data, config, utils.Data)
	if e != nil {
		return e
	}
	driveId := data["drive_id"]
	if driveId != "" {
		return utils.Data.Save(types.SM{"drive_id": driveId})
	}
	return nil
}

func getUser(req *req.Client) (userProfile, error) {
	user := userProfile{}
	resp, e := req.Get("https://graph.microsoft.com/v1.0/me", nil)
	if e != nil {
		return user, e
	}
	if e := resp.Json(&user); e != nil {
		return user, e
	}
	return user, nil
}

func getDrives(req *req.Client) ([]driveInfo, error) {
	o := userDrives{}
	resp, e := req.Get("https://graph.microsoft.com/v1.0/me/drives", nil)
	if e != nil {
		return nil, e
	}
	if e := resp.Json(&o); e != nil {
		return nil, e
	}
	return o.Drives, nil
}

// uploadSmallFile uploads a new file that less than 4Mb
func (o *OneDrive) uploadSmallFile(parentId, filename string, size int64,
	reader io.Reader, ctx types.TaskCtx) (*oneDriveEntry, error) {
	ctx.Total(size, true)
	resp, e := o.c.Request("PUT",
		idURL(parentId)+":"+common.BuildURL("/{}:/content", filename),
		types.SM{"Content-Type": "application/octet-stream"},
		req.NewReaderBody(reader, size),
	)
	if e != nil {
		return nil, e
	}
	ctx.Progress(size, false)
	return o.toEntry(resp)
}

// uploadSmallFile uploads file that less than 4Mb, override if exists
func (o *OneDrive) uploadSmallFileOverride(id string, size int64,
	reader io.Reader, ctx types.TaskCtx) (*oneDriveEntry, error) {
	ctx.Total(size, true)
	resp, e := o.c.Request("PUT",
		idURL(id)+"/content",
		types.SM{"Content-Type": "application/octet-stream"},
		req.NewReaderBody(drive_util.ProgressReader(reader, ctx), size),
	)
	if e != nil {
		return nil, e
	}
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
	var finalResp req.Response = nil
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
			req.NewReaderBody(drive_util.ProgressReader(io.LimitReader(reader, chunkSize), ctx), end-s),
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
		req.NewJsonBody(types.M{"item": types.M{"@microsoft.graph.conflictBehavior": conflictBehavior}}),
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
		if s.Status != "inProgress" && s.Status != "notStarted" {
			if s.Status != "completed" {
				return errors.New(fmt.Sprintf("unknown action status: %s", s.Status))
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
		thumbnail:            ed["th"],
	}, nil
}

func ifApiCallError(resp req.Response) error {
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
