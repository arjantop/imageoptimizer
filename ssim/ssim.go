package ssim

import (
	"image"
	"math"
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

func Ssim(img1 *image.Gray, img2 *image.Gray) float64 {
	boundsMin := img1.Bounds().Min
	boundsMax := img2.Bounds().Max

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
