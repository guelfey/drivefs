package main

import (
	"code.google.com/p/google-api-go-client/drive/v2"
	"container/list"
	"github.com/hanwen/go-fuse/fuse"
	"log"
	"sync"
	"syscall"
	"time"
)

type dirNode struct {
	atime   time.Time
	id      string
	mode    uint32
	mtime   time.Time
	name    string
	fuse.DefaultFsNode
	sync.RWMutex
}

func newDirNode(file *driveFile) *dirNode {
	var err error

	n := new(dirNode)
	_ = fs.root.Inode().New(true, n)
	n.id = file.Id
	n.name = file.Title
	n.mtime, err = time.Parse(time.RFC3339, file.ModifiedDate)
	if err != nil {
		n.mtime = time.Unix(0, 0)
		log.Println(n.name, err)
	}
	n.mode = uint32(fuse.S_IFDIR | 0500)
	if file.Editable {
		n.mode |= 0200
	}
	var t string
	if file.LastViewedByMeDate == "" {
		t = file.CreatedDate
	} else {
		t = file.LastViewedByMeDate
	}
	n.atime, err = time.Parse(time.RFC3339Nano, t)
	if err != nil {
		n.atime = time.Unix(0, 0)
		log.Println(n.name, err)
	}
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
	n.RLock()
	defer n.RUnlock()
	if n == nil {
		return fuse.ENOENT
	}
	out.Atime = uint64(n.atime.Unix())
	out.Mtime = uint64(n.mtime.Unix())
	out.Owner.Uid = fs.uid
	out.Owner.Gid = fs.gid
	out.Mode = n.mode
	return fuse.OK
}

func (n *dirNode) Name() string {
	return n.name
}

func (n *dirNode) Rmdir(name string, context *fuse.Context) fuse.Status {
	n.Lock()
	defer n.Unlock()
	if context.Uid != fs.uid || n.mode&0200 == 0 {
		return fuse.EACCES
	}
	cinode := n.Inode().Children()[name]
	if cinode == nil {
		return fuse.ENOENT
	}
	switch child := cinode.FsNode().(type) {
	case (*docDirNode):
		if child.mode&0200 == 0 {
			return fuse.EPERM
		}
		err := srv.Files.Delete(child.id).Do()
		if err != nil {
			log.Print(err)
			return fuse.EIO
		}
		n.Inode().RmChild(name)
	case (*dirNode):
		if child.mode&0200 == 0 {
			return fuse.EPERM
		}
		child.Lock()
		defer child.Unlock()
		if len(child.Inode().Children()) != 0 {
			return fuse.Status(syscall.ENOTEMPTY)
		}
		err := srv.Files.Delete(child.id).Do()
		if err != nil {
			log.Print(err)
			return fuse.EIO
		}
		n.Inode().RmChild(name)
	case (*fileNode):
		return fuse.ENOTDIR
	default:
		return fuse.EINVAL
	}
	n.setAtime(time.Now())
	return fuse.OK
}

// n must already be locked for writing
func (n *dirNode) setAtime(t time.Time) error {
	n.atime = t
	f := &drive.File{LastViewedByMeDate: t.Format(time.RFC3339Nano)}
	_, err := srv.Files.Patch(n.id, f).UpdateViewedDate(false).Do()
	return err
}

func (n *dirNode) Unlink(name string, context *fuse.Context) fuse.Status {
	n.Lock()
	defer n.Unlock()
	if context.Uid != fs.uid || n.mode&0200 == 0 {
		return fuse.EACCES
	}
	cinode := n.Inode().Children()[name]
	if cinode == nil {
		return fuse.ENOENT
	}
	cnode := cinode.FsNode()
	switch child := cnode.(type) {
	case (*docDirNode):
		// XXX: POSIX says that EPERM should be returned, but Linux returns
		// EISDIR according to unlink(2). What to do?
		return fuse.EPERM
	case (*dirNode):
		return fuse.EPERM
	case (*fileNode):
		if child.mode&0200 == 0 {
			return fuse.EPERM
		}
		child.Lock()
		defer child.Unlock()
		if child.refcount == 0 {
			err := srv.Files.Delete(child.id).Do()
			if err != nil {
				log.Print(err)
				return fuse.EIO
			}
		} else {
			child.toDelete = true
		}
		n.Inode().RmChild(name)
	}
	n.setAtime(time.Now())
	return fuse.OK
}
