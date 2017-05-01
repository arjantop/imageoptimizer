package optimizer

import (
	"bufio"
	"context"
	"errors"
	"image/jpeg"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"

	"image"

	"fmt"

	"image/png"

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
	optimizePrecheck func(ctx context.Context, sourcePath string) (bool, error)
	optimizerType    string
}

func (o *mozjpegQualityOptimizer) OptimizePrecheck(ctx context.Context, sourcePath string) (bool, error) {
	if o.optimizePrecheck != nil {
		return o.optimizePrecheck(ctx, sourcePath)
	} else {
		return true, nil
	}
}

func convertToJpeg(sourcePath string) (string, error) {
	sourcePathConverted := tempFilename(os.TempDir(), path.Base(sourcePath))
	fileInput, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer fileInput.Close()

	fileOutput, err := os.Create(sourcePathConverted)
	if err != nil {
		return "", err
	}
	defer fileOutput.Close()

	imgInput, err := png.Decode(fileInput)
	if err != nil {
		return "", err
	}
	err = jpeg.Encode(fileOutput, imgInput, &jpeg.Options{
		Quality: 100,
	})
	if err != nil {
		return "", err
	}
	return sourcePathConverted, nil
}

func (o *mozjpegQualityOptimizer) OptimizeQuality(ctx context.Context, sourcePath string, quality int) (*ImageDescription, error) {
	// TODO: Remove hack
	realSourcePath := sourcePath
	if o.optimizerType == "image/png" {
		p, err := convertToJpeg(sourcePath)
		if err != nil {
			return nil, err
		}
		realSourcePath = p
	}

	outputPath := tempFilename(os.TempDir(), path.Base(sourcePath))
	cmd := exec.CommandContext(ctx, "cjpeg", "-optimize", "-quality", strconv.Itoa(quality), realSourcePath)

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
		Optimizer: Name(fmt.Sprintf("mozjpeg-lossy[%s]", o.optimizerType)),
		Path:      outputPath,
		MimeType:  "image/jpeg",
		Size:      fileStat.Size(),
	}, nil
}

func (o *mozjpegQualityOptimizer) CompareImages(ctx context.Context, sourcePath string, imageDesc *ImageDescription, hidpi bool) (float64, error) {
	// TODO: Remove hack
	if o.optimizerType == "image/png" {
		file1, err := os.Open(sourcePath)
		if err != nil {
			return 0, err
		}
		defer file1.Close()
		img1, err := png.Decode(file1)
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
	} else {

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
}

func (o *mozjpegQualityOptimizer) CanOptimize(mimeType string, acceptedTypes []string) bool {
	return mimeType == o.optimizerType && isFiletypeAccepted(acceptedTypes, []string{"image/jpeg", "image/*", "*/*"})
}

func (o *mozjpegQualityOptimizer) Optimize(ctx context.Context, sourcePath string, hidpi bool) (*ImageDescription, error) {
	return o.OptimizeQuality(ctx, sourcePath, 100)
}

func NewMozjpegPngLossyOptimizer(minSsim float64) ImageOptimizer {
	opt := &mozjpegQualityOptimizer{
		optimizerType: "image/png",
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
	}
	return &AutomaticOptimizer{
		Optimizer: opt,
		MinSsim:   minSsim,
	}
}

func NewMozjpegLossyOptimizer(minSsim float64) ImageOptimizer {
	opt := &mozjpegQualityOptimizer{
		optimizerType: "image/jpeg",
	}
	return &AutomaticOptimizer{
		Optimizer: opt,
		MinSsim:   minSsim,
	}
}
