package main

import (
	"context"
	"log"

	"dmama_api/internal/config"
	"dmama_api/internal/database"
	"dmama_api/internal/middleware"
	"dmama_api/internal/repository"
	"dmama_api/internal/router"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Feature CRUD + usage logging need PostgreSQL 16 (topology engine, dmama_layer mirror,
	// auth_logs.dmama_use) and Vallaris/claystone MongoDB. All are optional: when a backend is
	// unreachable we log a warning and keep serving the existing read-only spatial endpoints.
	gisPool := connectGIS(ctx, cfg)
	if gisPool != nil {
		defer gisPool.Close()
	}

	usageRec := startUsageRecorder(ctx, gisPool)
	featureDB, usersDB := connectMongo(ctx, cfg)

	// Feature CRUD needs both backends AND the dmama_layer schema/tables to exist. EnsureSchema is
	// best-effort here: if it fails (app role lacks DDL, PostGIS missing, etc.) we log loudly and
	// leave featureReady false so the routes return a clear 503 instead of a per-request 42P01.
	featureReady := false
	if gisPool != nil {
		if err := repository.NewTopologyRepo(gisPool).EnsureSchema(ctx); err != nil {
			log.Printf("WARN: dmama_layer schema not ready; feature CRUD disabled "+
				"(run migrations/0001_dmama_layer.sql): %v", err)
		} else if featureDB != nil {
			featureReady = true
		}
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"success": false,
				"error":   err.Error(),
			})
		},
	})

	middleware.Setup(app, cfg)
	router.Setup(app, pool, gisPool, featureDB, usersDB, usageRec, featureReady, cfg)

	log.Printf("Server starting on port %s", cfg.Port)
	log.Fatal(app.Listen(":" + cfg.Port))
}

// connectGIS opens the PostgreSQL 16 pool. Returns nil (with a warning) when GISDATA_URL is unset
// or the database is unreachable. The dmama_layer schema is ensured by the caller so its result can
// gate feature route registration.
func connectGIS(ctx context.Context, cfg *config.Config) *pgxpool.Pool {
	if cfg.GISDataURL == "" {
		log.Printf("WARN: GISDATA_URL not set; feature CRUD + usage logging disabled")
		return nil
	}
	gisPool, err := database.NewPool(ctx, cfg.GISDataURL)
	if err != nil {
		log.Printf("WARN: PostgreSQL 16 (gis_data) unavailable; feature CRUD + usage logging disabled: %v", err)
		return nil
	}
	return gisPool
}

// startUsageRecorder ensures auth_logs.dmama_use and starts the background writer. Returns nil
// (usage logging disabled) when there is no PG16 pool or the schema cannot be created.
func startUsageRecorder(ctx context.Context, gisPool *pgxpool.Pool) *repository.UsageRecorder {
	if gisPool == nil {
		return nil
	}
	rec := repository.NewUsageRecorder(gisPool, 1024)
	if err := rec.EnsureSchema(ctx); err != nil {
		log.Printf("WARN: failed to ensure auth_logs schema; usage logging disabled: %v", err)
		return nil
	}
	rec.Start(context.Background())
	return rec
}

// connectMongo opens the shared Mongo client and returns the Vallaris feature database and the
// claystone user directory. Returns (nil, nil) when MONGO_URI is unset or the server is unreachable.
func connectMongo(ctx context.Context, cfg *config.Config) (featureDB, usersDB *mongo.Database) {
	if cfg.MongoURI == "" {
		log.Printf("WARN: MONGO_URI not set; feature CRUD disabled")
		return nil, nil
	}
	client, err := database.NewMongoClient(ctx, cfg.MongoURI)
	if err != nil {
		log.Printf("WARN: MongoDB unavailable; feature CRUD disabled: %v", err)
		return nil, nil
	}
	return client.Database(cfg.MongoDB), client.Database(cfg.MongoUsersDB)
}
