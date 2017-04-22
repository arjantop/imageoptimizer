package optimizer

import (
	"context"
	"log"
	"net/http"
	"os"
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

var DefaultPool = NewTaskPool()

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

	originalImage := &ImageDescription{
		Optimizer: Name("original"),
		Path:      sourcePath,
		MimeType:  originalType,
		Size:      originalSize,
	}

	suitableOptimizers := make([]ImageOptimizer, 0, len(optimizers))
	for _, opt := range optimizers {
		if opt.CanOptimize(originalType, acceptedTypes) {
			suitableOptimizers = append(suitableOptimizers, opt)
		}
	}

	if len(suitableOptimizers) == 0 {
		return nil, nil
	}

	return DefaultPool.Do(ctx, &Task{
		OriginalImage: originalImage,
		Optimizers:    suitableOptimizers,
	})
}

func CanOptimize(optimizers []ImageOptimizer, mimeType string, acceptedTyped []string) bool {
	for _, opt := range optimizers {
		if opt.CanOptimize(mimeType, acceptedTyped) {
			return true
		}
	}
	return false
}
