package util

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"sync"
)

const Workers = 5

func ParallelizeSyncNode(ctx context.Context, workers int, nodelist []*v1.Node, Do func(ctx context.Context, node *v1.Node)) {
	var stop <-chan struct{}
	pieces := len(nodelist)
	toProcess := make(chan *v1.Node, pieces)
	for _, node := range nodelist {
		toProcess <- node
	}
	close(toProcess)
	if pieces < workers {
		workers = pieces
	}
	wg := sync.WaitGroup{}
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for node := range toProcess {
				select {
				case <-stop:
					return
				default:
					Do(ctx, node)
				}
			}
		}()
	}
	wg.Wait()
}
