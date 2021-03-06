package api

import (
	"encoding/json"
	"net/http"

	"salsa.debian.org/autodeb-team/autodeb/internal/http/middleware"
	"salsa.debian.org/autodeb-team/autodeb/internal/server/appctx"
	"salsa.debian.org/autodeb-team/autodeb/internal/server/handlers/middleware/auth"
	"salsa.debian.org/autodeb-team/autodeb/internal/server/models"
)

//UserGetHandler returns a handler returns the current user
func UserGetHandler(appCtx *appctx.Context) http.Handler {

	handlerFunc := func(w http.ResponseWriter, r *http.Request, user *models.User) {
		if err := json.NewEncoder(w).Encode(user); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			appCtx.RequestLogger().Error(r, err)
			return
		}
	}

	handler := auth.WithUserOr403(handlerFunc, appCtx)

	handler = middleware.JSONHeaders(handler)

	return handler
}
