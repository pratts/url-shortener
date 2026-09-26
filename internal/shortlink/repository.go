package shortlink

import (
	"context"
	"errors"

	"shortener/internal/platform/pgerr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormRepository stores links in Postgres.
type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, l *Link) error {
	err := r.db.WithContext(ctx).Create(l).Error
	if pgerr.IsUniqueViolation(err, "short_code") {
		return ErrCodeTaken
	}
	return err
}

func (r *GormRepository) Get(ctx context.Context, id, owner uint64) (Link, error) {
	var l Link
	err := r.db.WithContext(ctx).Where("id = ? AND created_by = ?", id, owner).First(&l).Error
	return l, notFound(err)
}

func (r *GormRepository) ByCode(ctx context.Context, code string) (Link, error) {
	var l Link
	err := r.db.WithContext(ctx).Where("short_code = ?", code).First(&l).Error
	return l, notFound(err)
}

func (r *GormRepository) List(ctx context.Context, owner uint64, limit int, cursor uint64) ([]Link, error) {
	query := r.db.WithContext(ctx).Where("created_by = ?", owner)
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	var links []Link
	err := query.Order("id DESC").Limit(limit).Find(&links).Error
	return links, err
}

// UpdateTarget changes the long URL in one statement, returning the row.
func (r *GormRepository) UpdateTarget(ctx context.Context, id, owner uint64, target string) (Link, error) {
	var l Link
	res := r.db.WithContext(ctx).Model(&l).Clauses(clause.Returning{}).
		Where("id = ? AND created_by = ?", id, owner).
		Update("long_url", target)
	return l, affected(res)
}

// Delete removes the link in one statement, returning the deleted row.
func (r *GormRepository) Delete(ctx context.Context, id, owner uint64) (Link, error) {
	var l Link
	res := r.db.WithContext(ctx).Clauses(clause.Returning{}).
		Where("id = ? AND created_by = ?", id, owner).
		Delete(&l)
	return l, affected(res)
}

func notFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

func affected(res *gorm.DB) error {
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
