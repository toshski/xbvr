package migrations

import (
	"encoding/json"
	"os"
	"path/filepath"

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
			ID: "0012-Ocount",
			Migrate: func(tx *gorm.DB) error {
				return tx.AutoMigrate(&models.OCount{}).Error
			},
		},
		{
			ID: "0013-actor_scraper_config",
			Migrate: func(tx *gorm.DB) error {
				type actorScraperConfig struct {
					StashSceneMatching         map[string]models.StashSiteConfig
					GenericActorScrapingConfig map[string]models.GenericScraperRuleSet
				}

				fName := filepath.Join(common.AppDir, "actor_scraper_custom_config.json")
				if _, err := os.Stat(fName); os.IsNotExist(err) {
					return nil
				}
				var newCustomScrapeRules models.ActorScraperConfig
				b, err := os.ReadFile(fName)
				if err != nil {
					return err
				}
				e := json.Unmarshal(b, &newCustomScrapeRules)
				if e == nil {
					// if we can read the file with the new model, it has already been converted
					return nil
				}

				var oldCustomScrapeRules actorScraperConfig
				e = json.Unmarshal(b, &oldCustomScrapeRules)
				if e != nil {
					// can't read the old layout either ?
					return err
				}

				newCustomScrapeRules = models.ActorScraperConfig{}
				newCustomScrapeRules.GenericActorScrapingConfig = oldCustomScrapeRules.GenericActorScrapingConfig
				newCustomScrapeRules.StashSceneMatching = map[string][]models.StashSiteConfig{}
				for key, scraper := range oldCustomScrapeRules.StashSceneMatching {
					common.Log.Infof("%s %s", key, scraper)
					common.Log.Infof("%s", oldCustomScrapeRules.StashSceneMatching[key])
					newScraperCofig := oldCustomScrapeRules.StashSceneMatching[key]
					common.Log.Infof("%s", newScraperCofig)
					newCustomScrapeRules.StashSceneMatching[key] = []models.StashSiteConfig{}
					newCustomScrapeRules.StashSceneMatching[key] = append(newCustomScrapeRules.StashSceneMatching[key], newScraperCofig)
				}
				out, _ := json.MarshalIndent(newCustomScrapeRules, "", "  ")
				e = os.WriteFile(fName, out, 0644)
				return e

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