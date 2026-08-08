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
	w          http.ResponseWriter
	gz         *gzip.Writer
	accepted   bool // true if the client accepts gzip (Accept-Encoding header)
	wrote      bool // true once the status line has been sent
	compressed bool // true if the body is being written through the gzip writer
}

func newGzipResponseWriter(w http.ResponseWriter, accepted bool) *gzipResponseWriter {
	return &gzipResponseWriter{
		w:        w,
		accepted: accepted,
	}
}

// Header returns the underlying ResponseWriter's header map.
func (g *gzipResponseWriter) Header() http.Header {
	return g.w.Header()
}

func (g *gzipResponseWriter) WriteHeader(statusCode int) {
	if g.wrote {
		return
	}
	g.wrote = true

	if g.accepted && statusCode < 300 && shouldCompress(g.w.Header().Get("Content-Type")) {
		g.compressed = true
		g.gz = gzip.NewWriter(g.w)
		g.w.Header().Set("Content-Encoding", "gzip")
		g.w.Header().Del("Content-Length")
	}
	g.w.WriteHeader(statusCode)
}

func (g *gzipResponseWriter) Write(p []byte) (int, error) {
	// If WriteHeader was never called by the handler, default to 200 OK.
	if !g.wrote {
		g.WriteHeader(http.StatusOK)
	}
	if g.compressed {
		return g.gz.Write(p)
	}
	return g.w.Write(p)
}

func (g *gzipResponseWriter) Flush() {
	if !g.wrote {
		g.WriteHeader(http.StatusOK)
	}
	if g.compressed {
		g.gz.Flush()
	}
	if f, ok := g.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (g *gzipResponseWriter) Close() error {
	if g.compressed {
		return g.gz.Close()
	}
	return nil
}

// gzipReader is a wrapper around io.ReadCloser that decompresses the response body.
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

func (g gzipReader) Read(p []byte) (n int, err error) {
	return g.zr.Read(p)
}

func (g *gzipReader) Close() error {
	if err := g.r.Close(); err != nil {
		return err
	}
	return g.zr.Close()
}

func shouldCompress(contentType string) bool {
	for _, ct := range compressibleContentTypes {
		if strings.HasPrefix(contentType, ct) {
			return true
		}
	}
	return false
}

func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decompress the request body if it was sent gzip-encoded.
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := newGzipReader(r.Body)
			if err != nil {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		// Wrap the writer so the response can be compressed if appropriate.
		// The final decision is made at WriteHeader time, based on the response
		// Content-Type and the client's Accept-Encoding header.
		cw := newGzipResponseWriter(w, strings.Contains(r.Header.Get("Accept-Encoding"), "gzip"))
		next.ServeHTTP(cw, r)
		cw.Close()
	})
}
