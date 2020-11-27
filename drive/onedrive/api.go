package onedrive

import (
	"fmt"
	"go-drive/common"
	"net/url"
	path2 "path"
	"time"
)

const uploadChunkSize = 4 * 1024 * 1024

// https://docs.microsoft.com/en-us/graph/api/resources/driveitem?view=graph-rest-1.0#instance-attributes
const downloadUrlTTL = 40 * time.Minute

type userProfile struct {
	DisplayName       string `json:"displayName"`
	UserPrincipalName string `json:"userPrincipalName"`
}

type driveInfo struct {
	Id        string `json:"id"`
	DriveType string `json:"driveType"`
	Quota     struct {
		Total int64 `json:"total"`
		Used  int64 `json:"used"`
	} `json:"quota"`
}

type userDrives struct {
	Drives []driveInfo `json:"value"`
}

// https://docs.microsoft.com/en-us/graph/api/resources/audio?view=graph-rest-1.0
type audioInfo struct {
	Duration int    `json:"duration"`
	Album    string `json:"album"`
	Artist   string `json:"artist"`
	Title    string `json:"title"`
}

// https://docs.microsoft.com/en-us/graph/api/resources/deleted?view=graph-rest-1.0
type deleteInfo struct {
	State string `json:"state"`
}

// https://docs.microsoft.com/en-us/graph/api/resources/file?view=graph-rest-1.0
type fileInfo struct {
	MimeType string `json:"mimeType"`
	Hashes   struct {
		QuickXorHash string `json:"quickXorHash"`
		Sha1Hash     string `json:"sha1Hash"`
	} `json:"hashes"`
}

// https://docs.microsoft.com/en-us/graph/api/resources/folder?view=graph-rest-1.0
type folderInfo struct {
	ChildCount int `json:"childCount"`
}

// https://docs.microsoft.com/en-us/graph/api/resources/image?view=graph-rest-1.0
type imageInfo struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// https://docs.microsoft.com/en-us/graph/api/resources/video?view=graph-rest-1.0
type videoInfo struct {
	Duration int `json:"duration"`
	Width    int `json:"width"`
	Height   int `json:"height"`
}

type thumbnailItem struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	URL    string `json:"url"`
}

type thumbnailInfo struct {
	Large  *thumbnailItem `json:"large"`
	Medium *thumbnailItem `json:"medium"`
	Small  *thumbnailItem `json:"small"`
}

// https://docs.microsoft.com/en-us/graph/api/resources/driveitem?view=graph-rest-1.0
type driveItem struct {
	Id      string      `json:"id"`
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	Deleted *deleteInfo `json:"deleted"`

	ETag string `json:"eTag"`

	File   *fileInfo   `json:"file"`
	Folder *folderInfo `json:"folder"`

	Image *imageInfo `json:"image"`
	Audio *audioInfo `json:"audio"`
	Video *videoInfo `json:"video"`

	ModTime string `json:"lastModifiedDateTime"`

	DownloadURL string `json:"@microsoft.graph.downloadUrl"`

	Thumbnails []thumbnailInfo `json:"thumbnails"`

	Parent struct {
		Id   string `json:"id"`
		Path string `json:"path"`
	} `json:"parentReference"`
}

func (d driveItem) Path() string {
	// Remove parent prefix /drive/root:
	parentPath, e := url.PathUnescape(d.Parent.Path[12:])
	if e != nil {
		parentPath = d.Parent.Path[12:]
	}
	return common.CleanPath(path2.Join(parentPath, d.Name))
}

type driveItems struct {
	Items []driveItem `json:"value"`
}

type createUploadSessionResp struct {
	UploadURL string `json:"uploadUrl"`
}

type actionProgress struct {
	Percent    float32 `json:"percentageComplete"`
	ResourceId string  `json:"resourceId"`
	Status     string  `json:"status"`
}

type apiError struct {
	Err struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (a apiError) Error() string {
	return fmt.Sprintf("%s: %s", a.Err.Code, a.Err.Message)
}
