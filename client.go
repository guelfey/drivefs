package main

import (
	"code.google.com/p/google-api-go-client/drive/v2"
	"encoding/json"
	"errors"
	"net/http"
)

type driveFileList struct {
	Items []driveFile
}

// We define our own type because the one from google-api-go-client is bugged
// (exportLinks is an empty struct).
type driveFile struct {
	CreatedDate        string
	DownloadUrl        string
	Editable           bool
	ExportLinks        map[string]string
	FileSize           int64 `json:",string"`
	Id                 string
	LastViewedByMeDate string
	MimeType           string
	ModifiedDate       string
	Parents            []drive.ParentReference
	Title              string
}

func getRoot() (root *driveFile, err error) {
	const url = "https://www.googleapis.com/drive/v2/files/root?fields=createdDate%2Ceditable%2Cid%2ClastViewedByMeDate%2CmodifiedDate%2Ctitle"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	resp, err := transport.Client().Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, errors.New(resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	root = new(driveFile)
	err = dec.Decode(root)
	return
}

func listFiles() (list driveFileList, err error) {
	const url = "https://www.googleapis.com/drive/v2/files?maxResults=2147483647&fields=items(createdDate%2CdownloadUrl%2Ceditable%2CexportLinks%2CfileSize%2Cid%2ClastViewedByMeDate%2CmimeType%2CmodifiedDate%2Cparents%2Ctitle)"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	resp, err := transport.Client().Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return driveFileList{}, errors.New(resp.Status)
	}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&list)
	return
}
