package main

import (
	"container/list"
	"github.com/hanwen/go-fuse/fuse"
	"log"
)

type Filesystem struct {
	root *dirNode
	uid  uint32
	gid  uint32
}

func (fs *Filesystem) OnMount(conn *fuse.FileSystemConnector) {
	rootFile, err := getRoot()
	if err != nil {
		log.Fatal("Failed to get root folder metadata:", err)
	}
	list, err := listFiles()
	if err != nil {
		log.Fatal("Failed to list files:", err)
	}
	root := newDirNode(rootFile)
	fs.root.atime = root.atime
	fs.root.id = root.id
	fs.root.mode = root.mode
	fs.root.mtime = root.mtime
	fs.root.name = root.name
	fs.root.attachChildren(toList(list))
}

func (fs *Filesystem) OnUnmount() {
}

func (fs *Filesystem) Root() fuse.FsNode {
	return fs.root
}

func (fs *Filesystem) String() string {
	return "drivefs"
}

type Node interface {
	fuse.FsNode
	Name() string
}

func newNode(f *driveFile) (node Node) {
	switch f.MimeType {
	case "application/vnd.google-apps.folder":
		node = newDirNode(f)
	case "application/vnd.google-apps.document",
		"application/vnd.google-apps.spreadsheet",
		"application/vnd.google-apps.presentation",
		"application/vnd.google-apps.drawing":
		node = newDocDirNode(f)
	default:
		node = newFileNode(f)
	}
	return
}

func toList(files driveFileList) (l *list.List) {
	l = list.New()
	for _, v := range files.Items {
		l.PushBack(v)
	}
	return
}
