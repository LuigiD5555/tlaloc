// Package prototypelab is the public, small experimental-recording surface
// for development systems that consume Tlaloc (for example TONAL). It exposes
// the common Episode + RunManifest + Summary bundle without exposing Tlaloc's
// internal experiment packages.
//
// Target-specific/native raw evidence remains authoritative. This package is
// only the reusable projection used to carry experience between prototype
// iterations.
package prototypelab

import (
	"time"

	"tlaloc.local/behaviorlab/internal/episode"
	"tlaloc.local/behaviorlab/internal/experimentalspine"
)

const (
	EpisodeSchema  = episode.Schema
	ManifestSchema = experimentalspine.ManifestSchema
	SummarySchema  = experimentalspine.SummarySchema
)

type Episode = episode.Episode
type Step = episode.Step
type Cost = episode.Cost

type RunManifest = experimentalspine.RunManifest
type Prototype = experimentalspine.Prototype
type Repos = experimentalspine.Repos
type Model = experimentalspine.Model

type Summary = experimentalspine.Summary
type CostSummary = experimentalspine.CostSummary
type LatencySummary = experimentalspine.LatencySummary
type Count = experimentalspine.Count
type EpisodeBreakdown = experimentalspine.EpisodeBreakdown
type CapabilityBreakdown = experimentalspine.CapabilityBreakdown
type BundlePaths = experimentalspine.BundlePaths

// Summarize deterministically reduces a prototype's Episodes. It makes no
// model/network calls and does not perform promotion.
func Summarize(manifest RunManifest, episodes []Episode) Summary {
	return experimentalspine.Summarize(manifest, episodes)
}

// WriteBundle atomically publishes <outDir>/experience containing manifest,
// immutable Episodes and summary. Existing bundles are never overwritten.
func WriteBundle(outDir string, manifest RunManifest, episodes []Episode, observedAt time.Time) (BundlePaths, error) {
	return experimentalspine.WriteBundle(outDir, manifest, episodes, observedAt)
}

func StoreEpisode(root string, ep Episode, observedAt time.Time) (string, error) {
	return episode.Store(root, ep, observedAt)
}

func StoreEpisodes(root string, episodes []Episode, observedAt time.Time) ([]string, error) {
	return episode.StoreAll(root, episodes, observedAt)
}

func LoadEpisode(path string) (Episode, error) {
	return episode.Load(path)
}
