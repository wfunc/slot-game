package repository

import (
	"context"
	
	"gorm.io/gorm"
)

// BaseRepository 基础仓储接口
type BaseRepository interface {
	// GetDB 获取数据库实例
	GetDB() *gorm.DB
	// WithTx 使用事务
	WithTx(tx *gorm.DB) BaseRepository
}

// Pagination 分页参数
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int64 `json:"total"`
}

// NewPagination 创建分页参数
func NewPagination(page, pageSize int) *Pagination {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return &Pagination{
		Page:     page,
		PageSize: pageSize,
	}
}

// Offset 计算偏移量
func (p *Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}

// Paginate 分页查询
func Paginate(p *Pagination) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset(p.Offset()).Limit(p.PageSize)
	}
}

// BaseRepo 基础仓储实现
type BaseRepo struct {
	db *gorm.DB
}

// NewBaseRepo 创建基础仓储
func NewBaseRepo(db *gorm.DB) *BaseRepo {
	return &BaseRepo{db: db}
}

// GetDB 获取数据库实例
func (r *BaseRepo) GetDB() *gorm.DB {
	return r.db
}

// WithTx 使用事务
func (r *BaseRepo) WithTx(tx *gorm.DB) *BaseRepo {
	return &BaseRepo{db: tx}
}

// Transaction 执行事务
func (r *BaseRepo) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}