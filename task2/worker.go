package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
)

type Result struct {
	Movie Movie
	Err   error
}

func worker(
	ctx context.Context,
	client *http.Client,
	jobs <-chan int,
	results chan<- Result,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return

		case id, ok := <-jobs:
			if !ok {
				return
			}

			movie, err := getMovieWithRetry(
				ctx,
				client,
				id,
			)

			result := Result{
				Movie: movie,
				Err:   err,
			}

			select {
			case <-ctx.Done():
				return

			case results <- result:
			}
		}
	}
}

func run(ctx context.Context, config Config) error {
	client := &http.Client{
		Timeout: config.Timeout,
	}

	jobs := make(chan int)
	results := make(chan Result)

	var wg sync.WaitGroup

	// Запускаем workers.
	for i := 0; i < config.Workers; i++ {
		wg.Add(1)

		go worker(
			ctx,
			client,
			jobs,
			results,
			&wg,
		)
	}

	// Передаём задания workers.
	go func() {
		defer close(jobs)

		for id := config.From; id <= config.To; id++ {
			select {
			case <-ctx.Done():
				return

			case jobs <- id:
			}
		}
	}()

	// Когда все workers закончат работу,
	// закрываем results.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Получаем результаты.
	for result := range results {
		if result.Err != nil {
			fmt.Println("error:", result.Err)
			continue
		}

		fmt.Printf(
			"%d - %s - %d - %s\n",
			result.Movie.ID,
			result.Movie.Title,
			result.Movie.Year,
			result.Movie.Director,
		)
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}
