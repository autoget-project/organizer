package stage3planner

import (
	"context"
	"path"
	"path/filepath"
	"strings"

	"organizer/internal/model"
	"organizer/internal/ptr"
)

// PornPlanner plans non-bango porn downloads with the local naming fallback
// chain (L9): entities id -> entities name -> enriched title -> file stem.
// Targets follow porn|porn_vr/{name}/{name}{ext}; companion subtitles are left
// to Stage 4 pairing and non-media files are skipped.
type PornPlanner struct{}

// NewPornPlanner creates a new PornPlanner.
func NewPornPlanner() *PornPlanner { return &PornPlanner{} }

// Plan generates the porn move plan (pure local logic, no LLM).
func (p *PornPlanner) Plan(_ context.Context, pc *PlannerContext) ([]model.PlanAction, error) {
	videos, _, others := partitionFiles(pc.Files)

	root := string(model.TargetDirPorn)
	if pc.Metadata.IsVR {
		root = string(model.TargetDirPornVR)
	}

	actions := make([]model.PlanAction, 0, len(videos)+len(others))
	for _, v := range videos {
		name := pornDisplayName(pc, v)
		actions = append(actions, model.PlanAction{
			File:   v,
			Action: "move",
			Target: ptr.Str(path.Join(root, name, name+filepath.Ext(v))),
		})
	}
	for _, o := range others {
		actions = append(actions, model.PlanAction{File: o, Action: "skip"})
	}
	return actions, nil
}

// pornDisplayName implements the L9 naming fallback chain.
func pornDisplayName(pc *PlannerContext, video string) string {
	if pc.Entities != nil {
		if id, ok := pc.Entities["id"].(string); ok && strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
		if name, ok := pc.Entities["name"].(string); ok && strings.TrimSpace(name) != "" {
			return strings.TrimSpace(name)
		}
	}
	if t := strings.TrimSpace(pc.Metadata.Title); t != "" {
		return t
	}
	base := filepath.Base(video)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
