package domain

import "io"

type RandomAccessFile interface {
	io.Reader
	io.Writer
	io.Closer
	io.Seeker
	Truncate(size int64) error
}
