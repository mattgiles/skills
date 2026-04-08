package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/pflag"
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
			fmt.Fprintln(os.Stderr, err)
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
	if isUsageError(err) {
		return exitCodeUsage
	}
	return exitCodeFailure
}

func isUsageError(err error) bool {
	message := strings.TrimSpace(err.Error())
	for _, prefix := range []string{
		"unknown flag: ",
		"unknown shorthand flag: ",
		"flag needs an argument: ",
		"bad flag syntax: ",
		"unknown command ",
		"accepts ",
		"requires at least ",
		"requires at most ",
		"requires between ",
		"requires ",
		"invalid argument ",
		"arguments accepted",
		"subcommand is required",
	} {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}

	return false
}
