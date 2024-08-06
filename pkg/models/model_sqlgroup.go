package models

import "github.com/avast/retry-go/v4"

type SqlGroup struct {
	ID uint `gorm:"primary_key" json:"id"  xbvrbackup:"-"`

	Name        string            `json:"name" xbvrbackup:"name"`
	Description string            `json:"description" xbvrbackup:"description"`
	Seq         int               `json:"seq" xbvrbackup:"seq"`
	Triggers    []SqlEventTrigger `json:"triggers" xbvrbackup:"triggers"`
	Commands    []SqlCmd          `json:"commands" xbvrbackup:"commands"`
}

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

func (a *SqlGroup) GetIfExist(id uint) error {
	db, _ := GetDB()
	defer db.Close()

	return db.Where(&SqlCmd{ID: id}).First(a).Error
}

func (a *SqlGroup) Save() {
	db, _ := GetDB()
	defer db.Close()

	err := retry.Do(
		func() error {
			err := db.Save(&a).Error
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

func AddSqlGroup(name string, desc string, seq int, trigger []SqlEventTrigger, cmds []SqlCmd) {
	sqlcmd := SqlGroup{
		Name:        name,
		Description: desc,
		Seq:         seq,
		Triggers:    trigger,
		Commands:    cmds,
	}
	sqlcmd.Save()
}
