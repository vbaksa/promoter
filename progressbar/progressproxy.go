package progressbar

import (
	"io"
)

// PassThru wraps an existing io.Reader.
//
// It simply forwards the Read() call, while displaying
// the results from individual calls to it.
type PassThru struct {
	io.ReadCloser
	Total *chan int64 // Total # of bytes transferred
}

// Read 'overrides' the underlying io.ReadCloser's Read method.
// This is the one that will be called by io.Copy(). We simply
// use it to keep track of byte counts and then forward the call.
func (pt *PassThru) Read(p []byte) (int, error) {

	n, err := pt.ReadCloser.Read(p)

	*pt.Total <- int64(n)

	return n, err
}

//Close overrides underlying io.ReadCloser Close method
func (pt *PassThru) Close() error {
	err := pt.ReadCloser.Close()
	return err
}
