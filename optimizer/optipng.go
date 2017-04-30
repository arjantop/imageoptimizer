package optimizer

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
)

var _ ImageOptimizer = &OptipngOptimizer{}

type OptipngOptimizer struct {
	Args []string
}

func (o *OptipngOptimizer) CanOptimize(mimeType string, acceptedTypes []string) bool {
	return mimeType == "image/png" && isFiletypeAccepted(acceptedTypes, []string{"image/png", "image/*", "*/*"})
}

func (o *OptipngOptimizer) Optimize(ctx context.Context, sourcePath string, hidpi bool) (*ImageDescription, error) {
	outputPath := tempFilename(os.TempDir(), path.Base(sourcePath))
	args := []string{sourcePath, "-out", outputPath}
	err := exec.CommandContext(ctx, "optipng", append(args, o.Args...)...).Run()
	if err != nil {
		return nil, errors.New("transforming file with optipng: " + err.Error())
	}

	fileStat, err := os.Stat(outputPath)
	if err != nil {
		return nil, err
	}

	return &ImageDescription{
		Optimizer: Name("optipng"),
		Path:      outputPath,
		MimeType:  "image/png",
		Size:      fileStat.Size(),
	}, nil
}
