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
	root := newDirNode(rootFile, 0)
	fs.root.atime = root.atime
	fs.root.id = root.id
	fs.root.mode = root.mode
	fs.root.mtime = root.mtime
	fs.root.name = root.name
	fs.root.ino = 1
	fs.idToNode = make(map[string]Node)
	fs.idToNode[fs.root.id] = fs.root
	for i, v := range list.Items {
		n := newNode(&v, uint64(i+2))
		fs.idToNode[v.Id] = n
	}
	for _, v := range list.Items {
		n := fs.idToNode[v.Id]
		// TODO what to do with files having no parents (trash etc.)?
	parents:
		for _, p := range v.Parents {
			parent, _ := fs.idToNode[p.Id].(*dirNode)
			if parent == nil {
				continue
			}
			parent.Inode().AddChild(n.Name(), n.Inode())
			switch n := n.(type) {
			case (*dirNode):
				// XXX Multiple parents for directories are impossible to
				// implement correctly, since the only way to represent them is
				// using hard links which are impossible (because the kernel
				// assumes them to be impossible). A hacky solution would be to
				// use symlinks. For now, we ignore any parents after the first.
				break parents
			case (*fileNode):
				n.nlink++
			}
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

func newNode(f *driveFile, ino uint64) (node Node) {
	switch f.MimeType {
	case "application/vnd.google-apps.folder":
		node = newDirNode(f, ino)
	case "application/vnd.google-apps.document",
		"application/vnd.google-apps.spreadsheet",
		"application/vnd.google-apps.presentation",
		"application/vnd.google-apps.drawing":
		node = newDocDirNode(f, ino)
	default:
		node = newFileNode(f, ino)
	}
	return
}
