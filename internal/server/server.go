package server

// API_SOURCE

import (
	"github.com/magooney-loon/pb-ext/core/server/api"
	"github.com/pocketbase/pocketbase/core"

	"arcadia/internal/server/collections"
	"arcadia/internal/server/handlers"
)

// RegisterRoutes registers all API routes.
func RegisterRoutes(app core.App) {
	handlers.RegisterRoutes(app)
}

// RegisterCollections registers all collection schemas.
func RegisterCollections(app core.App) {
	collections.RegisterCollections(app)
}

// InitVersionedSystem creates the versioned API router.
func InitVersionedSystem() *api.APIVersionManager {
	return handlers.InitVersionedSystem()
}
