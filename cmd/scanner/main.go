package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/serhankarakoc/turkish-open-source/internal/config"
	"github.com/serhankarakoc/turkish-open-source/internal/discovery"
	"github.com/serhankarakoc/turkish-open-source/internal/framework"
	"github.com/serhankarakoc/turkish-open-source/internal/generator"
	gh "github.com/serhankarakoc/turkish-open-source/internal/github"
	"github.com/serhankarakoc/turkish-open-source/internal/project"
)

func main() {
	log.SetFlags(0)
	opts := parseFlags()
	if err := run(context.Background(), opts); err != nil {
		log.Fatal(err)
	}
}

type options struct {
	DryRun     bool
	Discover   bool
	Update     bool
	Frameworks bool
	Generate   bool
	Validate   bool
	Verbose    bool
	Root       string
}

func parseFlags() options {
	var opts options
	flag.BoolVar(&opts.DryRun, "dry-run", false, "query GitHub and print a report without writing files")
	flag.BoolVar(&opts.Discover, "discover", false, "run discovery searches")
	flag.BoolVar(&opts.Update, "update", false, "refresh existing catalog entries from GitHub")
	flag.BoolVar(&opts.Frameworks, "frameworks", false, "scan curated framework seeds from config/frameworks.yml")
	flag.BoolVar(&opts.Generate, "generate", false, "regenerate README from data/projects.json")
	flag.BoolVar(&opts.Validate, "validate", false, "validate the dataset without calling GitHub")
	flag.BoolVar(&opts.Verbose, "verbose", false, "log scanner progress (never logs tokens)")
	flag.StringVar(&opts.Root, "root", "", "repository root (defaults to current directory)")
	flag.Parse()

	explicit := false
	flag.CommandLine.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "discover", "update", "frameworks", "generate", "validate":
			explicit = true
		}
	})
	if !explicit {
		opts.Discover = true
		opts.Update = true
		opts.Frameworks = true
		opts.Generate = true
	}
	if opts.DryRun && !explicit {
		opts.Discover = true
		opts.Update = true
		opts.Frameworks = true
		opts.Generate = true
	}
	return opts
}

type stdLogger struct {
	verbose bool
}

func (l stdLogger) Printf(format string, v ...any) {
	if !l.verbose {
		return
	}
	log.Printf(format, v...)
}

