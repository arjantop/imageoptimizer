package optimizer

import (
	"context"
	"log"
)

type ImageQualityOptimizer interface {
	OptimizePrecheck(ctx context.Context, sourcePath string) (bool, error)
	OptimizeQuality(ctx context.Context, sourcePath string, quality int) (*ImageDescription, error)
	CompareImages(ctx context.Context, sourcePath string, imageDesc *ImageDescription, hidpi bool) (float64, error)
	ImageOptimizer
}

var _ ImageOptimizer = &AutomaticOptimizer{}

type AutomaticOptimizer struct {
	Optimizer ImageQualityOptimizer
	MinSsim   float64
}

func (o *AutomaticOptimizer) CanOptimize(mimeType string, acceptedTypes []string) bool {
	return o.Optimizer.CanOptimize(mimeType, acceptedTypes)
}

func (o *AutomaticOptimizer) Optimize(ctx context.Context, sourcePath string, hidpi bool) (*ImageDescription, error) {
	ok, err := o.Optimizer.OptimizePrecheck(ctx, sourcePath)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	var best *ImageDescription
	qualityMax := 100
	qualityMin := 0
	for qualityMax-qualityMin >= 0 {
		log.Println(qualityMin, qualityMax)
		quality := (qualityMax + qualityMin) / 2
		log.Printf("Trying quality %d", quality)

		imageDesc, err := o.Optimizer.OptimizeQuality(ctx, sourcePath, quality)
		if err != nil {
			return nil, err
		}

		score, err := o.Optimizer.CompareImages(ctx, sourcePath, imageDesc, hidpi)
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
