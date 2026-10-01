package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"
)

type Config struct {
	From    int
	To      int
	Workers int
	Timeout time.Duration
}

func parseConfig() Config {
	from := flag.Int(
		"from",
		0,
		"ID первого фильма",
	)

	to := flag.Int(
		"to",
		0,
		"ID последнего фильма",
	)

	workers := flag.Int(
		"workers",
		10,
		"Количество workers",
	)

	timeout := flag.Duration(
		"timeout",
		5*time.Second,
		"Таймаут HTTP-запроса",
	)

	flag.Parse()

	return Config{
		From:    *from,
		To:      *to,
		Workers: *workers,
		Timeout: *timeout,
	}
}

func validateConfig(config Config) error {
	if config.From <= 0 {
		return fmt.Errorf(
			"--from должен быть положительным",
		)
	}

	if config.To <= 0 {
		return fmt.Errorf(
			"--to должен быть положительным",
		)
	}

	if config.From > config.To {
		return fmt.Errorf(
			"--from не может быть больше --to",
		)
	}

	if config.Workers <= 0 {
		return fmt.Errorf(
			"--workers должен быть положительным",
		)
	}

	if config.Timeout <= 0 {
		return fmt.Errorf(
			"--timeout должен быть положительным",
		)
	}

	return nil
}

func main() {
	config := parseConfig()

	if err := validateConfig(config); err != nil {
		fmt.Println("error:", err)
		return
	}

	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	defer cancel()

	signalChan := make(chan os.Signal, 1)

	signal.Notify(
		signalChan,
		os.Interrupt,
	)

	defer signal.Stop(signalChan)

	go func() {
		<-signalChan

		fmt.Println(
			"\nПолучен Ctrl+C, завершаем работу...",
		)

		cancel()
	}()

	start := time.Now()
	err := run(ctx, config)
	elapsed := time.Since(start)

	if err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}

		fmt.Println("error:", err)
		return
	}

	fmt.Printf("\nВремя выполнения: %v\n", elapsed)
}