func run(ctx context.Context, opts options) error {
	root, err := findRoot(opts.Root)
	if err != nil {
		return err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return err
	}
	logger := stdLogger{verbose: opts.Verbose}
	now := time.Now().UTC()

	dataDir := filepath.Join(root, "data")
	projectsPath := filepath.Join(dataDir, "projects.json")
	categoriesPath := filepath.Join(dataDir, "categories.json")
	frameworksPath := filepath.Join(dataDir, "frameworks.json")
	readmePath := filepath.Join(root, "README.md")

	existing, err := project.LoadDataset(projectsPath)
	if err != nil {
		return err
	}
	existingFrameworks, err := framework.LoadDataset(frameworksPath)
	if err != nil {
		return err
	}

	catalogAPI := opts.Discover || opts.Update
	frameworkAPI := opts.Frameworks

	if opts.Validate && !catalogAPI && !frameworkAPI && !opts.Generate {
		return validateAll(existing, existingFrameworks, cfg)
	}
	if !catalogAPI && !frameworkAPI {
		return generateOnly(existing, existingFrameworks.Frameworks, cfg, readmePath, categoriesPath, now, opts.DryRun)
	}

	token := strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	if catalogAPI && token == "" {
		if frameworkAPI {
			fmt.Fprintln(os.Stderr, "GITHUB_TOKEN is not set; skipping catalog discovery/update. Scanning framework seeds with the public API.")
			catalogAPI = false
			opts.Discover = false
			opts.Update = false
		} else {
			return fmt.Errorf("GITHUB_TOKEN is not set; discovery/update require GitHub API access")
		}
	}

	client, err := gh.NewClient(gh.Options{
		BaseURL:        cfg.GitHub.APIURL,
		APIVersion:     cfg.GitHub.APIVersion,
		Token:          token,
		Timeout:        time.Duration(cfg.Scanner.RequestTimeoutSeconds) * time.Second,
		MaxRetries:     cfg.Scanner.MaxRetries,
		InitialBackoff: time.Duration(cfg.Scanner.RetryBackoffMilliseconds) * time.Millisecond,
		MaxBackoff:     time.Duration(cfg.Scanner.MaxBackoffSeconds) * time.Second,
		Logger:         logger,
	})
	if err != nil {
		return err
	}

	merged := existing.Projects
	discReport := discovery.Report{}
	eval := discovery.EvalStats{}
	changes := changeSet{}
	if catalogAPI {
		set := discovery.NewSet()
		if opts.Discover {
			set, discReport, err = discovery.Discover(ctx, client, cfg, logger)
			if err != nil {
				return err
			}
		}
		if opts.Update {
			for _, p := range existing.Projects {
				if p.Owner == "" || p.Name == "" {
					continue
				}
				repo, err := client.GetRepository(ctx, p.Owner, p.Name)
				if err != nil {
					if gh.IsNotFound(err) {
						logger.Printf("existing %s not found", p.FullName)
						continue
					}
					if gh.IsRateLimited(err) {
						return fmt.Errorf("GitHub API rate limit exceeded. Set GITHUB_TOKEN for a higher quota: %w", err)
					}
					logger.Printf("refresh %s: %v", p.FullName, err)
					continue
				}
				set.AddRepository(*repo, "existing")
			}
		}
		accepted, stats, err := discovery.Evaluate(ctx, client, cfg, set.Candidates(), now, logger)
		if err != nil {
			return err
		}
		eval = stats
		merged = project.MergeAll(existing.Projects, accepted, true)
		changes = diffChanges(existing.Projects, merged)
		printReport(discReport, eval, changes, client)
	}

	frameworks := existingFrameworks.Frameworks
	fwReport := framework.Report{}
	if frameworkAPI {
		seeds, err := framework.LoadSeeds(root)
		if err != nil {
			return err
		}
		scanned, report, err := framework.Scan(ctx, client, cfg, seeds, now, logger)
		if err != nil {
			return err
		}
		frameworks = scanned
		fwReport = report
		printFrameworkReport(fwReport, client)
	}

	if opts.Validate {
		ds := project.Dataset{Version: project.DatasetVersion, GeneratedAt: now.Format(time.RFC3339), Projects: merged}
		fwds := framework.Dataset{Version: framework.DatasetVersion, GeneratedAt: now.Format(time.RFC3339), Frameworks: frameworks}
		if err := validateAll(ds, fwds, cfg); err != nil {
			return err
		}
	}

	if opts.DryRun {
		return nil
	}

	if catalogAPI {
		ds := project.Dataset{
			Version:     project.DatasetVersion,
			GeneratedAt: now.Format(time.RFC3339),
			Projects:    merged,
		}
		if err := project.SaveDataset(projectsPath, ds); err != nil {
			return err
		}
		cats := generator.BuildCategoryFile(merged, cfg, ds.GeneratedAt)
		raw, err := project.MarshalJSON(cats)
		if err != nil {
			return err
		}
		if err := os.WriteFile(categoriesPath, raw, 0o644); err != nil {
			return err
		}
	}

	if frameworkAPI {
		fwds := framework.Dataset{
			Version:     framework.DatasetVersion,
			GeneratedAt: now.Format(time.RFC3339),
			Frameworks:  frameworks,
		}
		if err := framework.SaveDataset(frameworksPath, fwds); err != nil {
			return err
		}
	}

	if opts.Generate {
		existingREADME, _ := os.ReadFile(readmePath)
		if err := generator.WriteREADME(readmePath, string(existingREADME), merged, frameworks, cfg, now); err != nil {
			return err
		}
	}
	return nil
}

func validateAll(ds project.Dataset, fw framework.Dataset, cfg *config.Config) error {
	errs := project.ValidateDataset(ds, cfg.CategoryKeys())
	errs = append(errs, framework.ValidateDataset(fw)...)
	if len(errs) == 0 {
		fmt.Printf("dataset ok: %d projects, %d frameworks\n", len(ds.Projects), len(fw.Frameworks))
		return nil
	}
	for _, err := range errs {
		fmt.Fprintf(os.Stderr, "invalid: %v\n", err)
	}
	return fmt.Errorf("dataset validation failed: %d error(s)", len(errs))
}

func generateOnly(ds project.Dataset, frameworks []framework.Framework, cfg *config.Config, readmePath, categoriesPath string, now time.Time, dryRun bool) error {
	if dryRun {
		fmt.Println("dry-run: would regenerate README.md and data/categories.json")
		return nil
	}
	cats := generator.BuildCategoryFile(ds.Projects, cfg, now.Format(time.RFC3339))
	raw, err := project.MarshalJSON(cats)
	if err != nil {
		return err
	}
	if err := os.WriteFile(categoriesPath, raw, 0o644); err != nil {
		return err
	}
	existingREADME, _ := os.ReadFile(readmePath)
	return generator.WriteREADME(readmePath, string(existingREADME), ds.Projects, frameworks, cfg, now)
}

