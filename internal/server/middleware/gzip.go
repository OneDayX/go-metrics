package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

var compressibleContentTypes = []string{
	"application/json",
	"text/html",
}

// gzipResponseWriter is a wrapper around http.ResponseWriter that compresses the response body.
type gzipResponseWriter struct {
	w     http.ResponseWriter
	gz    *gzip.Writer 
	wrote bool         // true once the status line has been sent
}

func newGzipResponseWriter(w http.ResponseWriter) *gzipResponseWriter {
	return &gzipResponseWriter{w: w}
}

func (g *gzipResponseWriter) Header() http.Header {
	return g.w.Header()
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	if g.wrote {
		return
	}
	g.wrote = true

	if statusCode < 300 && shouldCompress(g.w.Header().Get("Content-Type")) {
		g.gz = gzip.NewWriter(g.w)
		g.w.Header().Set("Content-Encoding", "gzip")
		g.w.Header().Add("Vary", "Accept-Encoding")
		g.w.Header().Del("Content-Length")
	}
	g.w.WriteHeader(statusCode)
}

func (g *gzipResponseWriter) Write(p []byte) (int, error) {
	// If WriteHeader was never called by the handler, default to 200 OK.
	if !g.wrote {
		g.WriteHeader(http.StatusOK)
	}
	if g.gz != nil {
		return g.gz.Write(p)
	}
	return g.w.Write(p)
}

func (g *gzipResponseWriter) Close() error {
	if g.gz != nil {
		return g.gz.Close()
	}
	return nil
}

// gzipReader is a wrapper around io.ReadCloser that decompresses the request body.
type gzipReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newGzipReader(r io.ReadCloser) (*gzipReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &gzipReader{
		r:  r,
		zr: zr,
	}, nil
}

func (g *gzipReader) Read(p []byte) (n int, err error) {
	return g.zr.Read(p)
}

func (g *gzipReader) Close() error {
	if err := g.r.Close(); err != nil {
		return err
	}
	return g.zr.Close()
}

func acceptsEncoding(header, encoding string) bool {
	for _, part := range strings.Split(header, ",") {
		enc := strings.TrimSpace(strings.Split(part, ";")[0])
		if enc == encoding || (encoding == "gzip" && enc == "x-gzip") {
			return true
		}
	}
	return false
}

func shouldCompress(contentType string) bool {
	for _, ct := range compressibleContentTypes {
		if strings.HasPrefix(contentType, ct) {
			return true
		}
	}
	return false
}

func decompressRequest(r *http.Request) error {
	if !acceptsEncoding(r.Header.Get("Content-Encoding"), "gzip") {
		return nil
	}

	gz, err := newGzipReader(r.Body)
	if err != nil {
		r.Body.Close()
		return err
	}

	r.Body = gz
	r.Header.Del("Content-Encoding")
	r.Header.Del("Content-Length")
	r.ContentLength = -1
	return nil
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := decompressRequest(r); err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if acceptsEncoding(r.Header.Get("Accept-Encoding"), "gzip") {
			cw := newGzipResponseWriter(w)
			next.ServeHTTP(cw, r)
			_ = cw.Close()
			return
		}

		next.ServeHTTP(w, r)
	})
}
