package main

import (
	"code.google.com/p/google-api-go-client/drive/v2"
	"github.com/hanwen/go-fuse/fuse"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

var mimeToExt = map[string]string{
	"text/plain":                              ".txt",
	"text/html":                               ".html",
	"application/rtf":                         ".rtf",
	"application/vnd.oasis.opendocument.text": ".odt",
	"application/pdf":                         ".pdf",
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": ".docx",
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":       ".xlsx",
	"application/x-vnd.oasis.opendocument.spreadsheet":                        ".ods",
	"image/jpeg":    ".jpg",
	"image/png":     ".png",
	"image/svg+xml": ".svg",
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": ".pptx",
}

type docNode struct {
	data     []byte
	dir      *docDirNode
	dlurl    string
	hasSize  bool
	mode     uint32
	name     string
	size     uint64
	reader   io.ReadCloser
	refcount int
	fuse.DefaultFsNode
	sync.Mutex
}

func (n *docNode) GetAttr(out *fuse.Attr, file fuse.File, context *fuse.Context) fuse.Status {
	if n == nil {
		return fuse.ENOENT
	}
	n.Lock()
	defer n.Unlock()
	n.dir.RLock()
	defer n.dir.RUnlock()
	if !n.hasSize {
		resp, err := transport.Client().Head(n.dlurl)
		if err != nil {
			log.Print(err)
			return fuse.EIO
		}
		resp.Body.Close()
		n.size = uint64(resp.ContentLength)
		n.hasSize = true
	}
	out.SetTimes(&n.dir.atime, &n.dir.mtime, nil)
	out.Owner.Uid = fs.uid
	out.Owner.Gid = fs.gid
	out.Mode = n.mode
	out.Size = n.size
	return fuse.OK
}

func (n *docNode) Open(flags uint32, context *fuse.Context) (fuse.File, fuse.Status) {
	n.Lock()
	defer n.Unlock()
	n.dir.Lock()
	defer n.dir.Unlock()
	if context.Uid != fs.uid || flags&fuse.O_ANYWRITE != 0 {
		return nil, fuse.EPERM
	}
	f := new(docFile)
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
	n.dir.setTimes(time.Now(), time.Time{})
	return f, fuse.OK
}

func (f *docNode) Utimens(file fuse.File, atimens, mtimens int64, context *fuse.Context) fuse.Status {
	return f.dir.Utimens(file, atimens, mtimens, context)
}

type docFile struct {
	fuse.DefaultFile
	node *docNode
}

func (f *docFile) Read(dest []byte, off int64) (fuse.ReadResult, fuse.Status) {
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

func (f *docFile) Release() {
	f.node.Lock()
	defer f.node.Unlock()
	f.node.refcount--
	if f.node.refcount == 0 {
		if f.node.reader != nil {
			f.node.reader.Close()
			f.node.reader = nil
		}
		f.node.data = nil
	}
}

type docDirNode struct {
	atime time.Time
	id    string
	mode  uint32
	mtime time.Time
	name  string
	fuse.DefaultFsNode
	sync.RWMutex
}

func newDocDirNode(file *driveFile) *docDirNode {
	var err error

	n := new(docDirNode)
	_ = fs.root.Inode().New(true, n)
	n.id = file.Id
	n.mode = fuse.S_IFDIR | 0500
	if file.Editable {
		n.mode |= 0200
	}
	n.mtime, err = time.Parse(time.RFC3339, file.ModifiedDate)
	if err != nil {
		n.mtime = time.Unix(0, 0)
		log.Println(file.Title, err)
	}
	n.name = file.Title
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
	for mime, link := range file.ExportLinks {
		if ext := mimeToExt[mime]; ext != "" {
			c := &docNode{dir: n, dlurl: link, mode: fuse.S_IFREG | 0400,
				name: n.name + ext}
			_ = fs.root.Inode().New(false, c)
			n.Inode().AddChild(n.name+ext, c.Inode())
		}
	}
	return n
}

func (n *docDirNode) GetAttr(out *fuse.Attr, file fuse.File, context *fuse.Context) fuse.Status {
	if n == nil {
		return fuse.ENOENT
	}
	n.RLock()
	defer n.RUnlock()
	out.SetTimes(&n.atime, &n.mtime, nil)
	out.Owner.Uid = uint32(os.Getuid())
	out.Owner.Gid = uint32(os.Getgid())
	out.Mode = n.mode
	return fuse.OK
}

func (n *docDirNode) Name() string {
	return n.name
}

func (n *docDirNode) Utimens(file fuse.File, atimens, mtimens int64, context *fuse.Context) fuse.Status {
	var atime, mtime time.Time
	if atimens > 0 {
		atime = time.Unix(atimens/1e9, atimens%1e9)
	}
	if mtimens > 0 {
		mtime = time.Unix(mtimens/1e9, mtimens%1e9)
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

// n must already be locked for writing
func (n *docDirNode) setTimes(atime, mtime time.Time) error {
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
