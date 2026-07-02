// Package handler Phase 5: 灵活用工 Handler
package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/linqi/chain2plus1/internal/event"
	"github.com/linqi/chain2plus1/internal/service"
	"github.com/linqi/chain2plus1/pkg/database"
	"github.com/linqi/chain2plus1/pkg/dto"
	"github.com/linqi/chain2plus1/pkg/model"
)

// FreelanceHandler 灵活用工处理器
type FreelanceHandler struct {
	freelanceService *service.FreelanceService
	hub              *event.Hub
}

// NewFreelanceHandler 创建灵活用工处理器
func NewFreelanceHandler() *FreelanceHandler {
	return &FreelanceHandler{
		freelanceService: service.NewFreelanceService(),
		hub:              nil,
	}
}

// SetHub 设置 WebSocket Hub 引用
func (h *FreelanceHandler) SetHub(hub *event.Hub) {
	h.hub = hub
}

// RegisterFreelancer 自由职业者注册
// POST /api/v1/freelancer/register
func (h *FreelanceHandler) RegisterFreelancer(c *gin.Context) {
	var req dto.FreelancerRegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	freelancer, err := h.freelanceService.RegisterFreelancer(c, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "注册成功,等待审核",
		"freelancer": freelancer,
	})
}

// ApproveFreelancer 审核通过
// PATCH /api/v1/admin/freelancer/:id/approve
func (h *FreelanceHandler) ApproveFreelancer(c *gin.Context) {
	freelancerID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	approvedBy := c.GetUint("user_id")

	if err := h.freelanceService.ApproveFreelancer(uint(freelancerID), approvedBy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Phase 6: Emit freelancer_approved event to the applicant
	if h.hub != nil {
		h.hub.SendNotification(
			uint(freelancerID),
			event.FreelancerApproved,
			"您的自由职业者申请已通过审核",
			map[string]interface{}{
				"freelancer_id": freelancerID,
			},
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "审核通过",
	})
}

// RejectFreelancer 审核拒绝
// PATCH /api/v1/admin/freelancer/:id/reject
func (h *FreelanceHandler) RejectFreelancer(c *gin.Context) {
	freelancerID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	approvedBy := c.GetUint("user_id")

	var req struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.freelanceService.RejectFreelancer(uint(freelancerID), req.Reason, approvedBy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Phase 6: Emit freelancer_rejected event to the applicant
	if h.hub != nil {
		h.hub.SendNotification(
			uint(freelancerID),
			event.FreelancerRejected,
			"您的自由职业者申请未通过审核",
			map[string]interface{}{
				"freelancer_id": freelancerID,
				"reason":        req.Reason,
			},
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "已拒绝",
	})
}

// GetFreelancerProfile 获取自由职业者资料
// GET /api/v1/freelancer/:id
func (h *FreelanceHandler) GetFreelancerProfile(c *gin.Context) {
	freelancerID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	profile, err := h.freelanceService.GetFreelancerProfile(uint(freelancerID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"freelancer": profile,
	})
}

// CreateTask 创建任务
// POST /api/v1/task/create
func (h *FreelanceHandler) CreateTask(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req dto.TaskCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.freelanceService.CreateTask(c, userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Phase 6: Emit task_published event (broadcast to all)
	if h.hub != nil {
		h.hub.SendNotification(
			0,
			event.TaskPublished,
			"一个新任务已发布",
			map[string]interface{}{
				"task_id":     task.ID,
				"task_title":  task.Title,
				"category":    task.Category,
				"budget":      task.Budget,
				"duration":    task.DurationHours,
			},
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "任务创建成功",
		"task":    task,
	})
}

// GetTaskList 获取任务列表
// GET /api/v1/task/list
func (h *FreelanceHandler) GetTaskList(c *gin.Context) {
	var req dto.TaskListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tasks, total, err := h.freelanceService.GetTaskList(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tasks":    tasks,
		"pagination": gin.H{
			"total":  total,
			"page":   req.Page,
			"pages":  (total + int64(req.PageSize) - 1) / int64(req.PageSize),
		},
	})
}

// GetTaskDetail 获取任务详情
// GET /api/v1/task/:id
func (h *FreelanceHandler) GetTaskDetail(c *gin.Context) {
	taskID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	detail, err := h.freelanceService.GetTaskDetail(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"task": detail,
	})
}

// AssignTask 分配任务
// POST /api/v1/task/assign
func (h *FreelanceHandler) AssignTask(c *gin.Context) {
	var req dto.TaskAssignReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.freelanceService.AssignTask(req.TaskID, req.FreelancerID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Phase 6: Emit task_assigned event to the freelancer
	if h.hub != nil {
		taskDetail, _ := h.freelanceService.GetTaskDetail(req.TaskID)
		h.hub.SendNotification(
			req.FreelancerID,
			event.TaskAssigned,
			"你被分配了一个新任务",
			map[string]interface{}{
				"task_id":    taskDetail.ID,
				"task_title": taskDetail.Title,
				"budget":     taskDetail.Budget,
			},
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "任务已分配给自由职业者",
	})
}

// SubmitWork 提交工作成果
// POST /api/v1/task/:id/submit
func (h *FreelanceHandler) SubmitWork(c *gin.Context) {
	taskID, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var req struct {
		Submission string `json:"submission" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.freelanceService.SubmitWork(uint(taskID), req.Submission); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Phase 6: Emit work_submitted event to the task publisher
	if h.hub != nil {
		taskDetail, _ := h.freelanceService.GetTaskDetail(uint(taskID))
		h.hub.SendNotification(
			taskDetail.PublisherID,
			event.WorkSubmitted,
			"自由职业者已提交工作成果",
			map[string]interface{}{
				"task_id":    taskDetail.ID,
				"task_title": taskDetail.Title,
			},
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "工作成果已提交,等待审核",
	})
}

// ReviewWork 审核工作
// POST /api/v1/task/:id/review
func (h *FreelanceHandler) ReviewWork(c *gin.Context) {
	taskID, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	reviewerID := c.GetUint("user_id")

	var req struct {
		Approved  bool   `json:"approved" binding:"required"`
		Comment   string `json:"comment" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.freelanceService.ReviewWork(uint(taskID), reviewerID, req.Approved, req.Comment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Phase 6: Emit event based on approval result
	if h.hub != nil {
		taskDetail, _ := h.freelanceService.GetTaskDetail(uint(taskID))
		eventType := event.TaskApproved
		message := "你的工作成果已通过审核"
		freelancerID := uint(0)
		if taskDetail.AssignedTo != nil {
			freelancerID = *taskDetail.AssignedTo
		}
		if !req.Approved {
			eventType = event.TaskRejected
			message = "你的工作成果未通过审核"
		}
		h.hub.SendNotification(
			freelancerID,
			eventType,
			message,
			map[string]interface{}{
				"task_id":      taskDetail.ID,
				"task_title":   taskDetail.Title,
				"review_comment": req.Comment,
				"approved":     req.Approved,
			},
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "审核完成",
	})
}

// CreateTimeLog 创建工时记录
// POST /api/v1/time-log
func (h *FreelanceHandler) CreateTimeLog(c *gin.Context) {
	var req dto.TimeLogCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	// 先通过 user_id 查 freelancer 的 ID
	var freelancer model.Freelancer
	if err := database.DB.Where("user_id = ?", userID).First(&freelancer).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先注册为灵活就业人员"})
		return
	}

	timeLog, err := h.freelanceService.CreateTimeLog(c, freelancer.ID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "工时记录已创建",
		"time_log": timeLog,
	})
}

// CreateSettlement 创建结算
// POST /api/v1/settlement
func (h *FreelanceHandler) CreateSettlement(c *gin.Context) {
	userID := c.GetUint("user_id")
	// 先通过 user_id 查 freelancer 的 ID
	var freelancer model.Freelancer
	if err := database.DB.Where("user_id = ?", userID).First(&freelancer).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请先注册为灵活就业人员"})
		return
	}

	var req dto.SettlementCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	settlement, err := h.freelanceService.CreateSettlement(c, freelancer.ID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "结算记录已创建",
		"settlement": settlement,
	})
}

