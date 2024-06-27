package migrations

import (
	"github.com/jinzhu/gorm"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
	"gopkg.in/gormigrate.v1"
)

func MyMigrate() {
	db, _ := models.GetDB()

	m := gormigrate.New(db, gormigrate.DefaultOptions, []*gormigrate.Migration{
		{
			ID: "0001",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0002",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0003",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0004",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0005",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0006",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0007",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0008",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0009",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0010",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0011-Timestamp-Trade_Fields",
			Migrate: func(tx *gorm.DB) error {
				type Scene struct {
					TimestampTradeURL    string `json:"timestampTradeUrl" xbvrbackup:"timestampTradeUrl"`
					TimestampCuepointURL string `json:"timestampCuepointUrl" xbvrbackup:"timestampCuepointUrl"`
					TimestampTradeCount  int    `json:"timestampTradeCnt" xbvrbackup:"timestampTradeCnt"`
				}
				return tx.AutoMigrate(&Scene{}).Error
			},
		},
		{
			ID: "0012",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0013",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0014",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0015",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0016",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0017",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0018",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
		{
			ID: "0019",
			Migrate: func(tx *gorm.DB) error {
				return nil
			},
		},
	})

	if err := m.Migrate(); err != nil {
		common.Log.Fatalf("Could not migrate: %v", err)
	}
	common.Log.Printf("Migration did run successfully")

	db.Close()
}
