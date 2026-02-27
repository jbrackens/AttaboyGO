package handler

import (
	"net/http"

	"github.com/attaboy/platform/internal/service"
	"github.com/go-chi/chi/v5"
)

// PluginHandler handles plugin dispatch endpoints.
type PluginHandler struct {
	pluginSvc *service.PluginService
}

// NewPluginHandler creates a new PluginHandler.
func NewPluginHandler(pluginSvc *service.PluginService) *PluginHandler {
	return &PluginHandler{pluginSvc: pluginSvc}
}

// Dispatch handles POST /plugins/dispatch.
func (h *PluginHandler) Dispatch(w http.ResponseWriter, r *http.Request) {
	var input service.DispatchInput
	if err := DecodeJSON(r, &input); err != nil {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "invalid request body",
		})
		return
	}

	result, err := h.pluginSvc.Dispatch(r.Context(), input)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, result)
}

// ListPlugins handles GET /plugins.
func (h *PluginHandler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	plugins, err := h.pluginSvc.ListPlugins(r.Context())
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, plugins)
}

// ListDispatches handles GET /plugins/{pluginID}/dispatches.
func (h *PluginHandler) ListDispatches(w http.ResponseWriter, r *http.Request) {
	pluginID := chi.URLParam(r, "pluginID")
	if pluginID == "" {
		RespondJSON(w, http.StatusBadRequest, map[string]string{
			"code": "VALIDATION_ERROR", "message": "plugin_id required",
		})
		return
	}

	dispatches, err := h.pluginSvc.ListDispatches(r.Context(), pluginID)
	if err != nil {
		RespondError(w, err)
		return
	}

	RespondJSON(w, http.StatusOK, dispatches)
}
