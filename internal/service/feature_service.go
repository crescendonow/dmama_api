package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// FeatureService validates drawn features against topology rules (PostGIS 16) and persists the
// valid ones to MongoDB (Vallaris, source of truth) while mirroring them into dmama_layer.
type FeatureService struct {
	features   *repository.FeatureRepo
	topology   *repository.TopologyRepo
	users      *repository.UserRepo // claystone user directory; may be nil
	userDomain string
	systemUser primitive.ObjectID
}

// NewFeatureService wires the repositories. _createdBy/_updatedBy are resolved by looking up
// "<X-User-Id>@<userDomain>" in the claystone users directory; systemUser (an ObjectID hex) is the
// fallback when there is no user header, the lookup misses, or users is nil.
func NewFeatureService(features *repository.FeatureRepo, topology *repository.TopologyRepo, users *repository.UserRepo, systemUser, userDomain string) *FeatureService {
	uid, err := primitive.ObjectIDFromHex(systemUser)
	if err != nil {
		uid = primitive.NilObjectID
	}
	return &FeatureService{features: features, topology: topology, users: users, userDomain: userDomain, systemUser: uid}
}

// Validate runs the topology rules for a shape without persisting. excludeID is the id of the
// feature being updated (skipped during overlap comparison); pass NilObjectID on create/dry-run.
func (s *FeatureService) Validate(ctx context.Context, shape, pwaCode string, req *model.FeatureRequest, excludeID primitive.ObjectID) (*model.ValidationResult, error) {
	result := &model.ValidationResult{Valid: true}

	geomStr, err := geometryJSON(req.Geometry)
	if err != nil {
		result.Violations = append(result.Violations, err.Error())
		result.Valid = false
		return result, nil
	}

	if want := model.GeometryTypeForShape[shape]; want != "" {
		if gt, _ := req.Geometry["type"].(string); gt != want {
			result.Violations = append(result.Violations,
				fmt.Sprintf("geometry type %q does not match shape %q (expected %s)", gt, shape, want))
		}
	}

	valid, reason, err := s.topology.CheckValidity(ctx, geomStr)
	if err != nil {
		return nil, fmt.Errorf("validity check: %w", err)
	}
	if !valid {
		result.Violations = append(result.Violations, "invalid geometry: "+reason)
	}

	switch {
	case shape == model.ShapeDmaBoundary:
		exclude := ""
		if !excludeID.IsZero() {
			exclude = excludeID.Hex()
		}
		overlaps, err := s.topology.OverlapsExisting(ctx, shape, pwaCode, geomStr, exclude)
		if err != nil {
			return nil, fmt.Errorf("overlap check: %w", err)
		}
		if overlaps {
			result.Violations = append(result.Violations, "geometry overlaps an existing dma_boundary in the same branch")
		}

	case usesDmaCoverageRule(shape):
		within, hasCoverage, err := s.topology.WithinDmaCoverage(ctx, pwaCode, geomStr)
		if err != nil {
			return nil, fmt.Errorf("within-coverage check: %w", err)
		}
		if !hasCoverage {
			result.Warnings = append(result.Warnings,
				"no dma_boundary mirrored for this branch; within-coverage rule not checked (run sync first)")
		} else if !within {
			result.Violations = append(result.Violations, outsideDmaCoverageMessage(shape))
		}
	}

	result.Valid = len(result.Violations) == 0
	return result, nil
}

// Create validates then persists a new feature. On validation failure the returned feature is nil
// and the result carries the violations.
func (s *FeatureService) Create(ctx context.Context, shape, pwaCode string, req *model.FeatureRequest, userID string) (*model.Feature, *model.ValidationResult, error) {
	result, err := s.Validate(ctx, shape, pwaCode, req, primitive.NilObjectID)
	if err != nil {
		return nil, nil, err
	}

	dmaID := 0
	if shape == model.ShapeDmaBoundary {
		dmaID, err = s.assignDmaID(ctx, pwaCode, req.DmaID, primitive.NilObjectID, result)
		if err != nil {
			return nil, nil, err
		}
	}

	if !result.Valid {
		return nil, result, nil
	}

	id := primitive.NewObjectID()
	user := s.resolveUser(ctx, userID)
	now := time.Now().UTC()

	props := mergeProps(nil, req.Properties)
	props["_id"] = id
	props["pwaCode"] = pwaCode
	if shape == model.ShapeDmaBoundary {
		props["dmaId"] = dmaID
	}
	props["_createdAt"] = now
	props["_createdBy"] = user
	props["_updatedAt"] = now
	props["_updatedBy"] = user

	f := &model.Feature{ID: id, Type: "Feature", Geometry: req.Geometry, Properties: props}

	if err := s.features.Insert(ctx, pwaCode, shape, f); err != nil {
		return nil, nil, err
	}
	if err := s.mirror(ctx, shape, pwaCode, f); err != nil {
		return nil, nil, fmt.Errorf("mongo insert succeeded but dmama_layer mirror failed: %w", err)
	}
	return f, result, nil
}

