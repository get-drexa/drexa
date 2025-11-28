package ioext

import "io"

type LimitReadCloser struct {
	reader      io.ReadCloser
	limitReader io.Reader
}

func NewLimitReadCloser(reader io.ReadCloser, length int64) *LimitReadCloser {
	return &LimitReadCloser{
		reader:      reader,
		limitReader: io.LimitReader(reader, length),
	}
}

func (r *LimitReadCloser) Read(p []byte) (n int, err error) {
	return r.limitReader.Read(p)
}

func (r *LimitReadCloser) Close() error {
	return r.reader.Close()
}

