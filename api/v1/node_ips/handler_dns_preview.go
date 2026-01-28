package node_ips

import (
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"
	"strconv"

	"github.com/gin-gonic/gin"
)

// DNSPreview handles GET /api/v1/node-ips/:id/dns-preview
func (h *Handler) DNSPreview(c *gin.Context) {
	// Get IP ID from path parameter
	idStr := c.Param("id")
	ipID, err := strconv.Atoi(idStr)
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid IP ID"))
		return
	}

	// Build preview (assuming IP is enabled, so desiredState=present)
	previews, err := h.service.DNSLinker.BuildDesiredRecords(ipID, model.DNSRecordDesiredStatePresent)
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError("failed to build DNS preview", err))
		return
	}

	httpx.OK(c, gin.H{
		"items": previews,
	})
}
