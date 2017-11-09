package vfs

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/miseyu/rclone/fs"
)

// WriteFileHandle is an open for write handle on a File
type WriteFileHandle struct {
	baseHandle
	mu          sync.Mutex
	closed      bool // set if handle has been closed
	remote      string
	pipeWriter  *io.PipeWriter
	o           fs.Object
	result      chan error
	file        *File
	writeCalled bool // set the first time Write() is called
	offset      int64
}

// Check interfaces
var (
	_ io.Writer   = (*WriteFileHandle)(nil)
	_ io.WriterAt = (*WriteFileHandle)(nil)
	_ io.Closer   = (*WriteFileHandle)(nil)
)

func newWriteFileHandle(d *Dir, f *File, src fs.ObjectInfo) (*WriteFileHandle, error) {
	fh := &WriteFileHandle{
		remote: src.Remote(),
		result: make(chan error, 1),
		file:   f,
	}
	var pipeReader *io.PipeReader
	pipeReader, fh.pipeWriter = io.Pipe()
	go func() {
		// NB Rcat deals with Stats.Transferring etc
		o, err := fs.Rcat(d.f, src.Remote(), pipeReader, time.Now())
		if err != nil {
			fs.Errorf(fh.remote, "WriteFileHandle.New Rcat failed: %v", err)
		}
		// Close the pipeReader so the pipeWriter fails with ErrClosedPipe
		_ = pipeReader.Close()
		fh.o = o
		fh.result <- err
	}()
	fh.file.addWriters(1)
	fh.file.setSize(0)
	return fh, nil
}

// String converts it to printable
func (fh *WriteFileHandle) String() string {
	if fh == nil {
		return "<nil *WriteFileHandle>"
	}
	fh.mu.Lock()
	defer fh.mu.Unlock()
	if fh.file == nil {
		return "<nil *WriteFileHandle.file>"
	}
	return fh.file.String() + " (w)"
}

// Node returns the Node assocuated with this - satisfies Noder interface
func (fh *WriteFileHandle) Node() Node {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	return fh.file
}

// WriteAt writes len(p) bytes from p to the underlying data stream at offset
// off. It returns the number of bytes written from p (0 <= n <= len(p)) and
// any error encountered that caused the write to stop early. WriteAt must
// return a non-nil error if it returns n < len(p).
//
// If WriteAt is writing to a destination with a seek offset, WriteAt should
// not affect nor be affected by the underlying seek offset.
//
// Clients of WriteAt can execute parallel WriteAt calls on the same
// destination if the ranges do not overlap.
//
// Implementations must not retain p.
func (fh *WriteFileHandle) WriteAt(p []byte, off int64) (n int, err error) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	return fh.writeAt(p, off)
}

// Implementatino of WriteAt - call with lock held
func (fh *WriteFileHandle) writeAt(p []byte, off int64) (n int, err error) {
	// fs.Debugf(fh.remote, "WriteFileHandle.Write len=%d", len(p))
	if fh.closed {
		fs.Errorf(fh.remote, "WriteFileHandle.Write error: %v", EBADF)
		return 0, ECLOSED
	}
	if fh.offset != off {
		fs.Errorf(fh.remote, "WriteFileHandle.Write can't seek in file")
		return 0, ESPIPE
	}
	fh.writeCalled = true
	n, err = fh.pipeWriter.Write(p)
	fh.offset += int64(n)
	fh.file.setSize(fh.offset)
	if err != nil {
		fs.Errorf(fh.remote, "WriteFileHandle.Write error: %v", err)
		return 0, err
	}
	// fs.Debugf(fh.remote, "WriteFileHandle.Write OK (%d bytes written)", n)
	return n, nil
}

// Write writes len(p) bytes from p to the underlying data stream. It returns
// the number of bytes written from p (0 <= n <= len(p)) and any error
// encountered that caused the write to stop early. Write must return a non-nil
// error if it returns n < len(p). Write must not modify the slice data, even
// temporarily.
//
// Implementations must not retain p.
func (fh *WriteFileHandle) Write(p []byte) (n int, err error) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	// Since we can't seek, just call WriteAt with the current offset
	return fh.writeAt(p, fh.offset)
}

// Offset returns the offset of the file pointer
func (fh *WriteFileHandle) Offset() (offset int64) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	return fh.offset
}

// close the file handle returning EBADF if it has been
// closed already.
//
// Must be called with fh.mu held
func (fh *WriteFileHandle) close() error {
	if fh.closed {
		return ECLOSED
	}
	fh.closed = true
	fh.file.addWriters(-1)
	writeCloseErr := fh.pipeWriter.Close()
	err := <-fh.result
	if err == nil {
		fh.file.setObject(fh.o)
		err = writeCloseErr
	}
	return err
}

// Close closes the file
func (fh *WriteFileHandle) Close() error {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	return fh.close()
}

// Flush is called on each close() of a file descriptor. So if a
// filesystem wants to return write errors in close() and the file has
// cached dirty data, this is a good place to write back data and
// return any errors. Since many applications ignore close() errors
// this is not always useful.
//
// NOTE: The flush() method may be called more than once for each
// open(). This happens if more than one file descriptor refers to an
// opened file due to dup(), dup2() or fork() calls. It is not
// possible to determine if a flush is final, so each flush should be
// treated equally. Multiple write-flush sequences are relatively
// rare, so this shouldn't be a problem.
//
// Filesystems shouldn't assume that flush will always be called after
// some writes, or that if will be called at all.
func (fh *WriteFileHandle) Flush() error {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	if fh.closed {
		fs.Debugf(fh.remote, "WriteFileHandle.Flush nothing to do")
		return nil
	}
	// fs.Debugf(fh.remote, "WriteFileHandle.Flush")
	// If Write hasn't been called then ignore the Flush - Release
	// will pick it up
	if !fh.writeCalled {
		fs.Debugf(fh.remote, "WriteFileHandle.Flush ignoring flush on unwritten handle")
		return nil

	}
	err := fh.close()
	if err != nil {
		fs.Errorf(fh.remote, "WriteFileHandle.Flush error: %v", err)
	} else {
		// fs.Debugf(fh.remote, "WriteFileHandle.Flush OK")
	}
	return err
}

// Release is called when we are finished with the file handle
//
// It isn't called directly from userspace so the error is ignored by
// the kernel
func (fh *WriteFileHandle) Release() error {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	if fh.closed {
		fs.Debugf(fh.remote, "WriteFileHandle.Release nothing to do")
		return nil
	}
	fs.Debugf(fh.remote, "WriteFileHandle.Release closing")
	err := fh.close()
	if err != nil {
		fs.Errorf(fh.remote, "WriteFileHandle.Release error: %v", err)
	} else {
		// fs.Debugf(fh.remote, "WriteFileHandle.Release OK")
	}
	return err
}

// Stat returns info about the file
func (fh *WriteFileHandle) Stat() (os.FileInfo, error) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	return fh.file, nil
}
