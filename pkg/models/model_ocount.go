package models

import (
	"time"

	"github.com/avast/retry-go/v4"
)

type OCount struct {
	ID uint `gorm:"primary_key" json:"id" xbvrbackup:"-"`

	SceneID  uint      `json:"scene_id" xbvrbackup:"-"`
	Recorded time.Time `json:"recorded" xbvrbackup:"recorded"`
}

func (o *OCount) GetIfExistById(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Where(&OCount{ID: id}).First(o).Error
}

func (o *OCount) GetIfExistBySceneId(scene_id uint) []OCount {
	db, _ := GetDB()
	defer db.Close()
	var ocounts []OCount

	db.Where(&OCount{SceneID: scene_id}).Find(ocounts)
	return ocounts
}

func (o *OCount) Save() {
	db, _ := GetDB()
	defer db.Close()

	var err error = retry.Do(
		func() error {
			err := db.Save(&o).Error
			if err != nil {
				return err
			}
			return nil
		},
	)

	if err != nil {
		log.Fatal("Failed to save ", err)
	}
}

func (o *OCount) Delete() {
	db, _ := GetDB()
	db.Delete(&o)
	db.Close()
}
