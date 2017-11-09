package vfs

import (
	"os"
	"testing"

	"github.com/miseyu/rclone/fs"
	"github.com/miseyu/rclone/fstest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Open a file for write
func writeHandleCreate(t *testing.T, r *fstest.Run) (*VFS, *WriteFileHandle) {
	vfs := New(r.Fremote, nil)

	h, err := vfs.OpenFile("file1", os.O_WRONLY|os.O_CREATE, 0777)
	require.NoError(t, err)
	fh, ok := h.(*WriteFileHandle)
	require.True(t, ok)

	return vfs, fh
}

func TestWriteFileHandleMethods(t *testing.T) {
	r := fstest.NewRun(t)
	defer r.Finalise()
	vfs, fh := writeHandleCreate(t, r)

	// String
	assert.Equal(t, "file1 (w)", fh.String())
	assert.Equal(t, "<nil *WriteFileHandle>", (*WriteFileHandle)(nil).String())
	assert.Equal(t, "<nil *WriteFileHandle.file>", new(WriteFileHandle).String())

	// Node
	node := fh.Node()
	assert.Equal(t, "file1", node.Name())

	// Offset #1
	assert.Equal(t, int64(0), fh.Offset())
	assert.Equal(t, int64(0), node.Size())

	// Write (smoke test only since heavy lifting done in WriteAt)
	n, err := fh.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	// Offset #2
	assert.Equal(t, int64(5), fh.Offset())
	assert.Equal(t, int64(5), node.Size())

	// Stat
	var fi os.FileInfo
	fi, err = fh.Stat()
	assert.NoError(t, err)
	assert.Equal(t, int64(5), fi.Size())
	assert.Equal(t, "file1", fi.Name())

	// Close
	assert.NoError(t, fh.Close())

	// Check double close
	err = fh.Close()
	assert.Equal(t, ECLOSED, err)

	// check vfs
	root, err := vfs.Root()
	checkListing(t, root, []string{"file1,5,false"})

	// check the underlying r.Fremote but not the modtime
	file1 := fstest.NewItem("file1", "hello", t1)
	fstest.CheckListingWithPrecision(t, r.Fremote, []fstest.Item{file1}, []string{}, fs.ModTimeNotSupported)
}

func TestWriteFileHandleWriteAt(t *testing.T) {
	r := fstest.NewRun(t)
	defer r.Finalise()
	vfs, fh := writeHandleCreate(t, r)

	// Preconditions
	assert.Equal(t, int64(0), fh.offset)
	assert.False(t, fh.writeCalled)

	// Write the data
	n, err := fh.WriteAt([]byte("hello"), 0)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	// After write
	assert.Equal(t, int64(5), fh.offset)
	assert.True(t, fh.writeCalled)

	// Check can't seek
	n, err = fh.WriteAt([]byte("hello"), 100)
	assert.Equal(t, ESPIPE, err)
	assert.Equal(t, 0, n)

	// Write more data
	n, err = fh.WriteAt([]byte(" world"), 5)
	assert.NoError(t, err)
	assert.Equal(t, 6, n)

	// Close
	assert.NoError(t, fh.Close())

	// Check can't write on closed handle
	n, err = fh.WriteAt([]byte("hello"), 0)
	assert.Equal(t, ECLOSED, err)
	assert.Equal(t, 0, n)

	// check vfs
	root, err := vfs.Root()
	checkListing(t, root, []string{"file1,11,false"})

	// check the underlying r.Fremote but not the modtime
	file1 := fstest.NewItem("file1", "hello world", t1)
	fstest.CheckListingWithPrecision(t, r.Fremote, []fstest.Item{file1}, []string{}, fs.ModTimeNotSupported)
}

func TestWriteFileHandleFlush(t *testing.T) {
	r := fstest.NewRun(t)
	defer r.Finalise()
	_, fh := writeHandleCreate(t, r)

	// Check Flush does nothing if write not called
	err := fh.Flush()
	assert.NoError(t, err)
	assert.False(t, fh.closed)

	// Write some data
	n, err := fh.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	// Check Flush closes file if write called
	err = fh.Flush()
	assert.NoError(t, err)
	assert.True(t, fh.closed)

	// Check flush does nothing if called again
	err = fh.Flush()
	assert.NoError(t, err)
	assert.True(t, fh.closed)
}

func TestWriteFileHandleRelease(t *testing.T) {
	r := fstest.NewRun(t)
	defer r.Finalise()
	_, fh := writeHandleCreate(t, r)

	// Check Release closes file
	err := fh.Release()
	assert.NoError(t, err)
	assert.True(t, fh.closed)

	// Check Release does nothing if called again
	err = fh.Release()
	assert.NoError(t, err)
	assert.True(t, fh.closed)
}
