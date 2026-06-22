package handler

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"dmama_api/internal/model"
	"dmama_api/internal/repository"
	"dmama_api/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// FeatureHandler exposes CRUD + validation for drawn dma_boundary/flow_meter/step_test features.
// Validation + mirror run on PostGIS 16 (gisPool); the documents persist to Vallaris MongoDB (db).
type FeatureHandler struct {
	svc *service.FeatureService
}

func NewFeatureHandler(gisPool *pgxpool.Pool, featureDB, usersDB *mongo.Database, systemUser, userDomain string) *FeatureHandler {
	featureRepo := repository.NewFeatureRepo(featureDB)
	topoRepo := repository.NewTopologyRepo(gisPool)
	var userRepo *repository.UserRepo
	if usersDB != nil {
		userRepo = repository.NewUserRepo(usersDB)
	}
	return &FeatureHandler{svc: service.NewFeatureService(featureRepo, topoRepo, userRepo, systemUser, userDomain)}
}

// Validate runs topology rules without persisting (dry run for live frontend feedback).
// POST /api/features/:shape/:pwaCode/validate
func (h *FeatureHandler) Validate(c *fiber.Ctx) error {
	shape, pwaCode, ok := shapeAndBranch(c)
	if !ok {
		return c.Status(400).JSON(model.ErrorResponse("invalid shape"))
	}

	var req model.FeatureRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.ErrorResponse("invalid request body"))
	}

	result, err := h.svc.Validate(c.Context(), shape, pwaCode, &req, primitive.NilObjectID)
	if err != nil {
		return h.dbError(c, err)
	}
	return c.JSON(model.SuccessResponse(result))
}

// Create validates then stores a new feature.
// POST /api/features/:shape/:pwaCode
func (h *FeatureHandler) Create(c *fiber.Ctx) error {
	shape, pwaCode, ok := shapeAndBranch(c)
	if !ok {
		return c.Status(400).JSON(model.ErrorResponse("invalid shape"))
	}

	var req model.FeatureRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.ErrorResponse("invalid request body"))
	}

	feature, result, err := h.svc.Create(c.Context(), shape, pwaCode, &req, c.Get("X-User-Id"))
	if err != nil {
		return h.dbError(c, err)
	}
	if feature == nil {
		return c.Status(400).JSON(model.APIResponse{
			Success: false,
			Error:   "topology validation failed: " + strings.Join(result.Violations, "; "),
			Data:    result,
		})
	}
	return c.Status(201).JSON(model.SuccessResponse(feature))
}

// List returns features in a branch+shape, optionally filtered by bbox.
// GET /api/features/:shape/:pwaCode?bbox=minLng,minLat,maxLng,maxLat
func (h *FeatureHandler) List(c *fiber.Ctx) error {
	shape, pwaCode, ok := shapeAndBranch(c)
	if !ok {
		return c.Status(400).JSON(model.ErrorResponse("invalid shape"))
	}

	var bbox []float64
	if raw := c.Query("bbox"); raw != "" {
		parsed, err := parseBBox(raw)
		if err != nil {
			return c.Status(400).JSON(model.ErrorResponse(err.Error()))
		}
		bbox = parsed
	}

	features, err := h.svc.List(c.Context(), shape, pwaCode, bbox)
	if err != nil {
		return h.dbError(c, err)
	}
	return c.JSON(model.SuccessWithCount(features, len(features)))
}

// Get returns a single feature by id.
// GET /api/features/:shape/:pwaCode/:id
func (h *FeatureHandler) Get(c *fiber.Ctx) error {
	shape, pwaCode, ok := shapeAndBranch(c)
	if !ok {
		return c.Status(400).JSON(model.ErrorResponse("invalid shape"))
	}
	id, ok := parseObjectID(c)
	if !ok {
		return c.Status(400).JSON(model.ErrorResponse("invalid id"))
	}

	feature, err := h.svc.GetByID(c.Context(), shape, pwaCode, id)
	if err != nil {
		return h.dbError(c, err)
	}
	if feature == nil {
		return c.Status(404).JSON(model.ErrorResponse("feature not found"))
	}
	return c.JSON(model.SuccessResponse(feature))
}

