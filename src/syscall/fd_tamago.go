// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// File descriptor support for tamago (adapted from fd_nacl.go).

package syscall

import (
	"sync"
)

// files is the table indexed by a file descriptor.
var files struct {
	sync.RWMutex
	tab []*file
}

// A file is an open file, something with a file descriptor.
// A particular *file may appear in files multiple times, due to use of Dup or Dup2.
type file struct {
	fdref int      // uses in files.tab
	impl  fileImpl // underlying implementation
}

// A fileImpl is the implementation of something that can be a file.
type fileImpl interface {
	// Standard operations.
	// These can be called concurrently from multiple goroutines.
	stat(*Stat_t) error
	read([]byte) (int, error)
	write([]byte) (int, error)
	seek(int64, int) (int64, error)
	pread([]byte, int64) (int, error)
	pwrite([]byte, int64) (int, error)

	// Close is called when the last reference to a *file is removed
	// from the file descriptor table. It may be called concurrently
	// with active operations such as blocked read or write calls.
	close() error
}

// newFD adds impl to the file descriptor table,
// returning the new file descriptor.
// Like Unix, it uses the lowest available descriptor.
func newFD(impl fileImpl) int {
	files.Lock()
	defer files.Unlock()
	f := &file{impl: impl, fdref: 1}
	for fd, oldf := range files.tab {
		if oldf == nil {
			files.tab[fd] = f
			return fd
		}
	}
	fd := len(files.tab)
	files.tab = append(files.tab, f)
	return fd
}

// Reserve stdin, stdout, stderr descriptors to never allocate them with this
// API and avoid conflicts.
func init() {
	newFD(&pipeFile{})
	newFD(&pipeFile{})
	newFD(&pipeFile{})
}

// fdToFile retrieves the *file corresponding to a file descriptor.
func fdToFile(fd int) (*file, error) {
	files.Lock()
	defer files.Unlock()
	if fd < 0 || fd >= len(files.tab) || files.tab[fd] == nil {
		return nil, EBADF
	}
	return files.tab[fd], nil
}

func Close(fd int) error {
	files.Lock()
	if fd < 0 || fd >= len(files.tab) || files.tab[fd] == nil {
		files.Unlock()
		return EBADF
	}
	f := files.tab[fd]
	files.tab[fd] = nil
	f.fdref--
	fdref := f.fdref
	files.Unlock()
	if fdref > 0 {
		return nil
	}
	return f.impl.close()
}

func CloseOnExec(fd int) {
	// nothing to do - no exec
}

func Dup(fd int) (int, error) {
	files.Lock()
	defer files.Unlock()
	if fd < 0 || fd >= len(files.tab) || files.tab[fd] == nil {
		return -1, EBADF
	}
	f := files.tab[fd]
	f.fdref++
	for newfd, oldf := range files.tab {
		if oldf == nil {
			files.tab[newfd] = f
			return newfd, nil
		}
	}
	newfd := len(files.tab)
	files.tab = append(files.tab, f)
	return newfd, nil
}

func Dup2(fd, newfd int) error {
	files.Lock()
	if fd < 0 || fd >= len(files.tab) || files.tab[fd] == nil || newfd < 0 || newfd >= len(files.tab)+100 {
		files.Unlock()
		return EBADF
	}
	f := files.tab[fd]
	f.fdref++
	for cap(files.tab) <= newfd {
		files.tab = append(files.tab[:cap(files.tab)], nil)
	}
	oldf := files.tab[newfd]
	var oldfdref int
	if oldf != nil {
		oldf.fdref--
		oldfdref = oldf.fdref
	}
	files.tab[newfd] = f
	files.Unlock()
	if oldf != nil {
		if oldfdref == 0 {
			oldf.impl.close()
		}
	}
	return nil
}

func Fstat(fd int, st *Stat_t) error {
	f, err := fdToFile(fd)
	if err != nil {
		return err
	}
	return f.impl.stat(st)
}

func Read(fd int, b []byte) (int, error) {
	f, err := fdToFile(fd)
	if err != nil {
		return 0, err
	}
	return f.impl.read(b)
}

var zerobuf [0]byte

func Write(fd int, b []byte) (int, error) {
	if fd == 1 || fd == 2 {
		return write(fd, b)
	}

	if b == nil {
		// avoid nil in syscalls.
		b = zerobuf[:]
	}
	f, err := fdToFile(fd)
	if err != nil {
		return 0, err
	}
	return f.impl.write(b)
}

func Pread(fd int, b []byte, offset int64) (int, error) {
	f, err := fdToFile(fd)
	if err != nil {
		return 0, err
	}
	return f.impl.pread(b, offset)
}

func Pwrite(fd int, b []byte, offset int64) (int, error) {
	f, err := fdToFile(fd)
	if err != nil {
		return 0, err
	}
	return f.impl.pwrite(b, offset)
}

func Seek(fd int, offset int64, whence int) (int64, error) {
	f, err := fdToFile(fd)
	if err != nil {
		return 0, err
	}
	return f.impl.seek(offset, whence)
}

// defaulFileImpl implements fileImpl.
// It can be embedded to complete a partial fileImpl implementation.
type defaultFileImpl struct{}

func (*defaultFileImpl) close() error                      { return nil }
func (*defaultFileImpl) stat(*Stat_t) error                { return ENOSYS }
func (*defaultFileImpl) read([]byte) (int, error)          { return 0, ENOSYS }
func (*defaultFileImpl) write([]byte) (int, error)         { return 0, ENOSYS }
func (*defaultFileImpl) seek(int64, int) (int64, error)    { return 0, ENOSYS }
func (*defaultFileImpl) pread([]byte, int64) (int, error)  { return 0, ENOSYS }
func (*defaultFileImpl) pwrite([]byte, int64) (int, error) { return 0, ENOSYS }

// A pipeFile is an in-memory implementation of a pipe.
// The byteq implementation is in net_tamago.go.
type pipeFile struct {
	defaultFileImpl
	rd *byteq
	wr *byteq
}

func (f *pipeFile) close() error {
	if f.rd != nil {
		f.rd.close()
	}
	if f.wr != nil {
		f.wr.close()
	}
	return nil
}

func (f *pipeFile) read(b []byte) (int, error) {
	if f.rd == nil {
		return 0, EINVAL
	}
	n, err := f.rd.read(b, 0)
	if err == EAGAIN {
		err = nil
	}
	return n, err
}

func (f *pipeFile) write(b []byte) (int, error) {
	if f.wr == nil {
		return 0, EINVAL
	}
	n, err := f.wr.write(b, 0)
	if err == EAGAIN {
		err = EPIPE
	}
	return n, err
}

func Pipe(fd []int) error {
	q := newByteq()
	fd[0] = newFD(&pipeFile{rd: q})
	fd[1] = newFD(&pipeFile{wr: q})
	return nil
}
