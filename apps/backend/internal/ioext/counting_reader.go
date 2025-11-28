package ioext

import "io"

type CountingReader struct {
	reader io.Reader
	count  int64
}

func NewCountingReader(reader io.Reader) *CountingReader {
	return &CountingReader{reader: reader}
}

func (r *CountingReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.count += int64(n)
	return n, err
}

func (r *CountingReader) Count() int64 {
	return r.count
}

