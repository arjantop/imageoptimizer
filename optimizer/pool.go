package optimizer

import (
	"context"
	"log"
	"sort"
)

type Task struct {
	OriginalImage *ImageDescription
	Optimizers    []ImageOptimizer
}

type TaskPool struct {
	ScoringFunc func([]*ImageDescription, []error) (*ImageDescription, error)
}

func NewTaskPool() *TaskPool {
	return &TaskPool{
		ScoringFunc: func(descriptions []*ImageDescription, errors []error) (*ImageDescription, error) {
			sort.Sort(bySize(descriptions))
			for _, desc := range descriptions {
				log.Printf("optimizer=%s size=%d type=%s", desc.Optimizer, desc.Size, desc.MimeType)
			}
			return descriptions[0], nil
		},
	}
}

type result struct {
	desc *ImageDescription
	err  error
}

func (p *TaskPool) Do(ctx context.Context, task *Task) (*ImageDescription, error) {
	done := make(chan result, len(task.Optimizers))
	for _, imageOptimizer := range task.Optimizers {
		go func(opt ImageOptimizer) {
			desc, err := opt.Optimize(ctx, task.OriginalImage.Path)
			done <- result{
				desc: desc,
				err:  err,
			}
		}(imageOptimizer)
	}

	imageDescriptions := make([]*ImageDescription, 0, len(task.Optimizers)+1)
	imageDescriptions = append(imageDescriptions, task.OriginalImage)
	errors := make([]error, 0, len(task.Optimizers))

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-done:
			if result.err != nil {
				errors = append(errors, result.err)
			} else {
				imageDescriptions = append(imageDescriptions, result.desc)
			}
			if len(imageDescriptions)+len(errors) == len(task.Optimizers)+1 {
				break loop
			}
		}
	}
	return p.ScoringFunc(imageDescriptions, errors)
}
