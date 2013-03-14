package main

import (
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
	dlurl    string
	hasSize  bool
	mode     uint32
	modTime  time.Time
	name     string
	size     uint64
	reader   io.ReadCloser
	refcount int
	fuse.DefaultFsNode
	sync.Mutex
}

func (n *docNode) GetAttr(out *fuse.Attr, file fuse.File, context *fuse.Context) fuse.Status {
	n.Lock()
	defer n.Unlock()
	if n == nil {
		return fuse.ENOENT
	}
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
	out.Mtime = uint64(n.modTime.Unix())
	out.Owner.Uid = fs.uid
	out.Owner.Gid = fs.gid
	out.Mode = n.mode
	out.Size = n.size
	return fuse.OK
}

func (n *docNode) Open(flags uint32, context *fuse.Context) (fuse.File, fuse.Status) {
	n.Lock()
	defer n.Unlock()
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
	return f, fuse.OK
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
	id      string
	mode    uint32
	modTime time.Time
	name    string
	fuse.DefaultFsNode
}

func newDocDirNode(file *driveFile) *docDirNode {
	t, err := time.Parse(time.RFC3339, file.ModifiedDate)
	if err != nil {
		t = time.Unix(0, 0)
		log.Println(file.Title, err)
	}
	n := &docDirNode{id: file.Id, modTime: t, mode: fuse.S_IFDIR | 0500, name: file.Title}
	_ = fs.root.Inode().New(true, n)
	for mime, link := range file.ExportLinks {
		if ext := mimeToExt[mime]; ext != "" {
			c := &docNode{dlurl: link, mode: fuse.S_IFREG | 0400,
				modTime: n.modTime, name: n.name + ext}
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
	out.Mtime = uint64(n.modTime.Unix())
	out.Owner.Uid = uint32(os.Getuid())
	out.Owner.Gid = uint32(os.Getgid())
	out.Mode = n.mode
	return fuse.OK
}

func (n *docDirNode) Name() string {
	return n.name
}
