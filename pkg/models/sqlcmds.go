package models

import (
	"sync"

	"github.com/jinzhu/gorm"
)

type Job struct {
	Name string
	Wg   *sync.WaitGroup
}

func WaitSQLTrigger(trigger string) {
	var wg sync.WaitGroup
	RunSQLTrigger(trigger, &wg)
	wg.Wait()
}
func RunSQLTrigger(trigger string, wg *sync.WaitGroup) {
	db, _ := GetDB()
	defer db.Close()
	log.Printf("Processing trigger %s", trigger)
	var sqlGroups []SqlGroup
	db.Table("sql_groups").
		Joins("join sql_event_triggers on sql_event_triggers.sql_group_id=sql_groups.id").
		Where("sql_event_triggers.event_trigger = ?", trigger).
		Order("seq").
		Find(&sqlGroups)

	for _, sqlgrp := range sqlGroups {
		log.Printf("Processing %s", sqlgrp.Name)
		//RunSQLGroup(sqlgrp.Name)
		if wg != nil {
			wg.Add(1)
		}
		JobChan <- Job{sqlgrp.Name, wg}
	}
	log.Printf("Processing trigger %s Done", trigger)
}

func RunSQLGroup(groupName string) {
	db, _ := GetDB()
	defer db.Close()

	dbtype := db.Dialect().GetName()
	log.Printf("Using db type %s", dbtype)
	var sqlGroup SqlGroup
	db.Preload("Commands", func(db *gorm.DB) *gorm.DB { return db.Order("sql_cmds.seq") }).
		Model(&SqlGroup{}).
		Where("name = ?", groupName).
		First(&sqlGroup)

	for _, cmd := range sqlGroup.Commands {
		if cmd.DbType == "" || cmd.DbType == dbtype {
			log.Printf("executing %s: %v, %s", cmd.DbType, cmd.Seq, groupName)
			err := db.Exec(cmd.Cmd).Error
			if err != nil {
				log.Printf("error executing %s %s: %s", cmd.DbType, cmd.Cmd, err.Error())
				tlog := log.WithField("task", "scrape")
				tlog.Infof("Error executing %s", err.Error())
			}
		}
	}
}

var JobChan chan Job

func SetupSQLChannel() {
	// make a channel with a capacity of 1500.
	JobChan = make(chan Job)
	// start the worker
	go worker(JobChan)

}

func worker(jobChan <-chan Job) {
	log.Printf("SQL buffer size %v", len(jobChan))
	for job := range jobChan {
		RunSQLGroup(job.Name)
		if job.Wg != nil {
			job.Wg.Done()
		}
	}
}
