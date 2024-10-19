package main

import (
	"bytes"
	"compress/gzip"
	"github.com/andybalholm/brotli"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
)

func NewProxyHandler(
	URL *url.URL,
	logChan chan Log,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Host = URL.Host
		r.URL.Host = URL.Host
		r.URL.Scheme = URL.Scheme

		var reqBody bytes.Buffer
		r.Body = io.NopCloser(io.TeeReader(r.Body, &reqBody))

		res := httptest.NewRecorder()
		httputil.NewSingleHostReverseProxy(URL).ServeHTTP(res, r)

		for k, v := range res.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(res.Code)
		w.Write(res.Body.Bytes())

		r.Body = io.NopCloser(bytes.NewReader(reqBody.Bytes()))

		if res.Header().Get("Content-Encoding") == "gzip" {
			reader, _ := gzip.NewReader(res.Body)
			data, _ := io.ReadAll(reader)
			reader.Close()
			res.Body = bytes.NewBuffer(data)
		} else if res.Header().Get("Content-Encoding") == "br" {
			reader := brotli.NewReader(res.Body)
			data, _ := io.ReadAll(reader)
			res.Body = bytes.NewBuffer(data)
		}

		logChan <- Log{
			URL:      *r.URL,
			Request:  *r,
			Response: *res.Result(),
		}
	})
}
