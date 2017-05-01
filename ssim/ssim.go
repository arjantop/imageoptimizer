package ssim

import (
	"image"
	"math"

	"log"

	"github.com/disintegration/gift"
)

const (
	k1 = 0.01
	k2 = 0.03
	l  = 255
	c1 = (k1 * l) * (k1 * l)
	c2 = (k2 * l) * (k2 * l)
)

var kernel = [][]float64{
	{1.0576e-06, 7.8144e-06, 3.7022e-05, 1.1246e-04, 2.1905e-04, 2.7356e-04, 2.1905e-04, 1.1246e-04, 3.7022e-05, 7.8144e-06, 1.0576e-06},
	{7.8144e-06, 5.7741e-05, 2.7356e-04, 8.3101e-04, 1.6186e-03, 2.0214e-03, 1.6186e-03, 8.3101e-04, 2.7356e-04, 5.7741e-05, 7.8144e-06},
	{3.7022e-05, 2.7356e-04, 1.2961e-03, 3.9371e-03, 7.6684e-03, 9.5766e-03, 7.6684e-03, 3.9371e-03, 1.2961e-03, 2.7356e-04, 3.7022e-05},
	{1.1246e-04, 8.3101e-04, 3.9371e-03, 1.1960e-02, 2.3294e-02, 2.9091e-02, 2.3294e-02, 1.1960e-02, 3.9371e-03, 8.3101e-04, 1.1246e-04},
	{2.1905e-04, 1.6186e-03, 7.6684e-03, 2.3294e-02, 4.5371e-02, 5.6662e-02, 4.5371e-02, 2.3294e-02, 7.6684e-03, 1.6186e-03, 2.1905e-04},
	{2.7356e-04, 2.0214e-03, 9.5766e-03, 2.9091e-02, 5.6662e-02, 7.0762e-02, 5.6662e-02, 2.9091e-02, 9.5766e-03, 2.0214e-03, 2.7356e-04},
	{2.1905e-04, 1.6186e-03, 7.6684e-03, 2.3294e-02, 4.5371e-02, 5.6662e-02, 4.5371e-02, 2.3294e-02, 7.6684e-03, 1.6186e-03, 2.1905e-04},
	{1.1246e-04, 8.3101e-04, 3.9371e-03, 1.1960e-02, 2.3294e-02, 2.9091e-02, 2.3294e-02, 1.1960e-02, 3.9371e-03, 8.3101e-04, 1.1246e-04},
	{3.7022e-05, 2.7356e-04, 1.2961e-03, 3.9371e-03, 7.6684e-03, 9.5766e-03, 7.6684e-03, 3.9371e-03, 1.2961e-03, 2.7356e-04, 3.7022e-05},
	{7.8144e-06, 5.7741e-05, 2.7356e-04, 8.3101e-04, 1.6186e-03, 2.0214e-03, 1.6186e-03, 8.3101e-04, 2.7356e-04, 5.7741e-05, 7.8144e-06},
	{1.0576e-06, 7.8144e-06, 3.7022e-05, 1.1246e-04, 2.1905e-04, 2.7356e-04, 2.1905e-04, 1.1246e-04, 3.7022e-05, 7.8144e-06, 1.0576e-06},
}

func SsimWithAlpha(img1 *image.Gray, img2 *image.Gray, alpha *image.Alpha) float64 {
	boundsMin := img1.Bounds().Min
	boundsMax := img1.Bounds().Max

	const windowSize = 11

	var sum float64
	var numWindows uint
	var numTransparentWindows uint
	for y := boundsMin.Y; y < boundsMax.Y-windowSize; y++ {
		for x := boundsMin.X; x < boundsMax.X-windowSize; x++ {
			rect := image.Rect(x, y, x+windowSize, y+windowSize)
			alphaWindow := alpha.SubImage(rect).(*image.Alpha)
			if isFullyTransparent(alphaWindow) {
				numTransparentWindows += 1
				continue
			}
			img1Window := img1.SubImage(rect).(*image.Gray)
			img2Window := img2.SubImage(rect).(*image.Gray)
			sum += ssimWindow(img1Window, img2Window)
			numWindows++
		}
	}

	log.Printf("Number of windows: %d Transparent: %d", numWindows, numTransparentWindows)

	return sum / float64(numWindows)
}

func isFullyTransparent(img *image.Alpha) bool {
	boundsMin := img.Bounds().Min
	boundsMax := img.Bounds().Max

	for y := boundsMin.Y; y < boundsMax.Y; y++ {
		for x := boundsMin.X; x < boundsMax.X; x++ {
			if img.AlphaAt(x, y).A > 0 {
				return false
			}
		}
	}
	return true
}

func Ssim(img1 *image.Gray, img2 *image.Gray) float64 {
	boundsMin := img1.Bounds().Min
	boundsMax := img1.Bounds().Max

	const windowSize = 11

	var sum float64
	var numWindows uint
	for y := boundsMin.Y; y < boundsMax.Y-windowSize; y++ {
		for x := boundsMin.X; x < boundsMax.X-windowSize; x++ {
			rect := image.Rect(x, y, x+windowSize, y+windowSize)
			img1Window := img1.SubImage(rect).(*image.Gray)
			img2Window := img2.SubImage(rect).(*image.Gray)
			sum += ssimWindow(img1Window, img2Window)
			numWindows++
		}
	}

	return sum / float64(numWindows)
}

