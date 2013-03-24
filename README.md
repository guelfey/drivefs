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

### Implemented features

<table>
	<tr>
		<th>FUSE method</th>
		<th>Normal files</th>
		<th>Normal directories</th>
		<th>Document files</th>
		<th>Document directories</th>
	</tr>
	<tr>
		<td>Access</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
	</tr>
	<tr>
		<td>Chmod</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
	</tr>
	<tr>
		<td>Chown</td>
		<td>-</td>
		<td>-</td>
		<td>-</td>
		<td>-</td>
	</tr>
	<tr>
		<td>GetAttr</td>
		<td>Yes</td>
		<td>Yes</td>
		<td>Yes</td>
		<td>Yes</td>
	</tr>
	<tr>
		<td>GetXAttr</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
	</tr>
	<tr>
		<td>Link</td>
		<td>No</td>
		<td>-</td>
		<td>-</td>
		<td>-</td>
	</tr>
	<tr>
		<td>ListXAttr</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
	</tr>
	<tr>
		<td>Mkdir</td>
		<td>-</td>
		<td>No</td>
		<td>-</td>
		<td>-</td>
	</tr>
	<tr>
		<td>Mknod</td>
		<td>-</td>
		<td>-</td>
		<td>-</td>
		<td>-</td>
	</tr>
	<tr>
		<td>Open</td>
		<td>Yes</td>
		<td>-</td>
		<td>Yes</td>
		<td>-</td>
	</tr>
	<tr>
		<td>OpenDir</td>
		<td>-</td>
		<td>Yes</td>
		<td>-</td>
		<td>Yes</td>
	</tr>
	<tr>
		<td>Read</td>
		<td>Yes</td>
		<td>-</td>
		<td>Yes</td>
		<td>-</td>
	</tr>
	<tr>
		<td>Readlink</td>
		<td>-</td>
		<td>-</td>
		<td>-</td>
		<td>-</td>
	</tr>
	<tr>
		<td>RemoveXAttr</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
	</tr>
	<tr>
		<td>Rename</td>
		<td>No</td>
		<td>No</td>
		<td>-</td>
		<td>No</td>
	</tr>
	<tr>
		<td>Rmdir</td>
		<td>-</td>
		<td>Yes</td>
		<td>-</td>
		<td>Yes</td>
	</tr>
	<tr>
		<td>SetXAttr</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
	</tr>
	<tr>
		<td>Symlink</td>
		<td>-</td>
		<td>-</td>
		<td>-</td>
		<td>-</td>
	</tr>
	<tr>
		<td>Truncate</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
		<td>No</td>
	</tr>
	<tr>
		<td>Unlink</td>
		<td>Yes</td>
		<td>-</td>
		<td>-</td>
		<td>-</td>
	</tr>
	<tr>
		<td>Utimens</td>
		<td>Yes</td>
		<td>Yes</td>
		<td>Yes</td>
		<td>Yes</td>
	</tr>
	<tr>
		<td>Write</td>
		<td>No</td>
		<td>-</td>
		<td>No</td>
		<td>-</td>
	</tr>
</table>

### To do

* Implement missing FUSE methods (see above).
	* Figure out how to represent permissions.
* Handle files with same name somehow.
* Make read buffering smarter.
* Moar performance!
	* Use gzip for some calls?

### License

Drivefs is licensed under a modified BSD license. See the LICENSE file for the
full text.
