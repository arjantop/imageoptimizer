package optimizer

import (
	"context"
	"log"
)

var _ ImageOptimizer = &AutomaticOptimizer{}

type AutomaticOptimizer struct {
	CanOptimizeImage func(mimeType string, acceptedTypes []string) bool
	OptimizePrecheck func(ctx context.Context, sourcePath string) (bool, error)
	OptimizeQuality  func(ctx context.Context, sourcePath string, quality int) (*ImageDescription, error)
	CompareImages    func(ctx context.Context, sourcePath string, imgDesc *ImageDescription, hidpi bool) (float64, error)
	MinSsim          float64
}

func (o *AutomaticOptimizer) CanOptimize(mimeType string, acceptedTypes []string) bool {
	return o.CanOptimizeImage(mimeType, acceptedTypes)
}

func (o *AutomaticOptimizer) Optimize(ctx context.Context, sourcePath string, hidpi bool) (*ImageDescription, error) {
	if o.OptimizePrecheck != nil {
		ok, err := o.OptimizePrecheck(ctx, sourcePath)
		if err != nil {
			return nil, err
		} else if !ok {
			return nil, nil
		}
	}

	var best *ImageDescription
	qualityMax := 100
	qualityMin := 0
	for qualityMax-qualityMin >= 0 {
		log.Println(qualityMin, qualityMax)
		quality := (qualityMax + qualityMin) / 2
		log.Printf("Trying quality %d", quality)

		imageDesc, err := o.OptimizeQuality(ctx, sourcePath, quality)
		if err != nil {
			return nil, err
		}

		score, err := o.CompareImages(ctx, sourcePath, imageDesc, hidpi)
		if err != nil {
			return nil, err
		}
		log.Printf("ssim = %f", score)
		if score < o.MinSsim {
			qualityMin = quality + 1
		} else {
			qualityMax = quality - 1
			log.Printf("Using quality %d", quality)
			best = imageDesc
		}
	}

	return best, nil
}
