package optimizer

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
)

var _ ImageOptimizer = &WebpLosslessOptimizer{}

type WebpLosslessOptimizer struct {
	Args []string
}

func (o *WebpLosslessOptimizer) CanOptimize(mimeType string, acceptedTypes []string) bool {
	return mimeType == "image/png" && isFiletypeAccepted(acceptedTypes, []string{"image/webp"})
}

func (o *WebpLosslessOptimizer) Optimize(ctx context.Context, sourcePath string) (*ImageDescription, error) {
	outputPath := tempFilename(os.TempDir(), path.Base(sourcePath))
	args := []string{sourcePath, "-o", outputPath, "-lossless"}
	err := exec.CommandContext(ctx, "cwebp", append(args, o.Args...)...).Run()
	if err != nil {
		return nil, errors.New("transforming file with cwebp-lossless: " + err.Error())
	}

	fileStat, err := os.Stat(outputPath)
	if err != nil {
		return nil, err
	}

	return &ImageDescription{
		Optimizer: Name("cwebp-lossless"),
		Path:      outputPath,
		MimeType:  "image/webp",
		Size:      fileStat.Size(),
	}, nil
}

var _ ImageOptimizer = &WebpLossyOptimizer{}

type WebpLossyOptimizer struct {
	Args []string
}

func (o *WebpLossyOptimizer) CanOptimize(mimeType string, acceptedTypes []string) bool {
	return mimeType == "image/jpeg" && isFiletypeAccepted(acceptedTypes, []string{"image/webp"})
}

func (o *WebpLossyOptimizer) Optimize(ctx context.Context, sourcePath string) (*ImageDescription, error) {
	outputPath := tempFilename(os.TempDir(), path.Base(sourcePath))
	args := []string{sourcePath, "-o", outputPath}
	err := exec.CommandContext(ctx, "cwebp", append(args, o.Args...)...).Run()
	if err != nil {
		return nil, errors.New("transforming file with cwebp-lossy: " + err.Error())
	}

	fileStat, err := os.Stat(outputPath)
	if err != nil {
		return nil, err
	}

	return &ImageDescription{
		Optimizer: Name("cwebp-lossy"),
		Path:      outputPath,
		MimeType:  "image/webp",
		Size:      fileStat.Size(),
	}, nil
}
