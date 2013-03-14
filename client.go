package main

import (
	"code.google.com/p/google-api-go-client/drive/v2"
	"encoding/json"
	"net/http"
)

type driveFileList struct {
	Items []driveFile
}

// We define our own type because the one from google-api-go-client is bugged
// (exportLinks is an empty struct).
type driveFile struct {
	DownloadUrl  string
	Editable     bool
	ExportLinks  map[string]string
	FileSize     int64 `json:",string"`
	Id           string
	MimeType     string
	ModifiedDate string
	Parents      []drive.ParentReference
	Title        string
}

func getRoot() (root *driveFile, err error) {
	const url = "https://www.googleapis.com/drive/v2/files/root?fields=editable%2Cid%2Ctitle%2CmodifiedDate"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	resp, err := transport.Client().Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	root = new(driveFile)
	err = dec.Decode(root)
	return
}

func listFiles() (list driveFileList, err error) {
	const url = "https://www.googleapis.com/drive/v2/files?maxResults=2147483647&fields=items(downloadUrl%2Ceditable%2CexportLinks%2CfileSize%2Cid%2CmimeType%2CmodifiedDate%2Cparents%2Ctitle)"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	resp, err := transport.Client().Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&list)
	return
}
