package optimizer

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"

	"github.com/arjantop/imageoptimizer/ssim"
	"github.com/disintegration/gift"
	"golang.org/x/image/webp"
)

var _ ImageOptimizer = &WebpLosslessOptimizer{}

type WebpLosslessOptimizer struct {
	Args []string
}

func (o *WebpLosslessOptimizer) CanOptimize(mimeType string, acceptedTypes []string) bool {
	return mimeType == "image/png" && isFiletypeAccepted(acceptedTypes, []string{"image/webp"})
}

func (o *WebpLosslessOptimizer) Optimize(ctx context.Context, sourcePath string, hidpi bool) (*ImageDescription, error) {
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

type webpQualityOptimizer struct {
	optimizePrecheck func(ctx context.Context, sourcePath string) (bool, error)
	optimizerType    string
}

func (o *webpQualityOptimizer) OptimizePrecheck(ctx context.Context, sourcePath string) (bool, error) {
	if o.optimizePrecheck != nil {
		return o.optimizePrecheck(ctx, sourcePath)
	} else {
		return true, nil
	}
}

func (o *webpQualityOptimizer) OptimizeQuality(ctx context.Context, sourcePath string, quality int) (*ImageDescription, error) {
	outputPath := tempFilename(os.TempDir(), path.Base(sourcePath))
	cmd := exec.CommandContext(ctx, "cwebp", "-q", strconv.Itoa(quality), "-o", outputPath, sourcePath)

	outputFile, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}
	defer outputFile.Close()
	bufWriter := bufio.NewWriter(outputFile)
	cmd.Stdout = bufWriter

	err = cmd.Run()
	if err != nil {
		return nil, errors.New("transforming file with cwebp: " + err.Error())
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
		Optimizer: Name(fmt.Sprintf("cwebp-lossy[%s]", o.optimizerType)),
		Path:      outputPath,
		MimeType:  "image/webp",
		Size:      fileStat.Size(),
	}, nil
}

func (o *webpQualityOptimizer) CompareImages(ctx context.Context, sourcePath string, imageDesc *ImageDescription, hidpi bool) (float64, error) {
	converted, err := o.OptimizeQuality(ctx, sourcePath, 100)
	if err != nil {
		return 0, err
	}

	file1, err := os.Open(converted.Path)
	if err != nil {
		return 0, err
	}
	defer file1.Close()
	img1, err := webp.Decode(file1)
	if err != nil {
		return 0, err
	}

	file2, err := os.Open(imageDesc.Path)
	if err != nil {
		return 0, err
	}
	defer file2.Close()

	img2, err := webp.Decode(file2)
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

func (o *webpQualityOptimizer) CanOptimize(mimeType string, acceptedTypes []string) bool {
	return mimeType == o.optimizerType && isFiletypeAccepted(acceptedTypes, []string{"image/webp"})
}

func (o *webpQualityOptimizer) Optimize(ctx context.Context, sourcePath string, hidpi bool) (*ImageDescription, error) {
	return o.OptimizeQuality(ctx, sourcePath, 100)
}

func NewWebpLossyPngOptimizer(minSsim float64) ImageOptimizer {
	opt := &webpQualityOptimizer{
		optimizePrecheck: func(ctx context.Context, sourcePath string) (bool, error) {
			file, err := os.Open(sourcePath)
			if err != nil {
				return false, err
			}
			defer file.Close()

			img, err := png.Decode(file)
			if err != nil {
				return false, err
			}

			for y := 0; y < img.Bounds().Max.Y; y++ {
				for x := 0; x < img.Bounds().Max.X; x++ {
					_, _, _, a := img.At(x, y).RGBA()
					if a < uint32(^uint16(0)) {
						log.Println("Image has transparency")
						return false, nil
					}
				}
			}
			return true, nil
		},
		optimizerType: "image/png",
	}
	return &AutomaticOptimizer{
		Optimizer: opt,
		MinSsim:   minSsim,
	}
}

func NewWebpLossyJpegOptimizer(minSsim float64) ImageOptimizer {
	opt := &webpQualityOptimizer{
		optimizerType: "image/jpeg",
	}
	return &AutomaticOptimizer{
		Optimizer: opt,
		MinSsim:   minSsim,
	}
}