// Update re-validates then replaces an existing feature.
// PUT /api/features/:shape/:pwaCode/:id
func (h *FeatureHandler) Update(c *fiber.Ctx) error {
	shape, pwaCode, ok := shapeAndBranch(c)
	if !ok {
		return c.Status(400).JSON(model.ErrorResponse("invalid shape"))
	}
	id, ok := parseObjectID(c)
	if !ok {
		return c.Status(400).JSON(model.ErrorResponse("invalid id"))
	}

	var req model.FeatureRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(model.ErrorResponse("invalid request body"))
	}

	feature, result, found, err := h.svc.Update(c.Context(), shape, pwaCode, id, &req, c.Get("X-User-Id"))
	if err != nil {
		return h.dbError(c, err)
	}
	if !found {
		return c.Status(404).JSON(model.ErrorResponse("feature not found"))
	}
	if feature == nil {
		return c.Status(400).JSON(model.APIResponse{
			Success: false,
			Error:   "topology validation failed: " + strings.Join(result.Violations, "; "),
			Data:    result,
		})
	}
	return c.JSON(model.SuccessResponse(feature))
}

// Delete removes a feature by id.
// DELETE /api/features/:shape/:pwaCode/:id
func (h *FeatureHandler) Delete(c *fiber.Ctx) error {
	shape, pwaCode, ok := shapeAndBranch(c)
	if !ok {
		return c.Status(400).JSON(model.ErrorResponse("invalid shape"))
	}
	id, ok := parseObjectID(c)
	if !ok {
		return c.Status(400).JSON(model.ErrorResponse("invalid id"))
	}

	found, err := h.svc.Delete(c.Context(), shape, pwaCode, id)
	if err != nil {
		return h.dbError(c, err)
	}
	if !found {
		return c.Status(404).JSON(model.ErrorResponse("feature not found"))
	}
	return c.JSON(model.SuccessResponse(fiber.Map{"deleted": id.Hex()}))
}

// Sync backfills the dmama_layer mirror for a branch+shape from MongoDB.
// POST /api/features/:shape/:pwaCode/sync
func (h *FeatureHandler) Sync(c *fiber.Ctx) error {
	shape, pwaCode, ok := shapeAndBranch(c)
	if !ok {
		return c.Status(400).JSON(model.ErrorResponse("invalid shape"))
	}

	n, err := h.svc.Sync(c.Context(), shape, pwaCode)
	if err != nil {
		return h.dbError(c, err)
	}
	return c.JSON(model.SuccessResponse(fiber.Map{"synced": n}))
}

// FeatureUnavailable answers every feature route with a clear 503 when the feature backend is not
// ready: PostgreSQL 16 / MongoDB unconfigured or unreachable, or the dmama_layer schema/tables not
// provisioned. Registered in place of the real routes so misconfiguration surfaces as an explicit
// message instead of Fiber's "Cannot POST" 404 or a per-request 42P01.
func FeatureUnavailable(c *fiber.Ctx) error {
	return c.Status(503).JSON(model.ErrorResponse(
		"feature CRUD unavailable: PostgreSQL 16 (GISDATA_URL) / MongoDB not configured, " +
			"or dmama_layer schema not ready — run migrations/0001_dmama_layer.sql"))
}

// dbError maps a known not-found to 404, everything else to 500.
func (h *FeatureHandler) dbError(c *fiber.Ctx, err error) error {
	if errors.Is(err, repository.ErrCollectionNotFound) {
		return c.Status(404).JSON(model.ErrorResponse(err.Error()))
	}
	return c.Status(500).JSON(model.ErrorResponse(err.Error()))
}

// shapeAndBranch reads :shape (validated against the allowed set) and :pwaCode.
func shapeAndBranch(c *fiber.Ctx) (shape, pwaCode string, ok bool) {
	shape = c.Params("shape")
	if !model.AllowedShapes[shape] {
		return "", "", false
	}
	return shape, c.Params("pwaCode"), true
}

func parseObjectID(c *fiber.Ctx) (primitive.ObjectID, bool) {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return primitive.NilObjectID, false
	}
	return id, true
}

func parseBBox(raw string) ([]float64, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return nil, fmt.Errorf("bbox must be minLng,minLat,maxLng,maxLat")
	}
	bbox := make([]float64, 4)
	for i, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, fmt.Errorf("invalid bbox value: %s", p)
		}
		bbox[i] = v
	}
	return bbox, nil
}
