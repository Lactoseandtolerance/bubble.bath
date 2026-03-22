package handlers

import (
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/Lactoseandtolerance/bubble-bath/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func NewRouter(authH *AuthHandler, verifyH *VerifyHandler, rdb *redis.Client, maxLoginAttempts int) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.SetHeader("Content-Type", "application/json"))

	r.Get("/health", Health)

	rl := middleware.NewRateLimiter(rdb, maxLoginAttempts)

	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Use(rl.Middleware)
			r.Post("/signup", authH.Signup)
			r.Post("/login/direct", authH.LoginDirect)
		})
		r.Get("/verify", verifyH.Verify)
	})

	return r
}
