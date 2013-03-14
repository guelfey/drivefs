Drivefs - FUSE filesystem for Google Drive
------------------------------------------

Drivefs lets you mount your Google Drive as a folder in your filesystem. It is
in a very early stage, but it supports reading both normal files and documents.

### Installation

You need to set up a Go environment first and add `$GOPATH/bin` to your `$PATH`.
Then run:

```
$ go get github.com/guelfey/drivefs
$ drivefs --init
```

You can now mount your Google Drive with `drivefs MOUNTPOINT` and unmount it
with `fusermount -u MOUNTPOINT`.

### License

Drivefs is licensed under a modified BSD license. See the LICENSE file for the
full text.
