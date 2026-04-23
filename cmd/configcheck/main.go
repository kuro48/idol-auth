package main

import (
	"fmt"
	"os"

	"github.com/ryunosukekurokawa/idol-auth/internal/config"
)

func main() {
	if _, err := config.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "config check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintln(os.Stdout, "config ok")
}
