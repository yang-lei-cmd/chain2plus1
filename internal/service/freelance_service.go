// Package service 灵活用工服务 (Phase 5)
package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/linqi/chain2plus1/pkg/database"
	"github.com/linqi/chain2plus1/pkg/dto"
	"github.com/linqi/chain2plus1/pkg/model"
)

// FreelanceService 灵活用工服务
type FreelanceService struct{}

// NewFreelanceService 创建灵活用工服务实例
func NewFreelanceService() *FreelanceService {
	return &FreelanceService{}
}

// RegisterFreelancer 自由职业者注册
func (s *FreelanceService) RegisterFreelancer(c *gin.Context, userID uint, req *dto.FreelancerRegisterReq) (*model.Freelancer, error) {
	db := database.DB

	// 检查身份证号是否重复
	var existing model.Freelancer
	if err := db.Where("id_card = ?", req.IDCard).First(&existing).Error; err == nil {
		return nil, errors.New("该身份证已注册")
	}

	// 创建自由职业者记录
	freelancer := &model.Freelancer{
		UserID:     userID,
		RealName:   req.RealName,
		IDCard:     req.IDCard,
		Phone:      req.Phone,
		Email:      req.Email,
		SkillTags:  fmt.Sprintf("%v", req.SkillTags),
		Bio:        req.Bio,
		Status:     "pending",
	}

	if err := db.Create(freelancer).Error; err != nil {
		return nil, fmt.Errorf("注册失败: %v", err)
	}

	return freelancer, nil
}

// ApproveFreelancer 审核通过自由职业者
func (s *FreelanceService) ApproveFreelancer(freelancerID uint, approvedBy uint) error {
	db := database.DB
	now := time.Now()

	return db.Model(&model.Freelancer{}).
		Where("id = ?", freelancerID).
		Updates(map[string]interface{}{
			"status":       "approved",
			"approved_by":  approvedBy,
			"approved_at":  now,
		}).Error
}

// RejectFreelancer 审核拒绝自由职业者
func (s *FreelanceService) RejectFreelancer(freelancerID uint, rejectReason string, approvedBy uint) error {
	db := database.DB
	now := time.Now()

	return db.Model(&model.Freelancer{}).
		Where("id = ?", freelancerID).
		Updates(map[string]interface{}{
			"status":        "rejected",
			"reject_reason": rejectReason,
			"approved_by":   approvedBy,
			"approved_at":   now,
		}).Error
}

// GetFreelancerProfile 获取自由职业者资料
func (s *FreelanceService) GetFreelancerProfile(freelancerID uint) (*model.Freelancer, error) {
	var freelancer model.Freelancer
	if err := database.DB.First(&freelancer, freelancerID).Error; err != nil {
		return nil, errors.New("自由职业者不存在")
	}
	return &freelancer, nil
}

// CreateTask 创建任务
func (s *FreelanceService) CreateTask(c *gin.Context, publisherID uint, req *dto.TaskCreateReq) (*model.Task, error) {
	db := database.DB

	task := &model.Task{
		Title:         req.Title,
		Description:   req.Description,
		Category:      req.Category,
		Budget:        req.Budget,
		DurationHours: req.DurationHours,
		PublisherID:   publisherID,
		Status:        "open",
		Deadline:      req.Deadline,
	}

	if err := db.Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建任务失败: %v", err)
	}

	return task, nil
}

