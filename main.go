package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/arjantop/imageoptimizer/optimizer"
)

type ImageDescription struct {
	Optimizer string
	Path      string
	MimeType  string
	Size      int64
}

func reportError(w http.ResponseWriter, msg string, err error) {
	w.WriteHeader(http.StatusInternalServerError)
	log.Printf("%s err=%s", msg, err)
}

func parseAcceptedTypes(acceptHeader string) []string {
	acceptedTypes := make([]string, 0, 1)
	for _, part := range strings.Split(acceptHeader, ",") {
		acceptedType := part
		if strings.Contains(part, ";") {
			acceptedType = strings.SplitN(part, ";", 2)[0]
		}
		acceptedTypes = append(acceptedTypes, acceptedType)
	}
	return acceptedTypes
}

var baseUrl = flag.String("baseurl", "", "Base url to which proxied requests are appended")

func main() {
	flag.Parse()

	if _, err := url.Parse(*baseUrl); *baseUrl == "" || err != nil {
		log.Fatalf("Invalid base url: %s", *baseUrl)
	}

	optimizers := []optimizer.ImageOptimizer{
		&optimizer.WebpLosslessOptimizer{
			Args: []string{"-z", "9"},
		},
		optimizer.NewWebpLossyPngOptimizer(0.998),
		optimizer.NewWebpLossyJpegOptimizer(0.995),
		&optimizer.OptipngOptimizer{
			Args: []string{"-strip", "all"},
		},
		&optimizer.MozjpegOptimizer{
			Args: []string{"-copy", "none", "-optimize"},
		},
		optimizer.NewMozjpegPngLossyOptimizer(0.997),
		optimizer.NewMozjpegLossyOptimizer(0.994),
	}

	client := &http.Client{}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		acceptedTypes := parseAcceptedTypes(r.Header.Get("Accept"))

		requestUrl, err := url.ParseRequestURI(r.RequestURI)
		if err != nil {
			http.Error(w, "Invalid url", http.StatusBadRequest)
			return
		}

		hidpi := strings.Contains(requestUrl.Path, "@2x.")

		log.Printf("Proxying: %s (hidpi=%t)", requestUrl.Path+"?"+requestUrl.RawQuery, hidpi)

		req, err := http.NewRequest(http.MethodGet, *baseUrl+requestUrl.Path+"?"+requestUrl.RawQuery, nil)
		if err != nil {
			log.Fatal("Invalid request")
		}
		for key, vals := range r.Header {
			for _, val := range vals {
				req.Header.Set(key, val)
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			reportError(w, "Call failed", err)
			return
		}
		defer resp.Body.Close()

		if !optimizer.CanOptimize(optimizers, resp.Header.Get("Content-Type"), acceptedTypes) {
			for key, vals := range resp.Header {
				for _, val := range vals {
					w.Header().Add(key, val)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, err = io.Copy(w, resp.Body)
			if err != nil {
				reportError(w, "Could not copy data to client", err)
				return
			}
			return
		}

		tempFile, err := ioutil.TempFile(os.TempDir(), strings.Replace(r.RequestURI, "/", "", -1))
		if err != nil {
			reportError(w, "Could not create temp file", err)
			return
		}
		defer tempFile.Close()

		_, err = io.Copy(tempFile, resp.Body)
		if err != nil {
			reportError(w, "Could not copy data to temp file", err)
			return
		}

		optimizedImage, err := optimizer.Optimize(r.Context(), optimizers, optimizer.OptimizeParams{
			AcceptedTypes: acceptedTypes,
			SourcePath:    tempFile.Name(),
			Hidpi:         hidpi,
		})
		if err != nil {
			reportError(w, "Could not optimize the file", err)
			return
		}

		log.Printf("Chosen optimizer: %s", optimizedImage.Optimizer)

		file, err := os.Open(optimizedImage.Path)
		if err != nil {
			reportError(w, "opening file", err)
			return
		}
		w.Header().Set("Content-Type", optimizedImage.MimeType)
		w.Header().Set("Content-Length", strconv.FormatInt(optimizedImage.Size, 10))
		w.WriteHeader(http.StatusOK)

		_, err = io.Copy(w, file)
		if err != nil {
			reportError(w, "reading file", err)
			return
		}
	})

	log.Fatal(http.ListenAndServe(":8888", nil))
}