func ssimWindow(img1 *image.Gray, img2 *image.Gray) float64 {
	mean1 := mean(img1)
	mean2 := mean(img2)

	variance1 := stdev(mean1, img1)
	variance2 := stdev(mean2, img2)

	covar := covariance(mean1, img1, mean2, img2)

	a := (2*mean1*mean2 + c1) * (2*covar + c2)
	b := (mean1*mean1 + mean2*mean2 + c1) * (variance1*variance1 + variance2*variance2 + c2)

	return a / b
}

func mean(img *image.Gray) float64 {
	boundsMin := img.Bounds().Min
	boundsMax := img.Bounds().Max

	var sum float64
	for y := boundsMin.Y; y < boundsMax.Y; y++ {
		for x := boundsMin.X; x < boundsMax.X; x++ {
			sum += kernel[y-boundsMin.Y][x-boundsMin.X] * float64(img.GrayAt(x, y).Y)
		}
	}

	return sum
}

func stdev(mean float64, img *image.Gray) float64 {
	boundsMin := img.Bounds().Min
	boundsMax := img.Bounds().Max

	var sum float64
	for y := boundsMin.Y; y < boundsMax.Y; y++ {
		for x := boundsMin.X; x < boundsMax.X; x++ {
			val := float64(img.GrayAt(x, y).Y) - mean
			sum += kernel[y-boundsMin.Y][x-boundsMin.X] * (val * val)
		}
	}

	return math.Pow(sum, 0.5)
}

func covariance(mean1 float64, img1 *image.Gray, mean2 float64, img2 *image.Gray) float64 {
	boundsMin := img1.Bounds().Min
	boundsMax := img2.Bounds().Max

	var sum float64
	for y := boundsMin.Y; y < boundsMax.Y; y++ {
		for x := boundsMin.X; x < boundsMax.X; x++ {
			val1 := float64(img1.GrayAt(x, y).Y) - mean1
			val2 := float64(img2.GrayAt(x, y).Y) - mean2
			sum += kernel[y-boundsMin.Y][x-boundsMin.X] * val1 * val2
		}
	}

	return sum
}

type segmentedImage struct {
	edges           *image.Gray
	smoothRegions   *image.Gray
	texturedRegions *image.Gray
}

func segmentImage(img1 *image.Gray, sobel1 *image.Gray, img2 *image.Gray, sobel2 *image.Gray) (segmentedImage, segmentedImage) {
	boundsMin := sobel1.Bounds().Min
	boundsMax := sobel1.Bounds().Max

	var gmax uint8
	for y := boundsMin.Y; y < boundsMax.Y; y++ {
		for x := boundsMin.X; x < boundsMax.X; x++ {
			value := sobel1.GrayAt(x, y).Y
			if value > gmax {
				gmax = value
			}
		}
	}

	th1 := 0.12 * float64(gmax)
	th2 := 0.06 * float64(gmax)

	edges1 := image.NewGray(sobel1.Bounds())
	edges2 := image.NewGray(sobel1.Bounds())
	smoothRegions1 := image.NewGray(sobel1.Bounds())
	smoothRegions2 := image.NewGray(sobel1.Bounds())
	texturedRegions1 := image.NewGray(sobel1.Bounds())
	texturedRegions2 := image.NewGray(sobel1.Bounds())

	for y := boundsMin.Y; y < boundsMax.Y; y++ {
		for x := boundsMin.X; x < boundsMax.X; x++ {
			if float64(sobel1.GrayAt(x, y).Y) > th1 || float64(sobel2.GrayAt(x, y).Y) > th1 {
				edges1.SetGray(x, y, img1.GrayAt(x, y))
				edges2.SetGray(x, y, img2.GrayAt(x, y))
			} else if float64(sobel1.GrayAt(x, y).Y) < th2 || float64(sobel2.GrayAt(x, y).Y) <= th1 {
				smoothRegions1.SetGray(x, y, img1.GrayAt(x, y))
				smoothRegions2.SetGray(x, y, img2.GrayAt(x, y))
			} else {
				texturedRegions1.SetGray(x, y, img1.GrayAt(x, y))
				texturedRegions2.SetGray(x, y, img2.GrayAt(x, y))
			}
		}
	}

	return segmentedImage{
			edges:           edges1,
			smoothRegions:   smoothRegions1,
			texturedRegions: texturedRegions1,
		}, segmentedImage{
			edges:           edges2,
			smoothRegions:   smoothRegions2,
			texturedRegions: texturedRegions2,
		}
}

func ContentWeightedSsim(img1 *image.Gray, img2 *image.Gray) float64 {
	g := gift.New(gift.Sobel())

	sobel1 := image.NewGray(img1.Bounds())
	g.Draw(sobel1, img1)
	sobel2 := image.NewGray(img1.Bounds())
	g.Draw(sobel2, img2)

	segmented1, segmented2 := segmentImage(img1, sobel1, img2, sobel2)

	return 0.5*Ssim(segmented1.edges, segmented2.edges) +
		0.25*Ssim(segmented1.smoothRegions, segmented2.smoothRegions) +
		0.25*Ssim(segmented1.texturedRegions, segmented2.texturedRegions)
}
