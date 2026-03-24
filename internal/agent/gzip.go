package agent

import (
	"compress/gzip"
	"io"
	"net/http"
)

// CompressWriter - http.ResponseWriter с gzip-сжатием ответа.
type CompressWriter struct {
	writer        http.ResponseWriter
	zipWriter     *gzip.Writer
	headerWritten bool
}

// NewCompressWriter - создает gzip-обертку над ResponseWriter.
func NewCompressWriter(writer http.ResponseWriter) *CompressWriter {
	return &CompressWriter{
		writer:    writer,
		zipWriter: gzip.NewWriter(writer),
	}
}

// Header - возвращает заголовки исходного HTTP-ответа.
func (cw *CompressWriter) Header() http.Header {
	return cw.writer.Header()
}

// Write - пишет данные в gzip-поток ответа.
func (cw *CompressWriter) Write(data []byte) (int, error) {
	if !cw.headerWritten {
		cw.WriteHeader(http.StatusOK)
	}
	return cw.zipWriter.Write(data)
}

// WriteHeader - отправляет HTTP-статус и заголовок Content-Encoding.
func (cw *CompressWriter) WriteHeader(statusCode int) {
	if !cw.headerWritten {
		if statusCode >= 200 && statusCode < 300 {
			cw.writer.Header().Set("Content-Encoding", "gzip")
		}
		cw.headerWritten = true
	}
	cw.writer.WriteHeader(statusCode)
}

// Close - завершает gzip-поток ответа.
func (cw *CompressWriter) Close() error {
	return cw.zipWriter.Close()
}

// CompressReader - io.ReadCloser с прозрачной распаковкой gzip-запроса.
type CompressReader struct {
	reader    io.ReadCloser
	zipReader *gzip.Reader
}

// NewCompressReader - создает reader для gzip-сжатого тела запроса.
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

// Read - читает распакованные данные из gzip-потока.
func (cr *CompressReader) Read(data []byte) (n int, err error) {
	return cr.zipReader.Read(data)
}

// Close - закрывает исходный reader и gzip-обертку.
func (cr *CompressReader) Close() error {
	if err := cr.reader.Close(); err != nil {
		return err
	}
	return cr.zipReader.Close()
}
