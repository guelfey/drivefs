package main

import (
	"code.google.com/p/google-api-go-client/drive/v2"
	"github.com/hanwen/go-fuse/fuse"
	"io"
	"log"
	"sync"
	"time"
)

type fileNode struct {
	atime    time.Time
	data     []byte
	dlurl    string
	mode     uint32
	mtime    time.Time
	name     string
	id       string
	reader   io.ReadCloser
	refcount int
	size     uint64
	toDelete bool
	fuse.DefaultFsNode
	sync.Mutex
}

func newFileNode(file *driveFile) *fileNode {
	var err error

	n := new(fileNode)
	_ = fs.root.Inode().New(false, n)
	n.id = file.Id
	n.name = file.Title
	n.size = uint64(file.FileSize)
	n.mode = fuse.S_IFREG | 0400
	if file.Editable {
		n.mode |= 0200
	}
	n.dlurl = file.DownloadUrl
	n.mtime, err = time.Parse(time.RFC3339Nano, file.ModifiedDate)
	if err != nil {
		n.mtime = time.Unix(0, 0)
		log.Println(n.name, err)
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

func (n *fileNode) GetAttr(out *fuse.Attr, file fuse.File, context *fuse.Context) fuse.Status {
	n.Lock()
	defer n.Unlock()
	if n == nil {
		return fuse.ENOENT
	}
	out.Size = n.size
	out.Mtime = uint64(n.mtime.Unix())
	out.Atime = uint64(n.atime.Unix())
	out.Owner.Uid = fs.uid
	out.Owner.Gid = fs.gid
	out.Mode = n.mode
	return fuse.OK
}

func (n *fileNode) Name() string {
	return n.name
}

func (n *fileNode) Open(flags uint32, context *fuse.Context) (fuse.File, fuse.Status) {
	n.Lock()
	defer n.Unlock()
	if context.Uid != fs.uid || (flags&fuse.O_ANYWRITE != 0 && n.mode&0200 == 0) {
		return nil, fuse.EPERM
	}
	f := new(file)
	f.node = n
	n.refcount++
	if n.reader == nil {
		resp, err := transport.Client().Get(n.dlurl)
		if err != nil {
			log.Print(err)
			return nil, fuse.EIO
		}
		n.reader = resp.Body
	}
	if n.data == nil {
		n.data = make([]byte, 0)
	}
	err := n.setAtime(time.Now())
	if err != nil {
		log.Print(err)
		return nil, fuse.EIO
	}
	return f, fuse.OK
}

// n must already be locked
func (n *fileNode) setAtime(t time.Time) error {
	n.atime = t
	f := &drive.File{LastViewedByMeDate: t.Format(time.RFC3339Nano)}
	_, err := srv.Files.Patch(n.id, f).UpdateViewedDate(false).Do()
	return err
}

type file struct {
	fuse.DefaultFile
	node *fileNode
}

func (f *file) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
	f.node.Lock()
	defer f.node.Unlock()
	if off+int64(len(dest)) >= int64(len(f.node.data)) {
		for {
			if f.node.reader == nil {
				break
			}
			diff := off + int64(len(dest)-len(f.node.data))
			if diff == 0 {
				break
			}
			newData := make([]byte, diff)

			n, err := f.node.reader.Read(newData)
			if err != nil {
				if err == io.EOF {
					f.node.reader.Close()
					f.node.reader = nil
				} else {
					log.Println("read error:", err)
					return nil, fuse.EIO
				}
			}
			f.node.data = append(f.node.data, newData[:n]...)
		}
	}
	if off < int64(len(f.node.data)) {
		n := copy(dest, f.node.data[off:])
		return &fuse.ReadResultData{dest[:n]}, fuse.OK
	}
	return &fuse.ReadResultData{[]byte{}}, fuse.OK
}

func (f *file) Release() {
	f.node.Lock()
	defer f.node.Unlock()
	f.node.refcount--
	if f.node.refcount == 0 {
		if f.node.reader != nil {
			f.node.reader.Close()
			f.node.reader = nil
		}
		f.node.data = nil
		if f.node.toDelete {
			err := srv.Files.Delete(f.node.id).Do()
			if err != nil {
				log.Print(err)
			}
		}
	}
}
