package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

func Generator(ctx context.Context, id int) <-chan string {
	out := make(chan string)

	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)*9973))

	go func() {
		defer close(out)

		for n := 1; n <= 1000; n++ {
			msg := fmt.Sprintf("%d_%d", id, n)

			select {
			case <-ctx.Done():
				return
			case out <- msg:
			}

			delay := time.Duration(r.Intn(1001)) * time.Millisecond
			if delay == 0 {
				continue
			}

			t := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				t.Stop()
				return
			case <-t.C:
			}
		}
	}()

	return out
}
