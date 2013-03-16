package main

import (
	"github.com/hanwen/go-fuse/fuse"
	"log"
)

type Filesystem struct {
	root     *dirNode
	uid      uint32
	gid      uint32
	idToNode map[string]Node
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
	fs.idToNode = make(map[string]Node)
	fs.idToNode[fs.root.id] = fs.root
	for _, v := range list.Items {
		n := newNode(&v)
		fs.idToNode[v.Id] = n
	}
	for _, v := range list.Items {
		n := fs.idToNode[v.Id]
		// TODO what to do with files having no parents (trash etc.)?
		for _, p := range v.Parents {
			parent, _ := fs.idToNode[p.Id].(*dirNode)
			if parent == nil {
				continue
			}
			parent.Inode().AddChild(n.Name(), n.Inode())
		}
	}
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