// Update re-validates then replaces an existing feature. found is false when id does not exist.
func (s *FeatureService) Update(ctx context.Context, shape, pwaCode string, id primitive.ObjectID, req *model.FeatureRequest, userID string) (*model.Feature, *model.ValidationResult, bool, error) {
	existing, err := s.features.GetByID(ctx, pwaCode, shape, id)
	if err != nil {
		return nil, nil, false, err
	}
	if existing == nil {
		return nil, nil, false, nil
	}

	result, err := s.Validate(ctx, shape, pwaCode, req, id)
	if err != nil {
		return nil, nil, true, err
	}

	dmaID := model.AsInt(existing.Properties["dmaId"])
	if shape == model.ShapeDmaBoundary && req.DmaID > 0 {
		dmaID, err = s.assignDmaID(ctx, pwaCode, req.DmaID, id, result)
		if err != nil {
			return nil, nil, true, err
		}
	}

	if !result.Valid {
		return nil, result, true, nil
	}

	// Preserve created metadata, overlay client changes, then refresh server fields.
	props := mergeProps(existing.Properties, req.Properties)
	props["_id"] = id
	props["pwaCode"] = pwaCode
	if shape == model.ShapeDmaBoundary {
		props["dmaId"] = dmaID
	}
	props["_updatedAt"] = time.Now().UTC()
	props["_updatedBy"] = s.resolveUser(ctx, userID)

	f := &model.Feature{ID: id, Type: "Feature", Geometry: req.Geometry, Properties: props}

	found, err := s.features.Update(ctx, pwaCode, shape, id, f)
	if err != nil {
		return nil, nil, false, err
	}
	if !found {
		return nil, nil, false, nil
	}
	if err := s.mirror(ctx, shape, pwaCode, f); err != nil {
		return nil, nil, true, fmt.Errorf("mongo update succeeded but dmama_layer mirror failed: %w", err)
	}
	return f, result, true, nil
}

func (s *FeatureService) GetByID(ctx context.Context, shape, pwaCode string, id primitive.ObjectID) (*model.Feature, error) {
	return s.features.GetByID(ctx, pwaCode, shape, id)
}

func (s *FeatureService) List(ctx context.Context, shape, pwaCode string, bbox []float64) ([]model.Feature, error) {
	return s.features.List(ctx, pwaCode, shape, bbox)
}

// Delete removes a feature from Mongo and its mirror. found is false when id does not exist.
func (s *FeatureService) Delete(ctx context.Context, shape, pwaCode string, id primitive.ObjectID) (bool, error) {
	found, err := s.features.Delete(ctx, pwaCode, shape, id)
	if err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}
	if err := s.topology.Delete(ctx, shape, id.Hex()); err != nil {
		return true, fmt.Errorf("mongo delete succeeded but dmama_layer mirror delete failed: %w", err)
	}
	return true, nil
}

// Sync rebuilds the dmama_layer mirror for a branch+shape from MongoDB. Returns the count synced.
func (s *FeatureService) Sync(ctx context.Context, shape, pwaCode string) (int, error) {
	features, err := s.features.List(ctx, pwaCode, shape, nil)
	if err != nil {
		return 0, err
	}
	for i := range features {
		if err := s.mirror(ctx, shape, pwaCode, &features[i]); err != nil {
			return i, err
		}
	}
	return len(features), nil
}

func usesDmaCoverageRule(shape string) bool {
	return shape == model.ShapeFlowMeter || shape == model.ShapeStepTest
}

func outsideDmaCoverageMessage(shape string) string {
	if shape == model.ShapeStepTest {
		return "polygon is outside the branch dma_boundary coverage"
	}
	return "point is outside the branch dma_boundary coverage"
}

// assignDmaID returns the running dma_id to use. A client-supplied value is validated for
// uniqueness (recording a violation on conflict); otherwise the next max+1 is allocated.
func (s *FeatureService) assignDmaID(ctx context.Context, pwaCode string, requested int, exclude primitive.ObjectID, result *model.ValidationResult) (int, error) {
	if requested > 0 {
		exists, err := s.features.DmaIDExists(ctx, pwaCode, requested, exclude)
		if err != nil {
			return 0, err
		}
		if exists {
			result.Violations = append(result.Violations,
				fmt.Sprintf("dma_id %d already exists in branch %s", requested, pwaCode))
			result.Valid = false
		}
		return requested, nil
	}
	max, err := s.features.MaxDmaID(ctx, pwaCode)
	if err != nil {
		return 0, err
	}
	return max + 1, nil
}

// mirror upserts a feature into its dmama_layer table.
func (s *FeatureService) mirror(ctx context.Context, shape, pwaCode string, f *model.Feature) error {
	geomStr, err := geometryJSON(f.Geometry)
	if err != nil {
		return err
	}
	propsJSON, err := json.Marshal(f.Properties)
	if err != nil {
		return fmt.Errorf("marshal properties: %w", err)
	}

	var dmaPtr *int
	if shape == model.ShapeDmaBoundary {
		d := model.AsInt(f.Properties["dmaId"])
		dmaPtr = &d
	}
	// Prefer the document's own pwaCode; fall back to the request branch.
	code := pwaCode
	if v, ok := f.Properties["pwaCode"].(string); ok && v != "" {
		code = v
	}
	return s.topology.Upsert(ctx, shape, f.ID.Hex(), code, dmaPtr, string(propsJSON), geomStr)
}

// resolveUser maps an X-User-Id (e.g. "10090") to a claystone users._id by looking up
// "<userID>@<userDomain>". Falls back to systemUser on empty id, nil directory, miss, or error.
func (s *FeatureService) resolveUser(ctx context.Context, userID string) primitive.ObjectID {
	if userID != "" && s.users != nil && s.userDomain != "" {
		username := repository.Username(userID, s.userDomain)
		if id, found, err := s.users.LookupID(ctx, username); err == nil && found {
			return id
		}
	}
	return s.systemUser
}

// mergeProps returns a shallow copy of base with overlay applied on top.
func mergeProps(base, overlay map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

func geometryJSON(geom map[string]interface{}) (string, error) {
	if len(geom) == 0 {
		return "", fmt.Errorf("geometry is required")
	}
	b, err := json.Marshal(geom)
	if err != nil {
		return "", fmt.Errorf("invalid geometry: %w", err)
	}
	return string(b), nil
}
