package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/ailert/ailert/internal/alertmanager"
	"github.com/ailert/ailert/internal/changes"
	"github.com/ailert/ailert/internal/config"
	"github.com/ailert/ailert/internal/engine"
	"github.com/ailert/ailert/internal/pattern"
	"github.com/ailert/ailert/internal/snapshot"
	"github.com/ailert/ailert/internal/metrics"
	"github.com/ailert/ailert/internal/source"
	"github.com/ailert/ailert/internal/store"
	"github.com/ailert/ailert/internal/types"
)

const suppressCountThreshold = 5

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}
	sub := os.Args[1]
	args := os.Args[2:]
	var err error
	switch sub {
	case "run":
		err = cmdRun(args)
	case "suppress":
		err = cmdSuppress(args)
	case "detect-changes":
		err = cmdDetectChanges(args)
	case "suggest-rules":
		err = cmdSuggestRules(args)
	case "apply-rule":
		err = cmdApplyRule(args)
	default:
		printUsage()
		os.Exit(1)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "ailert: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `usage: ailert <command> [options]

Commands:
  run             Stream sources, detect patterns, emit to Alertmanager (optional)
  suppress        Add suppression by hash or pattern sample; optionally create Alertmanager silence
  detect-changes  Compare current store to last snapshot, print diff
  suggest-rules   From last run or snapshot, suggest suppress/alert rules (heuristic)
  apply-rule      Apply a rule: suppress <hash> or alert <hash>

Use -h with a command for details.
`)
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "Config YAML")
	saveSnapshot := fs.String("save-snapshot", "", "Save snapshot to this dir after run (for detect-changes)")
	metricsAddr := fs.String("metrics-addr", "", "If set, serve Prometheus metrics on this address (e.g. :9090)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
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

	var wg sync.WaitGroup
	for _, spec := range cfg.Sources {
		src := sourceFromSpec(spec)
		if src == nil {
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

	<-ctx.Done()
	if cfg.StorePath != "" {
		if err := st.Save(); err != nil {
			return fmt.Errorf("save store: %w", err)
		}
	}
	if *metricsAddr != "" {
		metrics.Serve(*metricsAddr)
	}
	if *saveSnapshot != "" {
		list := st.ListSeen()
		ents := make([]snapshot.PatternEnt, len(list))
		for i, p := range list {
			ents[i] = snapshot.PatternEnt{Level: p.Level, Hash: p.Hash, Sample: p.Sample, Count: p.Count}
		}
		path := filepath.Join(*saveSnapshot, "snapshot_latest.json")
		if err := snapshot.Save(path, ents); err != nil {
			return fmt.Errorf("save snapshot: %w", err)
		}
		fmt.Println("Snapshot saved to", path)
	}
	printSummary(st)
	return nil
}

func sourceFromSpec(spec config.SourceSpec) source.Source {
	switch spec.Type {
	case "file":
		return &source.FileSource{Path: spec.Path, SourceID: spec.ID}
	case "prometheus", "metrics":
		return &source.PrometheusSource{URL: spec.URL, SourceID: spec.ID}
	case "http":
		return &source.HTTPSource{URL: spec.URL, SourceID: spec.ID}
	default:
		return nil
	}
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
			metrics.RecordsProcessed.Add(1)
			if res.Suppressed {
				metrics.PatternsSuppressed.Add(1)
				continue
			}
			if res.IsNew {
				metrics.PatternsNew.Add(1)
			} else {
				metrics.PatternsKnown.Add(1)
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
	} else {
		metrics.AlertsEmitted.Add(1)
	}
}

func cmdSuppress(args []string) error {
	fs := flag.NewFlagSet("suppress", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "Config YAML")
	hash := fs.String("hash", "", "Pattern hash to suppress")
	patternLine := fs.String("pattern", "", "Sample log line (hash will be computed)")
	reason := fs.String("reason", "one-click", "Reason for suppression")
	createSilence := fs.Bool("create-silence", false, "Create Alertmanager silence (requires alertmanager_url in config)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *hash == "" && *patternLine == "" {
		return fmt.Errorf("suppress: provide -hash or -pattern")
	}
	h := *hash
	if h == "" {
		p := pattern.New(*patternLine)
		h = p.Hash()
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	st := store.New(cfg.StorePath)
	if err := st.Load(); err != nil {
		return err
	}
	st.Suppress(h, *reason)
	if cfg.StorePath != "" {
		if err := st.Save(); err != nil {
			return err
		}
	}
	fmt.Println("Suppressed pattern", h)
	if *createSilence && cfg.AlertmanagerURL != "" {
		client := alertmanager.NewClient(cfg.AlertmanagerURL)
		sil := alertmanager.Silence{
			Matchers:  []alertmanager.Matcher{{Name: "pattern_hash", Value: h, IsRegex: false}},
			StartsAt:  time.Now(),
			EndsAt:    time.Now().Add(8760 * time.Hour), // 1 year
			CreatedBy: "ailert",
			Comment:   *reason,
		}
		id, err := client.PostSilence(sil)
		if err != nil {
			return fmt.Errorf("create Alertmanager silence: %w", err)
		}
		fmt.Println("Alertmanager silence created:", id)
	}
	return nil
}

func cmdDetectChanges(args []string) error {
	fs := flag.NewFlagSet("detect-changes", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "Config YAML")
	snapshotDir := fs.String("snapshot-dir", "", "Directory with snapshot_latest.json (default: snapshot_dir from config)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	dir := *snapshotDir
	if dir == "" {
		dir = cfg.SnapshotDir
	}
	if dir == "" {
		return fmt.Errorf("detect-changes: set -snapshot-dir or snapshot_dir in config")
	}
	prev, err := snapshot.Load(filepath.Join(dir, "snapshot_latest.json"))
	if err != nil {
		return err
	}
	st := store.New(cfg.StorePath)
	if err := st.Load(); err != nil {
		return err
	}
	list := st.ListSeen()
	cur := make([]snapshot.PatternEnt, len(list))
	for i, p := range list {
		cur[i] = snapshot.PatternEnt{Level: p.Level, Hash: p.Hash, Sample: p.Sample, Count: p.Count}
	}
	ch := changes.Detect(cur, prev)
	fmt.Println("--- New patterns ---")
	for _, p := range ch.NewPatterns {
		fmt.Printf("  %s %s count=%d %s\n", p.Level.String(), p.Hash, p.Count, truncate(p.Sample, 50))
	}
	fmt.Println("--- Gone patterns ---")
	for _, p := range ch.GonePatterns {
		fmt.Printf("  %s %s (was count=%d)\n", p.Level.String(), p.Hash, p.Count)
	}
	fmt.Println("--- Count changes ---")
	for _, d := range ch.CountDeltas {
		fmt.Printf("  %s %s %d -> %d %s\n", d.Level.String(), d.Hash, d.OldCount, d.NewCount, truncate(d.Sample, 40))
	}
	return nil
}

func cmdSuggestRules(args []string) error {
	fs := flag.NewFlagSet("suggest-rules", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "Config YAML")
	snapshotDir := fs.String("snapshot-dir", "", "Directory with snapshot_latest.json")
	threshold := fs.Int64("suppress-threshold", suppressCountThreshold, "Suggest suppress for new INFO/DEBUG when count >= this")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	dir := *snapshotDir
	if dir == "" {
		dir = cfg.SnapshotDir
	}
	if dir == "" {
		return fmt.Errorf("suggest-rules: set -snapshot-dir or snapshot_dir in config")
	}
	prev, err := snapshot.Load(filepath.Join(dir, "snapshot_latest.json"))
	if err != nil {
		return err
	}
	st := store.New(cfg.StorePath)
	if err := st.Load(); err != nil {
		return err
	}
	list := st.ListSeen()
	cur := make([]snapshot.PatternEnt, len(list))
	for i, p := range list {
		cur[i] = snapshot.PatternEnt{Level: p.Level, Hash: p.Hash, Sample: p.Sample, Count: p.Count}
	}
	ch := changes.Detect(cur, prev)
	rules := changes.SuggestRules(ch, *threshold)
	fmt.Println("--- Suggested rules ---")
	for _, r := range rules {
		fmt.Printf("  %s %s %s %s\n", r.Action, r.Hash, r.Level.String(), truncate(r.Sample, 50))
		fmt.Printf("    reason: %s\n", r.Reason)
	}
	return nil
}

func cmdApplyRule(args []string) error {
	fs := flag.NewFlagSet("apply-rule", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "Config YAML")
	reason := fs.String("reason", "applied", "Reason (for suppress)")
	createSilence := fs.Bool("create-silence", false, "Create Alertmanager silence (suppress only)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	rest := fs.Args()
	if len(rest) < 2 {
		return fmt.Errorf("apply-rule: usage: apply-rule suppress <hash> | apply-rule alert <hash>")
	}
	action, hash := rest[0], rest[1]
	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	st := store.New(cfg.StorePath)
	if err := st.Load(); err != nil {
		return err
	}
	switch action {
	case "suppress":
		st.Suppress(hash, *reason)
		if cfg.StorePath != "" {
			if err := st.Save(); err != nil {
				return err
			}
		}
		fmt.Println("Suppressed", hash)
		if *createSilence && cfg.AlertmanagerURL != "" {
			client := alertmanager.NewClient(cfg.AlertmanagerURL)
			sil := alertmanager.Silence{
				Matchers:  []alertmanager.Matcher{{Name: "pattern_hash", Value: hash, IsRegex: false}},
				StartsAt:  time.Now(),
				EndsAt:    time.Now().Add(8760 * time.Hour),
				CreatedBy: "ailert",
				Comment:   *reason,
			}
			id, err := client.PostSilence(sil)
			if err != nil {
				return err
			}
			fmt.Println("Alertmanager silence:", id)
		}
	case "alert":
		if cfg.AlertmanagerURL == "" {
			return fmt.Errorf("apply-rule alert: set alertmanager_url in config")
		}
		client := alertmanager.NewClient(cfg.AlertmanagerURL)
		list := st.ListSeen()
		var sample string
		for _, p := range list {
			if p.Hash == hash {
				sample = p.Sample
				break
			}
		}
		a := alertmanager.Alert{
			Labels:      map[string]string{"alertname": "ailert", "pattern_hash": hash, "level": "ERROR", "source": "apply-rule"},
			Annotations: map[string]string{"summary": "applied rule", "description": sample},
			StartsAt:    time.Now(),
		}
		if err := client.PostAlerts([]alertmanager.Alert{a}); err != nil {
			return err
		}
		fmt.Println("Alert sent for", hash)
	default:
		return fmt.Errorf("apply-rule: action must be suppress or alert")
	}
	return nil
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
