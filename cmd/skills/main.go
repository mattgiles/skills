package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/pflag"

	"github.com/mattgiles/skills/internal/ui"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := newRootCommand().ExecuteContext(ctx); err != nil {
		exitCode := exitCodeForError(err)
		if exitCode != exitCodeSuccess {
			ui.PrintError(os.Stderr, err)
		}
		return exitCode
	}

	return exitCodeSuccess
}

const (
	exitCodeSuccess = 0
	exitCodeFailure = 1
	exitCodeUsage   = 2
	exitCodeDoctor  = 3
)

type usageError struct {
	err error
}

func (e usageError) Error() string {
	return e.err.Error()
}

func (e usageError) Unwrap() error {
	return e.err
}

func markUsage(err error) error {
	if err == nil {
		return nil
	}
	return usageError{err: err}
}

func exitCodeForError(err error) int {
	if err == nil {
		return exitCodeSuccess
	}
	if errors.Is(err, errDoctorFoundProblems) {
		return exitCodeDoctor
	}
	if errors.Is(err, flag.ErrHelp) || errors.Is(err, pflag.ErrHelp) {
		return exitCodeSuccess
	}
	var usage usageError
	if errors.As(err, &usage) {
		return exitCodeUsage
	}
	return exitCodeFailure
}
