package main

import (
	"context"
	"sync"
)

func Multiplex(ctx context.Context, inputs ...<-chan string) <-chan string {
	out := make(chan string)

	var wg sync.WaitGroup
	wg.Add(len(inputs))

	for _, ch := range inputs {
		in := ch
		go func() {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case v, ok := <-in:
					if !ok {
						return
					}
					out <- v
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
