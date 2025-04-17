package models

type UrlRedirect struct {
	ID         uint64 `gorm:"primaryKey;autoIncrement"`
	ShortCode  string `gorm:"index"` // <group>/<shortcode>
	ShortUrlID uint64 `gorm:"bigint,not null;index"`
	Agent      string
	IPAddress  string
	CreatedBy  uint64                 `gorm:"bigint,not null;index"`
	Metadata   map[string]interface{} `gorm:"type:jsonb;nullable"` // JSONB
}
