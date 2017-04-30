package optimizer

import (
	"context"
	"log"
	"sort"
)

type Task struct {
	OriginalImage *ImageDescription
	Optimizers    []ImageOptimizer
	Hidpi         bool
}

type TaskPool struct {
	ScoringFunc func([]*ImageDescription, []error) (*ImageDescription, error)
}

func NewTaskPool() *TaskPool {
	return &TaskPool{
		ScoringFunc: func(descriptions []*ImageDescription, errors []error) (*ImageDescription, error) {
			if len(errors) > 0 {
				log.Println(errors)
			}
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
			desc, err := opt.Optimize(ctx, task.OriginalImage.Path, task.Hidpi)
			done <- result{
				desc: desc,
				err:  err,
			}
		}(imageOptimizer)
	}

	imageDescriptions := make([]*ImageDescription, 0, len(task.Optimizers)+1)
	imageDescriptions = append(imageDescriptions, task.OriginalImage)
	errors := make([]error, 0, len(task.Optimizers))
	var numDone int

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case result := <-done:
			if result.err != nil {
				errors = append(errors, result.err)
			} else if result.desc != nil {
				imageDescriptions = append(imageDescriptions, result.desc)
			}
			numDone++
			if numDone == len(task.Optimizers) {
				break loop
			}
		}
	}
	return p.ScoringFunc(imageDescriptions, errors)
}