type changeSet struct {
	New     int
	Updated int
	Removed int
}

func diffChanges(before, after []project.Project) changeSet {
	prev := project.IndexByID(before)
	next := project.IndexByID(after)
	var cs changeSet
	for id := range next {
		if _, ok := prev[id]; !ok {
			cs.New++
		} else {
			cs.Updated++
		}
	}
	for id, old := range prev {
		if _, ok := next[id]; !ok {
			if project.IsCommunityVerified(old) {
				continue
			}
			cs.Removed++
		}
	}
	return cs
}

func printReport(disc discovery.Report, eval discovery.EvalStats, changes changeSet, client *gh.Client) {
	fmt.Println("Turkish Open Source Scanner")
	fmt.Println()
	fmt.Println("Discovery:")
	fmt.Printf("  Queries: %d\n", disc.Queries)
	if disc.InvalidQueries > 0 {
		fmt.Printf("  Invalid queries: %d\n", disc.InvalidQueries)
	}
	fmt.Printf("  Candidates: %d\n", disc.RawResults+disc.UserReposEnumerated)
	fmt.Printf("  Unique repositories: %d\n", disc.UniqueRepositories)
	fmt.Println()
	fmt.Println("Validation:")
	fmt.Printf("  Open source: %d\n", eval.OpenSource)
	fmt.Printf("  Rejected: %d\n", eval.Rejected)
	fmt.Println()
	fmt.Println("Turkey:")
	fmt.Printf("  Verified: %d\n", eval.Verified)
	fmt.Printf("  Likely: %d\n", eval.Likely)
	fmt.Printf("  Needs review: %d\n", eval.NeedsReview)
	fmt.Printf("  Excluded: %d\n", eval.Excluded)
	fmt.Println()
	fmt.Println("Changes:")
	fmt.Printf("  New: %d\n", changes.New)
	fmt.Printf("  Updated: %d\n", changes.Updated)
	fmt.Printf("  Removed: %d\n", changes.Removed)
	fmt.Println()
	fmt.Println("API:")
	if client != nil {
		fmt.Printf("  Requests: %d\n", client.RequestCount())
		snap := client.RateLimit()
		if snap.Remaining >= 0 {
			fmt.Printf("  Remaining: %d\n", snap.Remaining)
		} else {
			fmt.Printf("  Remaining: unknown\n")
		}
	}
}

func printFrameworkReport(rep framework.Report, client *gh.Client) {
	fmt.Println("Featured Frameworks")
	fmt.Println()
	fmt.Printf("  Total: %d\n", rep.Total)
	fmt.Printf("  verified: %d\n", rep.Verified)
	fmt.Printf("  pending_verification: %d\n", rep.PendingVerification)
	fmt.Printf("  historical: %d\n", rep.Historical)
	fmt.Printf("  repository_not_found: %d\n", rep.RepositoryNotFound)
	fmt.Printf("  excluded: %d\n", rep.Excluded)
	if rep.InvalidURLs > 0 {
		fmt.Printf("  invalid URLs: %d\n", rep.InvalidURLs)
	}
	fmt.Println()
	if len(rep.Frameworks) > 0 {
		fmt.Println("  Name | Language | Stars | License | Status | GitHub")
		for _, f := range rep.Frameworks {
			lang := f.Language
			if lang == "" {
				lang = "-"
			}
			fmt.Printf("  %s | %s | %d | %s | %s | %s\n", f.Name, lang, f.Stars, f.License, f.Status, f.GitHub)
		}
		fmt.Println()
	}
	if client != nil {
		fmt.Printf("  API requests: %d\n", client.RequestCount())
		snap := client.RateLimit()
		if snap.Remaining >= 0 {
			fmt.Printf("  API remaining: %d\n", snap.Remaining)
			if snap.Remaining == 0 {
				fmt.Fprintln(os.Stderr, "GitHub API rate limit exhausted. Set GITHUB_TOKEN for a higher quota.")
			}
		}
		fmt.Println()
	}
}

func findRoot(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "config", "settings.yml")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return wd, nil
		}
		dir = parent
	}
}
