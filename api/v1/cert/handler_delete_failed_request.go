package cert

import (
	"net/http"
	"strconv"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
)

// DeleteFailedCertificateRequest deletes a failed certificate request
// POST /api/v1/acme/certificate/requests/:requestId/delete
//
// Only allows deletion of requests that meet ALL of the following conditions:
// 1. status = "failed"
// 2. result_certificate_id IS NULL (no certificate generated)
//
// Any request that does not meet these conditions will be rejected.
func (h *Handler) DeleteFailedCertificateRequest(c *gin.Context) {
	requestIDStr := c.Param("requestId")
	requestID, err := strconv.Atoi(requestIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpx.Response{
			Code:    1001,
			Message: "invalid request ID",
			Data:    nil,
		})
		return
	}

	// 1. Query certificate_requests table
	var request model.CertificateRequest
	if err := h.db.Where("id = ?", requestID).First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, httpx.Response{
			Code:    1004,
			Message: "certificate request not found",
			Data:    nil,
		})
		return
	}

	// 2. Validate deletion conditions (must check in order)
	// Condition 1: status must be "failed"
	if request.Status != "failed" {
		c.JSON(http.StatusBadRequest, httpx.Response{
			Code:    3003,
			Message: "only failed certificate request can be deleted",
			Data:    nil,
		})
		return
	}

	// Condition 2: result_certificate_id must be NULL (no certificate generated)
	if request.ResultCertificateID != nil {
		c.JSON(http.StatusBadRequest, httpx.Response{
			Code:    3003,
			Message: "certificate already issued, request cannot be deleted",
			Data:    nil,
		})
		return
	}

	// 3. Physical deletion (no soft delete, no archive)
	// This terminates the certificate request lifecycle
	if err := h.db.Delete(&request).Error; err != nil {
		c.JSON(http.StatusInternalServerError, httpx.Response{
			Code:    1005,
			Message: "failed to delete certificate request",
			Data:    nil,
		})
		return
	}

	// 4. Success response
	c.JSON(http.StatusOK, httpx.Response{
		Code:    0,
		Message: "certificate request terminated",
		Data: map[string]interface{}{
			"requestId": requestID,
		},
	})
}
