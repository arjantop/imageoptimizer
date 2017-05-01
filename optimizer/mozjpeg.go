package optimizer

import (
	"bufio"
	"context"
	"errors"
	"image/jpeg"
	"os"
	"os/exec"
	"path"
	"strconv"

	"image"

	"github.com/arjantop/imageoptimizer/ssim"
	"github.com/disintegration/gift"
)

var _ ImageOptimizer = &MozjpegOptimizer{}

type MozjpegOptimizer struct {
	Args []string
}

func (o *MozjpegOptimizer) CanOptimize(mimeType string, acceptedTypes []string) bool {
	return mimeType == "image/jpeg" && isFiletypeAccepted(acceptedTypes, []string{"image/jpeg", "image/*", "*/*"})
}

func (o *MozjpegOptimizer) Optimize(ctx context.Context, sourcePath string, hidpi bool) (*ImageDescription, error) {
	outputPath := tempFilename(os.TempDir(), path.Base(sourcePath))
	args := make([]string, len(o.Args))
	copy(args, o.Args)
	cmd := exec.CommandContext(ctx, "jpegtran", append(args, sourcePath)...)

	outputFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer outputFile.Close()
	bufWriter := bufio.NewWriter(outputFile)
	cmd.Stdout = bufWriter

	err = cmd.Run()
	if err != nil {
		return nil, errors.New("transforming file with mozjpeg: " + err.Error())
	}

	err = bufWriter.Flush()
	if err != nil {
		return nil, err
	}

	fileStat, err := os.Stat(outputPath)
	if err != nil {
		return nil, err
	}

	return &ImageDescription{
		Optimizer: Name("mozjpeg"),
		Path:      outputPath,
		MimeType:  "image/jpeg",
		Size:      fileStat.Size(),
	}, nil
}

type mozjpegQualityOptimizer struct {
}

func (*mozjpegQualityOptimizer) OptimizePrecheck(ctx context.Context, sourcePath string) (bool, error) {
	return true, nil
}

func (*mozjpegQualityOptimizer) OptimizeQuality(ctx context.Context, sourcePath string, quality int) (*ImageDescription, error) {
	outputPath := tempFilename(os.TempDir(), path.Base(sourcePath))
	cmd := exec.CommandContext(ctx, "cjpeg", "-optimize", "-quality", strconv.Itoa(quality), sourcePath)

	outputFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer outputFile.Close()
	bufWriter := bufio.NewWriter(outputFile)
	cmd.Stdout = bufWriter

	err = cmd.Run()
	if err != nil {
		return nil, errors.New("transforming file with mozjpeg: " + err.Error())
	}

	err = bufWriter.Flush()
	if err != nil {
		return nil, err
	}

	fileStat, err := os.Stat(outputPath)
	if err != nil {
		return nil, err
	}

	return &ImageDescription{
		Optimizer: Name("mozjpeg-lossy"),
		Path:      outputPath,
		MimeType:  "image/jpeg",
		Size:      fileStat.Size(),
	}, nil
}

func (*mozjpegQualityOptimizer) CompareImages(ctx context.Context, sourcePath string, imageDesc *ImageDescription, hidpi bool) (float64, error) {
	file1, err := os.Open(sourcePath)
	if err != nil {
		return 0, err
	}
	defer file1.Close()
	img1, err := jpeg.Decode(file1)
	if err != nil {
		return 0, err
	}

	file2, err := os.Open(imageDesc.Path)
	if err != nil {
		return 0, err
	}
	defer file2.Close()

	img2, err := jpeg.Decode(file2)
	if err != nil {
		return 0, err
	}

	if hidpi {
		g := gift.New(
			gift.Resize(img1.Bounds().Dx()/2, 0, gift.LanczosResampling),
		)

		resized1 := image.NewRGBA(g.Bounds(img1.Bounds()))
		g.Draw(resized1, img1)
		img1 = resized1
		resized2 := image.NewRGBA(g.Bounds(img2.Bounds()))
		g.Draw(resized2, img2)
		img2 = resized2
	}

	return ssim.Ssim(convertToGrayscale(img1), convertToGrayscale(img2)), nil
}

func (*mozjpegQualityOptimizer) CanOptimize(mimeType string, acceptedTypes []string) bool {
	return mimeType == "image/jpeg" && isFiletypeAccepted(acceptedTypes, []string{"image/jpeg", "image/*", "*/*"})
}

func (o *mozjpegQualityOptimizer) Optimize(ctx context.Context, sourcePath string, hidpi bool) (*ImageDescription, error) {
	return o.OptimizeQuality(ctx, sourcePath, 100)
}

func NewMozjpegLossyOptimizer(minSsim float64) ImageOptimizer {
	opt := new(mozjpegQualityOptimizer)
	return &AutomaticOptimizer{
		Optimizer: opt,
		MinSsim:   minSsim,
	}
}
