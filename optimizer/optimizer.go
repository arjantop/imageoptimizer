package optimizer

import (
	"context"
	"log"
	"net/http"
	"os"
	"sort"
)

type Name string

type ImageDescription struct {
	Optimizer Name
	Path      string
	MimeType  string
	Size      int64
}

type ImageOptimizer interface {
	CanOptimize(mimeType string, acceptedTypes []string) bool
	Optimize(ctx context.Context, sourcePath string) (*ImageDescription, error)
}

type bySize []*ImageDescription

func (s bySize) Len() int {
	return len(s)
}

func (s bySize) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s bySize) Less(i, j int) bool {
	return s[i].Size < s[j].Size
}

func Optimize(ctx context.Context, optimizers []ImageOptimizer, acceptedTypes []string, sourcePath string) (*ImageDescription, error) {
	header := make([]byte, 512)
	file, err := os.Open(sourcePath)
	if err != nil {
		return nil, err
	}
	_, err = file.Read(header)
	if err != nil {
		return nil, err
	}
	originalStat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	originalType := http.DetectContentType(header)
	log.Printf("Detected file type: %s", originalType)
	originalSize := originalStat.Size()

	images := make([]*ImageDescription, 0, len(optimizers)+1)
	images = append(images, &ImageDescription{
		Optimizer: Name("original"),
		Path:      sourcePath,
		MimeType:  originalType,
		Size:      originalSize,
	})

	for _, opt := range optimizers {
		if opt.CanOptimize(originalType, acceptedTypes) {
			image, err := opt.Optimize(ctx, sourcePath)
			if err != nil {
				log.Println("error optimizing image: " + err.Error())
				continue
			}
			images = append(images, image)
		}
	}

	sort.Sort(bySize(images))
	for _, image := range images {
		log.Printf("optimizer=%s size=%d type=%s", image.Optimizer, image.Size, image.MimeType)
	}
	return images[0], nil
}
