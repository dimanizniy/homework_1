package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HTTPError struct {
	StatusCode int
	MovieID    int
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf(
		"movie %d: HTTP status %d",
		e.MovieID,
		e.StatusCode,
	)
}

func getMovie(ctx context.Context, client *http.Client, id int) (Movie, error) {
	url := fmt.Sprintf("https://homeworksite.site/%d/info.0.json", id)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		return Movie{}, fmt.Errorf("err %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return Movie{}, fmt.Errorf("err %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:

	case http.StatusNotFound:
		return Movie{}, fmt.Errorf("movie %d: not found", id)

	case http.StatusTooManyRequests:
		return Movie{}, fmt.Errorf("movie %d: rate limit exceeded", id)

	default:
		if resp.StatusCode >= 500 {
			return Movie{}, &HTTPError{
				StatusCode: resp.StatusCode,
				MovieID:    id,
			}
		}

		return Movie{}, fmt.Errorf(
			"movie %d: HTTP status %d",
			id,
			resp.StatusCode,
		)

	}

	var movie Movie

	if err := json.NewDecoder(resp.Body).Decode(&movie); err != nil {
		return Movie{}, fmt.Errorf(
			"decode movie %d: %w",
			id,
			err,
		)
	}
	return movie, nil
}
func isRetryable(err error) bool {
	httpErr, ok := err.(*HTTPError)
	if !ok {
		return false
	}

	return httpErr.StatusCode == http.StatusTooManyRequests ||
		httpErr.StatusCode >= 500
}

func getMovieWithRetry(
	ctx context.Context,
	client *http.Client,
	id int,
) (Movie, error) {
	const maxAttempts = 3

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		movie, err := getMovie(ctx, client, id)

		if err == nil {
			return movie, nil
		}

		if !isRetryable(err) {
			return Movie{}, err
		}

		if attempt == maxAttempts {
			return Movie{}, err
		}

		delay := time.Duration(attempt) * time.Second

		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return Movie{}, ctx.Err()

		case <-timer.C:
		}
	}

	return Movie{}, fmt.Errorf("unexpected error")
}
