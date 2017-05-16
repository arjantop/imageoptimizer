package optimizer

import (
	"crypto/rand"
	"encoding/hex"
	"image"
	"path"
	"strconv"
	"time"
)

func tempFilename(dir, originalFilename string) string {
	randomBytes := make([]byte, 10)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic("could not generate new temporary filename")
	}
	timestamp := strconv.FormatInt(time.Now().UnixNano(), 10)
	fileName := timestamp + "-" + hex.EncodeToString(randomBytes) + "-" + originalFilename
	return path.Join(dir, fileName)
}

func isFiletypeAccepted(acceptedFiletypes []string, matchingFiletypes []string) bool {
	for _, acceptedFiletype := range acceptedFiletypes {
		if contains(matchingFiletypes, acceptedFiletype) {
			return true
		}
	}
	return false
}

func contains(slice []string, elem string) bool {
	for _, s := range slice {
		if s == elem {
			return true
		}
	}
	return false
}

func convertToGrayscale(img image.Image) *image.Gray {
	output := image.NewGray(img.Bounds())
	for y := 0; y < img.Bounds().Max.Y; y++ {
		for x := 0; x < img.Bounds().Max.X; x++ {
			output.Set(x, y, img.At(x, y))
		}
	}
	return output
}
