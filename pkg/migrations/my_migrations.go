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
			ID: "KM0007 SQL Statements",
			Migrate: func(tx *gorm.DB) error {
				type SqlCmd struct {
					ID uint `gorm:"primary_key" json:"id"  xbvrbackup:"-"`

					SqlGroupID int    `json:"sql_group_id" xbvrbackup:"-"`
					Seq        int    `json:"seq" xbvrbackup:"seq"`
					DbType     string `json:"db_type" xbvrbackup:"db_type"`
					Cmd        string `json:"cmd" gorm:"size:4095" xbvrbackup:"cmd"`
				}
				type SqlEventTrigger struct {
					ID           uint   `gorm:"primary_key" json:"id"  xbvrbackup:"-"`
					SqlGroupID   int    `json:"sql_group_id" xbvrbackup:"-"`
					EventTrigger string `json:"event_trigger" xbvrbackup:"event_trigger"`
				}
				type SqlGroup struct {
					ID uint `gorm:"primary_key" json:"id"  xbvrbackup:"-"`

					Name        string            `json:"name" xbvrbackup:"name"`
					Description string            `json:"description" xbvrbackup:"description"`
					Seq         int               `json:"seq" xbvrbackup:"seq"`
					Triggers    []SqlEventTrigger `json:"triggers" xbvrbackup:"triggers"`
					Commands    []SqlCmd          `json:"commands" xbvrbackup:"commands"`
				}

				err := tx.AutoMigrate(&SqlGroup{}).Error
				if err != nil {
					return err
				}
				err = tx.AutoMigrate(&SqlCmd{}).Error
				if err != nil {
					return err
				}
				return tx.AutoMigrate(&SqlEventTrigger{}).Error
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
			ID: "0011",
			Migrate: func(tx *gorm.DB) error {
				return nil
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
