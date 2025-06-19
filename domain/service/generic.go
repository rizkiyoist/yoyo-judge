package service

import (
	"yoyo-judge/library/helper"

	"gorm.io/gorm"
)

type Repository[T any] interface {
	Create(model *T, tx *gorm.DB) (*T, error)
	Update(model *T, tx *gorm.DB) error
	Delete(model *T, tx *gorm.DB) error
	FindOneBy(criteria map[string]interface{}) (*T, error)
	FindBy(criteria map[string]interface{}, page, size int) ([]*T, int, error)
	Count(criteria map[string]interface{}) int
}

type repository[T any] struct {
	db *gorm.DB
}

func NewService[T any](db *gorm.DB) Repository[T] {
	return &repository[T]{db}
}

func (r *repository[T]) Create(model *T, tx *gorm.DB) (*T, error) {
	if err := tx.Create(model).Error; err != nil {
		return nil, err
	}
	return model, nil
}

func (r *repository[T]) Update(model *T, tx *gorm.DB) error {
	return tx.Save(model).Error
}

func (r *repository[T]) Delete(model *T, tx *gorm.DB) error {
	return tx.Delete(model).Error
}

func (r *repository[T]) FindOneBy(criteria map[string]interface{}) (*T, error) {
	m := new(T)
	if err := r.db.Where(criteria).First(m).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (r *repository[T]) FindBy(criteria map[string]interface{}, page, size int) ([]*T, int, error) {
	var data []*T
	limit, offset := helper.GetLimitOffset(page, size)

	if err := r.db.Where(criteria).Offset(offset).Order("id DESC").Limit(limit).Find(&data).Error; err != nil {
		return nil, 0, err
	}

	var total int64
	if err := r.db.Model(new(T)).Where(criteria).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	return data, int(total), nil
}

func (r *repository[T]) Count(criteria map[string]interface{}) int {
	var total int64
	if err := r.db.Model(new(T)).Where(criteria).Count(&total).Error; err != nil {
		return 0
	}
	return int(total)
}