// ListTimeLogs 查询工时记录列表
// GET /api/v1/timelog/list
func (h *FreelanceHandler) ListTimeLogs(c *gin.Context) {
	userID := c.GetUint("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	db := database.DB
	var timeLogs []model.TimeLog
	var total int64

	query := db.Model(&model.TimeLog{}).Where("freelancer_id IN (SELECT id FROM freelancer WHERE user_id = ?)", userID)
	query.Count(&total)
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&timeLogs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"time_logs": timeLogs,
		"pagination": gin.H{
			"total": total,
			"page":  page,
			"pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// ListSettlements 查询结算记录列表
// GET /api/v1/settlement/list
func (h *FreelanceHandler) ListSettlements(c *gin.Context) {
	userID := c.GetUint("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	db := database.DB
	var settlements []model.Settlement
	var total int64

	query := db.Model(&model.Settlement{}).Where("freelancer_id IN (SELECT id FROM freelancer WHERE user_id = ?)", userID)
	query.Count(&total)
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&settlements).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"settlements": settlements,
		"pagination": gin.H{
			"total": total,
			"page":  page,
			"pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// ListFreelancers 查询灵活用工人员列表 (管理员用)
// GET /api/v1/admin/freelancers
func (h *FreelanceHandler) ListFreelancers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	db := database.DB
	var freelancers []model.Freelancer
	var total int64

	query := db.Model(&model.Freelancer{})
	query.Count(&total)
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&freelancers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"freelancers": freelancers,
		"pagination": gin.H{
			"total": total,
			"page":  page,
			"pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// CreateRating 创建评分
// POST /api/v1/rating
func (h *FreelanceHandler) CreateRating(c *gin.Context) {
	publisherID := c.GetUint("user_id")

	var req dto.RatingCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rating, err := h.freelanceService.CreateRating(c, publisherID, req.FreelancerID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Phase 6: Emit rating_created event to the freelancer
	if h.hub != nil {
		h.hub.SendNotification(
			req.FreelancerID,
			event.RatingCreated,
			"你收到了一个新的评分",
			map[string]interface{}{
				"rating_id": rating.ID,
				"score":     rating.Score,
				"comment":   rating.Comment,
			},
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "评分已提交",
		"rating":  rating,
	})
}

// ListRatings 获取评分列表
// GET /api/v1/rating/list
func (h *FreelanceHandler) ListRatings(c *gin.Context) {
	var req dto.RatingListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ratings, total, err := h.freelanceService.GetRatingList(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ratings":    ratings,
		"pagination": gin.H{
			"total":  total,
			"page":   req.Page,
			"pages":  (total + int64(req.PageSize) - 1) / int64(req.PageSize),
		},
	})
}

// GetRatingStats 获取评分统计
// GET /api/v1/rating/stats/:freelancer_id
func (h *FreelanceHandler) GetRatingStats(c *gin.Context) {
	freelancerID, _ := strconv.ParseUint(c.Param("freelancer_id"), 10, 64)

	stats, err := h.freelanceService.GetRatingStats(uint(freelancerID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}
