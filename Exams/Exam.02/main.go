package main

import (
	"context"
	"fmt"
	"runtime"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	timer := time.NewTimer(1 * time.Minute)
	defer timer.Stop()
	timerCh := timer.C

	inputs := make([]<-chan string, 0, 5)
	for id := 1; id <= 5; id++ {
		inputs = append(inputs, Generator(ctx, id))
	}

	out := Multiplex(ctx, inputs...)

	for {
		select {
		case <-timerCh:
			fmt.Println(">>> demo cancelled after 1 minute")
			cancel()
			timerCh = nil
		case <-ctx.Done():
			for range out {
			}
			time.Sleep(50 * time.Millisecond)
			fmt.Printf("Active goroutines at end: %d\n", runtime.NumGoroutine())
			return
		case v, ok := <-out:
			if !ok {
				time.Sleep(50 * time.Millisecond)
				fmt.Printf("Active goroutines at end: %d\n", runtime.NumGoroutine())
				return
			}
			fmt.Println(v)
		}
	}
}
