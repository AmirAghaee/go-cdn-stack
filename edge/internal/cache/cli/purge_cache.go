package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/AmirAghaee/go-cdn-stack/edge/internal/cache"
)

type DomainPurger interface {
	PurgeDomain(context.Context, string) (cache.PurgeResult, error)
}

type PurgeCacheCommand struct {
	purger DomainPurger
	stdout io.Writer
	stderr io.Writer
}

func NewPurgeCacheCommand(purger DomainPurger) *PurgeCacheCommand {
	return &PurgeCacheCommand{purger: purger, stdout: os.Stdout, stderr: os.Stderr}
}

func (c *PurgeCacheCommand) Run(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("purge-cache", flag.ContinueOnError)
	flags.SetOutput(c.stderr)
	domain := flags.String("domain", "", "domain whose cached files should be purged")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if strings.TrimSpace(*domain) == "" {
		return fmt.Errorf("--domain is required")
	}

	result, err := c.purger.PurgeDomain(ctx, *domain)
	if err != nil {
		return fmt.Errorf("purge cache for %q: %w", *domain, err)
	}
	if _, err := fmt.Fprintf(c.stdout, "purged %d cached entries (%d bytes) for domain %s\n", result.EntriesPurged, result.BytesFreed, result.Domain); err != nil {
		return fmt.Errorf("write success message: %w", err)
	}
	return nil
}
