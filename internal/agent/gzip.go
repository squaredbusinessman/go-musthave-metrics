package agent

import (
	"compress/gzip"
	"io"
	"net/http"
)

// CompressWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type CompressWriter struct {
	writer        http.ResponseWriter
	zipWriter     *gzip.Writer
	headerWritten bool
}

func NewCompressWriter(writer http.ResponseWriter) *CompressWriter {
	return &CompressWriter{
		writer:    writer,
		zipWriter: gzip.NewWriter(writer),
	}
}

func (cw *CompressWriter) Header() http.Header {
	return cw.writer.Header()
}

func (cw *CompressWriter) Write(data []byte) (int, error) {
	if !cw.headerWritten {
		cw.WriteHeader(http.StatusOK)
	}
	return cw.zipWriter.Write(data)
}

func (cw *CompressWriter) WriteHeader(statusCode int) {
	if !cw.headerWritten {
		if statusCode >= 200 && statusCode < 300 {
			cw.writer.Header().Set("Content-Encoding", "gzip")
		}
		cw.headerWritten = true
	}
	cw.writer.WriteHeader(statusCode)
}

func (cw *CompressWriter) Close() error {
	return cw.zipWriter.Close()
}

// CompressReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type CompressReader struct {
	reader    io.ReadCloser
	zipReader *gzip.Reader
}

func NewCompressReader(reader io.ReadCloser) (*CompressReader, error) {
	zipReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}

	return &CompressReader{
		reader:    reader,
		zipReader: zipReader,
	}, nil
}

func (cr *CompressReader) Read(data []byte) (n int, err error) {
	return cr.zipReader.Read(data)
}

func (cr *CompressReader) Close() error {
	if err := cr.reader.Close(); err != nil {
		return err
	}
	return cr.zipReader.Close()
}
