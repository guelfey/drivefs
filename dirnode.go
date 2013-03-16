package main

import (
	"code.google.com/p/google-api-go-client/drive/v2"
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

func (n *dirNode) GetAttr(out *fuse.Attr, file fuse.File, context *fuse.Context) fuse.Status {
	n.RLock()
	defer n.RUnlock()
	if n == nil {
		return fuse.ENOENT
	}
	out.SetTimes(&n.atime, &n.mtime, nil)
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
	n.setTimes(time.Now(), time.Now())
	return fuse.OK
}

// n must already be locked for writing
func (n *dirNode) setTimes(atime, mtime time.Time) error {
	if atime.IsZero() && mtime.IsZero() {
		return nil
	}
	f := new(drive.File)
	if !atime.IsZero() {
		n.atime = atime
		f.LastViewedByMeDate = atime.Format(time.RFC3339Nano)
	}
	if !mtime.IsZero() {
		n.mtime = mtime
		f.ModifiedDate = mtime.Format(time.RFC3339Nano)
	}
	_, err := srv.Files.Patch(n.id, f).UpdateViewedDate(false).SetModifiedDate(true).Do()
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
	n.setTimes(time.Now(), time.Now())
	return fuse.OK
}

func (n *dirNode) Utimens(file fuse.File, atimens, mtimens int64, context *fuse.Context) fuse.Status {
	var atime, mtime time.Time
	if atimens > 0 {
		atime = time.Unix(atimens / 1e9, atimens % 1e9)
	}
	if mtimens > 0 {
		mtime = time.Unix(mtimens / 1e9, mtimens % 1e9)
	}
	n.Lock()
	err := n.setTimes(atime, mtime)
	n.Unlock()
	if err != nil {
		log.Print(err)
		return fuse.EIO
	}
	return fuse.OK
}
