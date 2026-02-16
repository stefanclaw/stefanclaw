package update

import (
	"context"
	"fmt"
	"runtime"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
)

const repo = "stefanclaw/stefanclaw"

// Result holds the outcome of an update check or apply.
type Result struct {
	CurrentVersion string
	LatestVersion  string
	UpdateAvailable bool
	Applied         bool
}

// Check queries GitHub for the latest release and reports whether an update is
// available. It does not download or replace anything.
func Check(ctx context.Context, currentVersion string) (*Result, error) {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("creating github source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source: source,
	})
	if err != nil {
		return nil, fmt.Errorf("creating updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(repo))
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}

	res := &Result{
		CurrentVersion: currentVersion,
	}
	if found {
		res.LatestVersion = latest.Version()
		if _, err := semver.NewVersion(currentVersion); err != nil {
			// Not valid semver (e.g. "dev") — any release is newer
			res.UpdateAvailable = true
		} else {
			res.UpdateAvailable = latest.GreaterThan(currentVersion)
		}
	}
	return res, nil
}

// Apply downloads and installs the latest release, replacing the current
// binary in-place.
func Apply(ctx context.Context, currentVersion string) (*Result, error) {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("creating github source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
	})
	if err != nil {
		return nil, fmt.Errorf("creating updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(repo))
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}

	res := &Result{
		CurrentVersion: currentVersion,
	}

	_, semverErr := semver.NewVersion(currentVersion)
	isValidSemver := semverErr == nil

	if !found || (isValidSemver && !latest.GreaterThan(currentVersion)) {
		if found {
			res.LatestVersion = latest.Version()
		}
		return res, nil
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return nil, fmt.Errorf("finding executable path: %w", err)
	}

	if err := updater.UpdateTo(ctx, latest, exe); err != nil {
		return nil, fmt.Errorf("applying update: %w", err)
	}

	res.LatestVersion = latest.Version()
	res.UpdateAvailable = true
	res.Applied = true
	return res, nil
}