// GetTaskList 获取任务列表
func (s *FreelanceService) GetTaskList(req *dto.TaskListReq) ([]model.Task, int64, error) {
	db := database.DB
	var tasks []model.Task
	var total int64

	query := db.Model(&model.Task{})

	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Category != "" {
		query = query.Where("category = ?", req.Category)
	}
	if req.MinBudget > 0 {
		query = query.Where("budget >= ?", req.MinBudget)
	}
	if req.MaxBudget > 0 {
		query = query.Where("budget <= ?", req.MaxBudget)
	}

	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// GetTaskDetail 获取任务详情
func (s *FreelanceService) GetTaskDetail(taskID uint) (*model.Task, error) {
	var task model.Task
	if err := database.DB.Preload("Publisher").Preload("AssignedFreelancer").First(&task, taskID).Error; err != nil {
		return nil, errors.New("任务不存在")
	}
	return &task, nil
}

// AssignTask 分配任务给自由职业者
func (s *FreelanceService) AssignTask(taskID uint, freelancerID uint) error {
	db := database.DB
	now := time.Now()

	return db.Model(&model.Task{}).
		Where("id = ? AND status = ?", taskID, "open").
		Updates(map[string]interface{}{
			"status":     "assigned",
			"assigned_to": freelancerID,
			"updated_at": now,
		}).Error
}

// SubmitWork 提交工作成果
func (s *FreelanceService) SubmitWork(taskID uint, submission string) error {
	db := database.DB
	now := time.Now()

	return db.Model(&model.Task{}).
		Where("id = ? AND status = 'assigned'", taskID).
		Updates(map[string]interface{}{
			"status":     "submitted",
			"submission": submission,
			"updated_at": now,
		}).Error
}

// ReviewWork 审核工作成果
func (s *FreelanceService) ReviewWork(taskID uint, reviewerID uint, approved bool, comment string) error {
	db := database.DB
	now := time.Now()

	status := "reviewed"
	if approved {
		status = "completed"
	}

	updateData := map[string]interface{}{
		"status":        status,
		"reviewer_id":   reviewerID,
		"reviewed_at":   now,
		"review_comment": comment,
	}

	if approved {
		updateData["paid_at"] = now
	}

	return db.Model(&model.Task{}).Where("id = ?", taskID).Updates(updateData).Error
}

// CreateTimeLog 创建工时记录
func (s *FreelanceService) CreateTimeLog(c *gin.Context, freelancerID uint, req *dto.TimeLogCreateReq) (*model.TimeLog, error) {
	db := database.DB

	// 获取 Task 的 PublisherID
	var task model.Task
	if err := db.First(&task, req.TaskID).Error; err != nil {
		return nil, errors.New("任务不存在")
	}

	timeLog := &model.TimeLog{
		TaskID:       req.TaskID,
		FreelancerID: freelancerID,
		Hours:        req.Hours,
		Date:         req.Date,
		Content:      req.Content,
		Status:       "pending",
	}

	if err := db.Create(timeLog).Error; err != nil {
		return nil, fmt.Errorf("创建工时记录失败: %v", err)
	}

	return timeLog, nil
}

// CreateSettlement 创建薪资结算
func (s *FreelanceService) CreateSettlement(c *gin.Context, freelancerID uint, req *dto.SettlementCreateReq) (*model.Settlement, error) {
	db := database.DB

	// 验证任务是否存在且已完成
	var task model.Task
	if err := db.Where("id = ? AND status = 'completed'", req.TaskID).First(&task).Error; err != nil {
		return nil, errors.New("任务不存在或未审核通过")
	}

	// 检查是否已结算
	var existingSettlement model.Settlement
	if err := db.Where("task_id = ?", req.TaskID).First(&existingSettlement).Error; err == nil {
		return nil, errors.New("该任务已结算")
	}

	settlement := &model.Settlement{
		TaskID:       req.TaskID,
		FreelancerID: freelancerID,
		Amount:       task.Budget,
		TaxAmount:    0,
		PlatformFee:  0,
		NetAmount:    task.Budget,
		PaymentMethod: req.PaymentMethod,
		BankName:     req.BankName,
		AccountName:  req.AccountName,
		AccountNo:    req.AccountNo,
		Status:       "pending",
	}

	if err := db.Create(settlement).Error; err != nil {
		return nil, fmt.Errorf("创建结算记录失败: %v", err)
	}

	return settlement, nil
}

// CreateRating 创建评分
func (s *FreelanceService) CreateRating(c *gin.Context, publisherID uint, freelancerID uint, req *dto.RatingCreateReq) (*model.Rating, error) {
	db := database.DB

	// 验证任务是否存在且已完成
	var task model.Task
	if err := db.Where("id = ? AND status = 'completed' AND publisher_id = ?", req.TaskID, publisherID).First(&task).Error; err != nil {
		return nil, errors.New("任务不存在或无权评分")
	}

	// 检查是否已经评过
	var existingRating model.Rating
	if err := db.Where("task_id = ?", req.TaskID).First(&existingRating).Error; err == nil {
		return nil, errors.New("该任务已评分")
	}

	rating := &model.Rating{
		TaskID:       req.TaskID,
		FreelancerID: freelancerID,
		PublisherID:  publisherID,
		Score:        req.Score,
		Comment:      req.Comment,
	}

	if err := db.Create(rating).Error; err != nil {
		return nil, fmt.Errorf("创建评分失败: %v", err)
	}

	// 更新自由职业者的平均评分和完成数
	var avgRating float64
	var totalJobs int
	db.Model(&model.Rating{}).Where("freelancer_id = ?", freelancerID).
		Select("AVG(score) as avg_score, COUNT(*) as total").
		Row().Scan(&avgRating, &totalJobs)

	db.Model(&model.Freelancer{}).
		Where("id = ?", freelancerID).
		Updates(map[string]interface{}{
			"avg_rating": avgRating,
			"total_jobs": totalJobs,
		})

	return rating, nil
}

// GetRatingList 获取评分列表
func (s *FreelanceService) GetRatingList(req *dto.RatingListReq) ([]model.Rating, int64, error) {
	db := database.DB
	var ratings []model.Rating
	var total int64

	query := db.Model(&model.Rating{})

	if req.FreelancerID > 0 {
		query = query.Where("freelancer_id = ?", req.FreelancerID)
	}
	if req.PublisherID > 0 {
		query = query.Where("publisher_id = ?", req.PublisherID)
	}
	if req.TaskID > 0 {
		query = query.Where("task_id = ?", req.TaskID)
	}

	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&ratings).Error; err != nil {
		return nil, 0, err
	}

	return ratings, total, nil
}

// GetRatingStats 获取评分统计
func (s *FreelanceService) GetRatingStats(freelancerID uint) (*dto.RatingStatsResp, error) {
	db := database.DB

	stats := &dto.RatingStatsResp{
		FreelancerID: freelancerID,
	}

	// 获取平均评分和总数
	var avgRating float64
	var totalCount int
	db.Model(&model.Rating{}).Where("freelancer_id = ?", freelancerID).
		Select("AVG(score) as avg_score, COUNT(*) as total").
		Row().Scan(&avgRating, &totalCount)

	stats.AvgRating = avgRating
	stats.TotalRatings = totalCount

	// 获取各星级数量
	type StarCount struct {
		Score int
		Count int
	}
	var starCounts []StarCount
	db.Model(&model.Rating{}).Where("freelancer_id = ?", freelancerID).
		Group("score").
		Pluck("score, COUNT(*) as cnt", &starCounts)

	for _, sc := range starCounts {
		switch sc.Score {
		case 5:
			stats.FiveStar = sc.Count
		case 4:
			stats.FourStar = sc.Count
		case 3:
			stats.ThreeStar = sc.Count
		case 2:
			stats.TwoStar = sc.Count
		case 1:
			stats.OneStar = sc.Count
		}
	}

	return stats, nil
}
