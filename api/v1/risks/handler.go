package risks

import (
	"go_cmdb/internal/httpx"
	"go_cmdb/internal/risk"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler 风险API处理器
type Handler struct {
	service *risk.Service
}

// NewHandler 创建风险API处理器
func NewHandler(db *gorm.DB) *Handler {
	return &Handler{
		service: risk.NewService(db),
	}
}

// ListRisks 查询风险列表（全局）
// GET /api/v1/risks
func (h *Handler) ListRisks(c *gin.Context) {
	var filter risk.ListRisksFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	result, err := h.service.ListRisks(filter)
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError(err.Error(), err))
		return
	}

	httpx.OK(c, result)
}

// ListWebsiteRisks 查询网站的风险列表
// GET /api/v1/websites/:id/risks
func (h *Handler) ListWebsiteRisks(c *gin.Context) {
	websiteID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid website id"))
		return
	}

	risks, err := h.service.ListWebsiteRisks(websiteID)
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError(err.Error(), err))
		return
	}

	httpx.OK(c, gin.H{
		"website_id": websiteID,
		"risks":      risks,
		"count":      len(risks),
	})
}

// ListCertificateRisks 查询证书的风险列表
// GET /api/v1/certificates/:id/risks
func (h *Handler) ListCertificateRisks(c *gin.Context) {
	certificateID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid certificate id"))
		return
	}

	risks, err := h.service.ListCertificateRisks(certificateID)
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError(err.Error(), err))
		return
	}

	httpx.OK(c, gin.H{
		"certificate_id": certificateID,
		"risks":          risks,
		"count":          len(risks),
	})
}

// ResolveRisk 解决风险（人工标记）
// POST /api/v1/risks/:id/resolve
func (h *Handler) ResolveRisk(c *gin.Context) {
	riskID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid risk id"))
		return
	}

	if err := h.service.ResolveRisk(riskID); err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError(err.Error(), err))
		return
	}

	httpx.OK(c, gin.H{
		"risk_id": riskID,
		"message": "Risk resolved successfully",
	})
}

// PrecheckHTTPS 前置风险预检
// POST /api/v1/websites/:id/precheck/https
func (h *Handler) PrecheckHTTPS(c *gin.Context) {
	websiteID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid("invalid website id"))
		return
	}

	var req risk.PrecheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.FailErr(c, httpx.ErrParamInvalid(err.Error()))
		return
	}

	result, err := h.service.PrecheckHTTPS(websiteID, req)
	if err != nil {
		httpx.FailErr(c, httpx.ErrDatabaseError(err.Error(), err))
		return
	}

	httpx.OK(c, result)
}
