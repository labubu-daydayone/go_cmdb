package cert

import (
	"net/http"
	"strings"
	"strconv"

	"go_cmdb/internal/httpx"
	"go_cmdb/internal/model"

	"github.com/gin-gonic/gin"
)

// DeleteLifecycleItem deletes a certificate or certificate request from the lifecycle
// POST /api/v1/certificate/:id/delete
//
// Unified deletion endpoint that handles both:
// - cert:{certificateId} - Delete certificate (only if not in use)
// - req:{requestId} - Delete certificate request (only if status=failed)
func (h *Handler) DeleteLifecycleItem(c *gin.Context) {
	id := c.Param("id")
	
	// Parse unified ID format
	parts := strings.SplitN(id, ":", 2)
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, httpx.Response{
			Code:    1001,
			Message: "invalid id format, expected cert:{id} or req:{id}",
			Data:    nil,
		})
		return
	}

	itemType := parts[0]
	itemIDStr := parts[1]
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, httpx.Response{
			Code:    1001,
			Message: "invalid id number",
			Data:    nil,
		})
		return
	}

	switch itemType {
	case "cert":
		h.deleteCertificate(c, itemID)
	case "req":
		h.deleteRequest(c, itemID)
	default:
		c.JSON(http.StatusBadRequest, httpx.Response{
			Code:    1001,
			Message: "invalid item type, expected cert or req",
			Data:    nil,
		})
	}
}

// deleteCertificate deletes a certificate if it's not in use
func (h *Handler) deleteCertificate(c *gin.Context, certificateID int) {
	// 1. Check if certificate exists
	var cert model.Certificate
	if err := h.db.Where("id = ?", certificateID).First(&cert).Error; err != nil {
		c.JSON(http.StatusNotFound, httpx.Response{
			Code:    1004,
			Message: "certificate not found",
			Data:    nil,
		})
		return
	}

	// 2. Check if certificate is in use (active bindings)
	var bindingCount int64
	h.db.Model(&model.CertificateBinding{}).
		Where("certificate_id = ? AND bind_type = ? AND is_active = ?", certificateID, "website", true).
		Count(&bindingCount)

	if bindingCount > 0 {
		c.JSON(http.StatusConflict, httpx.Response{
			Code:    3003,
			Message: "certificate is in use",
			Data:    nil,
		})
		return
	}

	// 3. Delete in transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete certificate_requests that reference this certificate (set result_certificate_id to NULL)
	if err := tx.Model(&model.CertificateRequest{}).Where("result_certificate_id = ?", certificateID).Update("result_certificate_id", nil).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, httpx.Response{
			Code:    1005,
			Message: "failed to update certificate requests",
			Data:    nil,
		})
		return
	}

	// Delete certificate_domains
	if err := tx.Where("certificate_id = ?", certificateID).Delete(&model.CertificateDomain{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, httpx.Response{
			Code:    1005,
			Message: "failed to delete certificate domains",
			Data:    nil,
		})
		return
	}

	// Delete certificate_bindings (including historical bindings)
	if err := tx.Where("certificate_id = ?", certificateID).Delete(&model.CertificateBinding{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, httpx.Response{
			Code:    1005,
			Message: "failed to delete certificate bindings",
			Data:    nil,
		})
		return
	}

	// Delete certificate
	if err := tx.Delete(&cert).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, httpx.Response{
			Code:    1005,
			Message: "failed to delete certificate",
			Data:    nil,
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, httpx.Response{
			Code:    1005,
			Message: "failed to commit transaction",
			Data:    nil,
		})
		return
	}

	// 4. Success response
	c.JSON(http.StatusOK, httpx.Response{
		Code:    0,
		Message: "certificate deleted",
		Data: map[string]interface{}{
			"id":      "cert:" + strconv.Itoa(certificateID),
			"deleted": true,
		},
	})
}

// deleteRequest deletes a certificate request if it's in failed status
func (h *Handler) deleteRequest(c *gin.Context, requestID int) {
	// 1. Check if request exists
	var request model.CertificateRequest
	if err := h.db.Where("id = ?", requestID).First(&request).Error; err != nil {
		c.JSON(http.StatusNotFound, httpx.Response{
			Code:    1004,
			Message: "certificate request not found",
			Data:    nil,
		})
		return
	}

	// 2. Check if request is deletable (only failed status)
	if request.Status != "failed" {
		c.JSON(http.StatusConflict, httpx.Response{
			Code:    3003,
			Message: "request is not deletable",
			Data:    nil,
		})
		return
	}

	// 3. Delete request (hard delete)
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
		Message: "certificate request deleted",
		Data: map[string]interface{}{
			"id":      "req:" + strconv.Itoa(requestID),
			"deleted": true,
		},
	})
}
