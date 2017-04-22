package main

import (
	"log"
	"net/http"
	"path"

	"strings"

	"os"

	"io"
	"strconv"

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

func main() {
	//optimizers := []optimizer.ImageOptimizer{
	//	&optimizer.WebpLosslessOptimizer{
	//		Args: []string{"-z", "9"},
	//	},
	//	&optimizer.WebpLossyOptimizer{
	//		Args: []string{"-q", "80"},
	//	},
	//	&optimizer.OptipngOptimizer{
	//		Args: []string{"-strip", "all"},
	//	},
	//	&optimizer.MozjpegOptimizer{
	//		Args: []string{"-copy", "none", "-optimize"},
	//	},
	//}

	optimizers := []optimizer.ImageOptimizer{
		&optimizer.MozjpegLosslessOptimizer{
			Args:    []string{},
			MinSsim: 0.993,
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		acceptedTypes := parseAcceptedTypes(r.Header.Get("Accept"))

		requestedFile := path.Join("images", r.RequestURI[1:])
		log.Println("Requested file: " + r.RequestURI)
		if _, err := os.Stat(requestedFile); err != nil {
			http.NotFound(w, r)
			return
		}

		optimizedImage, err := optimizer.Optimize(r.Context(), optimizers, acceptedTypes, requestedFile)
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
