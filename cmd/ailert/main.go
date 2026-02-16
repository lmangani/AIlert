package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/ailert/ailert/internal/alertmanager"
	"github.com/ailert/ailert/internal/config"
	"github.com/ailert/ailert/internal/engine"
	"github.com/ailert/ailert/internal/source"
	"github.com/ailert/ailert/internal/store"
	"github.com/ailert/ailert/internal/types"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to config YAML")
	flag.Parse()
	if err := run(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "ailert: %v\n", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	st := store.New(cfg.StorePath)
	if err := st.Load(); err != nil {
		return fmt.Errorf("load store: %w", err)
	}
	eng := engine.New(st)
	var amClient *alertmanager.Client
	if cfg.AlertmanagerURL != "" {
		amClient = alertmanager.NewClient(cfg.AlertmanagerURL)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Run each configured source and process records
	var wg sync.WaitGroup
	for _, spec := range cfg.Sources {
		var src source.Source
		switch spec.Type {
		case "file":
			src = &source.FileSource{Path: spec.Path, SourceID: spec.ID}
		case "prometheus", "metrics":
			src = &source.PrometheusSource{URL: spec.URL, SourceID: spec.ID}
		default:
			return fmt.Errorf("unknown source type %q", spec.Type)
		}
		wg.Add(1)
		go func(s source.Source) {
			defer wg.Done()
			runSource(ctx, eng, s, amClient)
		}(src)
	}
	go func() {
		wg.Wait()
		cancel()
	}()

	// Block until context is done (sources finished or Ctrl+C), then save store
	<-ctx.Done()
	if cfg.StorePath != "" {
		if err := st.Save(); err != nil {
			return fmt.Errorf("save store: %w", err)
		}
	}
	printSummary(st)
	return nil
}

func runSource(ctx context.Context, eng *engine.Engine, src source.Source, amClient *alertmanager.Client) {
	recCh, errCh := src.Stream(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case err, ok := <-errCh:
			if ok && err != nil {
				fmt.Fprintf(os.Stderr, "source %s: %v\n", src.ID(), err)
			}
			return
		case rec, ok := <-recCh:
			if !ok {
				return
			}
			res := eng.Process(&rec)
			if res.Suppressed {
				continue
			}
			status := "known"
			if res.IsNew {
				status = "new"
			}
			fmt.Printf("[%s] %s %s (count=%d) %s\n", res.Level.String(), status, res.Hash, res.Count, truncate(res.Sample, 60))
			if amClient != nil {
				emitAlert(amClient, &rec, &res)
			}
		}
	}
}

func emitAlert(client *alertmanager.Client, rec *types.Record, res *engine.Result) {
	now := time.Now()
	a := alertmanager.Alert{
		Labels: map[string]string{
			"alertname":    "ailert",
			"pattern_hash": res.Hash,
			"level":        res.Level.String(),
			"source":       rec.SourceID,
		},
		Annotations: map[string]string{
			"summary":     res.Level.String() + " pattern",
			"description": truncate(res.Sample, 500),
		},
		StartsAt: now,
		EndsAt:   time.Time{},
	}
	if err := client.PostAlerts([]alertmanager.Alert{a}); err != nil {
		fmt.Fprintf(os.Stderr, "alertmanager: %v\n", err)
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func printSummary(st *store.Store) {
	list := st.ListSeen()
	if len(list) == 0 {
		return
	}
	fmt.Println("\n--- Summary ---")
	var total int64
	for _, p := range list {
		total += p.Count
	}
	fmt.Printf("Total patterns: %d, total messages: %d\n", len(list), total)
	for _, p := range list {
		fmt.Printf("  %s %s count=%d %s\n", p.Level.String(), p.Hash, p.Count, truncate(p.Sample, 50))
	}
}
