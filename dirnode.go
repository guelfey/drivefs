package main

import (
	"container/list"
	"github.com/hanwen/go-fuse/fuse"
	"log"
	"time"
)

type dirNode struct {
	id      string
	mode    uint32
	modTime time.Time
	name    string
	fuse.DefaultFsNode
}

func newDirNode(file *driveFile) *dirNode {
	t, err := time.Parse(time.RFC3339, file.ModifiedDate)
	if err != nil {
		t = time.Unix(0, 0)
		log.Println(file.Title, err)
	}
	mode := uint32(fuse.S_IFDIR | 0500)
	if file.Editable {
		mode |= 0200
	}
	n := &dirNode{id: file.Id, modTime: t, mode: mode, name: file.Title}
	_ = fs.root.Inode().New(true, n)
	return n
}

func (n *dirNode) attachChildren(l *list.List) {
	for e := l.Front(); e != nil; {
		var attached bool

		v := e.Value.(driveFile)
		for _, p := range v.Parents {
			if p.Id == n.id {
				attached = true
				m := newNode(&v)
				n.Inode().AddChild(m.Name(), m.Inode())
				prev := e.Prev()
				l.Remove(e)
				if prev == nil {
					e = l.Front()
				} else {
					e = prev.Next()
				}
				break
			}
		}
		if !attached {
			e = e.Next()
		}
	}
	for _, v := range n.Inode().FsChildren() {
		if n, ok := v.FsNode().(*dirNode); ok {
			n.attachChildren(l)
		}
	}
}

func (n *dirNode) GetAttr(out *fuse.Attr, file fuse.File, context *fuse.Context) fuse.Status {
	if n == nil {
		return fuse.ENOENT
	}
	out.Mtime = uint64(n.modTime.Unix())
	out.Owner.Uid = fs.uid
	out.Owner.Gid = fs.gid
	out.Mode = n.mode
	return fuse.OK
}

func (n *dirNode) Name() string {
	return n.name
}
