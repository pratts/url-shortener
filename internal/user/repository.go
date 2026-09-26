package user

import (
	"context"
	"errors"

	"shortener/internal/platform/pgerr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GormRepository stores users in Postgres.
type GormRepository struct {
	db *gorm.DB
}

func NewGormRepository(db *gorm.DB) *GormRepository {
	return &GormRepository{db: db}
}

func (r *GormRepository) Create(ctx context.Context, u *User) error {
	err := r.db.WithContext(ctx).Create(u).Error
	if pgerr.IsUniqueViolation(err, "email") {
		return ErrEmailTaken
	}
	return err
}

func (r *GormRepository) ByID(ctx context.Context, id uint64) (User, error) {
	return r.first(ctx, "id = ?", id)
}

func (r *GormRepository) ByEmail(ctx context.Context, email string) (User, error) {
	return r.first(ctx, "email = ?", email)
}

func (r *GormRepository) first(ctx context.Context, query string, arg interface{}) (User, error) {
	var u User
	err := r.db.WithContext(ctx).Where(query, arg).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrNotFound
	}
	return u, err
}

func (r *GormRepository) Update(ctx context.Context, id uint64, fields map[string]interface{}) (User, error) {
	var u User
	res := r.db.WithContext(ctx).Model(&u).Clauses(clause.Returning{}).
		Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return User{}, res.Error
	}
	if res.RowsAffected == 0 {
		return User{}, ErrNotFound
	}
	return u, nil
}
