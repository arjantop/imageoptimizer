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

func (o *MozjpegOptimizer) Optimize(ctx context.Context, sourcePath string) (*ImageDescription, error) {
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

func NewMozjpegLossyOptimizer(minSsim float64) ImageOptimizer {
	return &AutomaticOptimizer{
		CanOptimizeImage: func(mimeType string, acceptedTypes []string) bool {
			return mimeType == "image/jpeg" && isFiletypeAccepted(acceptedTypes, []string{"image/jpeg", "image/*", "*/*"})
		},
		OptimizeQuality: optimizeQuality,
		CompareImages:   compareImages,
		MinSsim:         minSsim,
	}
}

func compareImages(_ context.Context, sourcePath string, imgDesc2 *ImageDescription) (float64, error) {
	file1, err := os.Open(sourcePath)
	if err != nil {
		return 0, err
	}
	defer file1.Close()
	img1, err := jpeg.Decode(file1)
	if err != nil {
		return 0, err
	}

	file2, err := os.Open(imgDesc2.Path)
	if err != nil {
		return 0, err
	}
	defer file2.Close()

	img2, err := jpeg.Decode(file2)
	if err != nil {
		return 0, err
	}

	g := gift.New(
		gift.Resize(img1.Bounds().Dx()/2, 0, gift.LanczosResampling),
	)

	resized1 := image.NewRGBA(g.Bounds(img1.Bounds()))
	g.Draw(resized1, img1)
	resized2 := image.NewRGBA(g.Bounds(img2.Bounds()))
	g.Draw(resized2, img2)

	//resized1 := img1
	//resized2 := img2

	return ssim.Ssim(convertToGrayscale(resized1), convertToGrayscale(resized2)), nil
}

func optimizeQuality(ctx context.Context, sourcePath string, quality int) (*ImageDescription, error) {
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
