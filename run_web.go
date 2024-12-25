//go:build ignore

// ====================================================
// program that starts static server on web
//
// usage :
// 	go run run_web.go
//
// To be honest, if you do that, windows keep asking
// for network permission.
//
// So just compile it to binary.
// ====================================================

package main

import (
	"flag"
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	TargetFolder string
	Port         uint
)

func init() {
	flag.StringVar(&TargetFolder, "folder", "./web_build", "folder to serve")
	flag.UintVar(&Port, "port", 6969, "port")
}

func main() {
	flag.Parse()

	if Port > math.MaxUint16 {
		fmt.Fprintf(os.Stderr, "port %v is bigger than max port value\n", Port)
		os.Exit(1)
	}

	if !filepath.IsLocal(TargetFolder) {
		fmt.Fprintf(os.Stderr, "%s is not a local folder\n", TargetFolder)
		os.Exit(1)
	}

	fmt.Printf("serving %s\n", TargetFolder)
	fmt.Printf("listening to http://localhost:%v\n", Port)

	fs := http.FileServer(http.Dir(TargetFolder))
	err := http.ListenAndServe(fmt.Sprintf(":%v", Port), NoCache(fs))

	if err != nil {
		panic(err)
	}
}

// copied from https://stackoverflow.com/questions/33880343/go-webserver-dont-cache-files-using-timestamp

var epoch = time.Unix(0, 0).Format(time.RFC1123)

var noCacheHeaders = map[string]string{
	"Expires":         epoch,
	"Cache-Control":   "no-cache, private, max-age=0",
	"Pragma":          "no-cache",
	"X-Accel-Expires": "0",
}

var etagHeaders = []string{
	"ETag",
	"If-Modified-Since",
	"If-Match",
	"If-None-Match",
	"If-Range",
	"If-Unmodified-Since",
}

func NoCache(h http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Delete any ETag headers that may have been set
		for _, v := range etagHeaders {
			if r.Header.Get(v) != "" {
				r.Header.Del(v)
			}
		}

		// Set our NoCache headers
		for k, v := range noCacheHeaders {
			w.Header().Set(k, v)
		}

		h.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
