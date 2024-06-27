package api

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/emicklei/go-restful/v3"
	"github.com/xbapps/xbvr/pkg/externalreference"
	"github.com/xbapps/xbvr/pkg/models"
	"github.com/xbapps/xbvr/pkg/scrape"
)

func (i ExternalReference) refreshStashPerformer(req *restful.Request, resp *restful.Response) {
	performerId := req.PathParameter("performerid")
	scrape.RefreshPerformer(performerId)
	resp.WriteHeader(http.StatusOK)
}

func (i ExternalReference) stashSceneApplyRules(req *restful.Request, resp *restful.Response) {
	go externalreference.ApplySceneRules()
}

func (i ExternalReference) matchAkaPerformers(req *restful.Request, resp *restful.Response) {
	go externalreference.MatchAkaPerformers()

}
func (i ExternalReference) stashDbUpdateData(req *restful.Request, resp *restful.Response) {
	go externalreference.UpdateAllPerformerData()

}
func (i ExternalReference) stashRunAll(req *restful.Request, resp *restful.Response) {
	StashdbRunAll()
}
func (i ExternalReference) linkScene2Stashdb(req *restful.Request, resp *restful.Response) {
	sceneId := req.PathParameter("scene-id")
	stashdbId := req.PathParameter("stashdb-id")
	stashdbId = strings.TrimPrefix(stashdbId, "https://stashdb.org/scenes/")
	var scene models.Scene

	db, _ := models.GetDB()
	defer db.Close()

	if strings.Contains(sceneId, "-") {
		scene.GetIfExist(sceneId)
	} else {
		id, _ := strconv.Atoi(req.PathParameter("scene-id"))
		scene.GetIfExistByPK(uint(id))
	}
	if scene.ID == 0 {
		return
	}
	stashScene := scrape.GetStashDbScene(stashdbId)

	var existingRef models.ExternalReference
	existingRef.FindExternalId("stashdb scene", stashdbId)

	jsonData, _ := json.MarshalIndent(stashScene.Data.Scene, "", "  ")

	// chek if we have the performers, may not in the case of loading scenes from the parent studio
	for _, performer := range stashScene.Data.Scene.Performers {
		scrape.UpdatePerformer(performer.Performer)
	}

	var xbrLink []models.ExternalReferenceLink
	xbrLink = append(xbrLink, models.ExternalReferenceLink{InternalTable: "scenes", InternalDbId: scene.ID, InternalNameId: scene.SceneID, ExternalSource: "stashdb scene", ExternalId: stashdbId, MatchType: 5})
	ext := models.ExternalReference{ExternalSource: "stashdb scene", ExternalURL: "https://stashdb.org/scenes/" + stashdbId, ExternalId: stashdbId, ExternalDate: stashScene.Data.Scene.Updated, ExternalData: string(jsonData),
		XbvrLinks: xbrLink}
	ext.AddUpdateWithId()

	// check for actor not yet linked
	for _, actor := range scene.Cast {
		var extreflinks []models.ExternalReferenceLink
		db.Preload("ExternalReference").Where(&models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, ExternalSource: "stashdb performer"}).Find(&extreflinks)
		if len(extreflinks) == 0 {
			stashPerformerId := ""
			for _, stashPerf := range stashScene.Data.Scene.Performers {
				if stashPerf.Performer.Name == actor.Name {
					stashPerformerId = stashPerf.Performer.ID
					continue
				}
				for _, alias := range stashPerf.Performer.Aliases {
					if alias == actor.Name {
						stashPerformerId = stashPerf.Performer.ID
					}
				}
			}
			if stashPerformerId != "" {
				scrape.RefreshPerformer(stashPerformerId)
				var actorRef models.ExternalReference
				actorRef.FindExternalId("stashdb performer", stashPerformerId)
				var performer models.StashPerformer
				json.Unmarshal([]byte(actorRef.ExternalData), &performer)

				xbvrLink := models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, MatchType: 90,
					ExternalReferenceID: actorRef.ID, ExternalSource: actorRef.ExternalSource, ExternalId: actorRef.ExternalId}
				actorRef.XbvrLinks = append(actorRef.XbvrLinks, xbvrLink)
				actorRef.AddUpdateWithId()

				externalreference.UpdateXbvrActor(performer, actor.ID)
			}
		}
	}

	// reread the scene to return updated data
	scene.GetIfExistByPK(scene.ID)
	resp.WriteHeaderAndEntity(http.StatusOK, scene)
}

func (i ExternalReference) searchForStashdb(req *restful.Request, resp *restful.Response) {
	// add title search, exact
	// return aa query string
	// parse a query, return error message
	// allow for an id or url

	query := req.QueryParameter("q")
	if query != "" && query != "undefined" {
		log.Infof(query)
	}

	var warnings []string
	type StashSearchperformerResult struct {
		Name string
		Url  string
	}
	type StashSearchResult struct {
		Url         string
		ImageUrl    string
		Performers  []StashSearchperformerResult
		Title       string
		Duration    string
		Description string
		Weight      int
		Date        string
	}
	type StashSearchResponse struct {
		Status  string
		Results map[string]StashSearchResult
	}
	results := make(map[string]StashSearchResult)

	sceneId := req.PathParameter("scene-id")
	var scene models.Scene

	db, _ := models.GetDB()
	defer db.Close()

	if strings.Contains(sceneId, "-") {
		scene.GetIfExist(sceneId)
	} else {
		id, _ := strconv.Atoi(req.PathParameter("scene-id"))
		scene.GetIfExistByPK(uint(id))
	}
	if scene.ID == 0 {
		var response StashSearchResponse
		response.Results = results
		response.Status = "XBVR Scene not found"
		resp.WriteHeaderAndEntity(http.StatusOK, response)
		return
	}
	stashStudioId := findStashStudioId(scene.ScraperId)
	if stashStudioId == "" {
		var response StashSearchResponse
		response.Results = results
		response.Status = "Cannot find Stashdb Studio"
		resp.WriteHeaderAndEntity(http.StatusOK, response)
		return
	}

	var performers []string
	for _, actor := range scene.Cast {
		var stashlinks []models.ExternalReferenceLink
		db.Preload("ExternalReference").Where(&models.ExternalReferenceLink{InternalTable: "actors", InternalDbId: actor.ID, ExternalSource: "stashdb performer"}).Find(&stashlinks)
		if len(stashlinks) == 0 {
			warnings = append(warnings, actor.Name+" is not linked to Stashdb")
		} else {
			for _, stashPerformer := range stashlinks {
				performers = append(performers, `"`+stashPerformer.ExternalId+`"`)
			}
		}
	}

	// define a function to update the results found
	updateResults := func(stashScenes scrape.QueryScenesResult, weightIncrement int, performers []string) {
		for _, stashscene := range stashScenes.Data.QueryScenes.Scenes {
			// consider adding weight bump for duration and date
			scoreBump := 0
			if stashscene.Date == scene.ReleaseDateText {
				scoreBump += 15
			}
			if stashscene.Title == scene.Title {
				scoreBump += 25
			}
			if stashscene.Duration > 0 && scene.Duration > 0 {
				stashDur := float64((stashscene.Duration / 60) - scene.Duration)
				if math.Abs(stashDur) <= 2 {
					scoreBump += 5 * int(3-math.Abs(stashDur))
				}
			}
			// check duration from video files
			for _, file := range scene.Files {
				if file.Type == "video" {
					diff := file.VideoDuration - float64(stashscene.Duration)
					if math.Abs(diff) <= 2 {
						scoreBump += 5 * int(3-math.Abs(diff))
					}
				}
			}

			// check actor matches using performer stash id
			for _, sp := range stashscene.Performers {
				foundId := false
				for _, xp := range performers {
					if strings.Contains(xp, sp.Performer.ID) {
						foundId = true
					}
				}
				if foundId {
					scoreBump += 5
				} else {
					scoreBump -= 5
				}
			}
			// check actor matches using names and aliases
			for _, actor := range scene.Cast {
				found := false
				for _, sp := range stashscene.Performers {
					if actor.Name == sp.Performer.Name {
						found = true
						continue
					}
					// try aliases
					for _, alias := range sp.Performer.Aliases {
						if alias == actor.Name {
							found = true
							continue
						}
					}
				}
				if found {
					scoreBump += 5
				} else {
					scoreBump -= 5
				}
			}
			if mapEntry, exists := results[stashscene.ID]; exists {
				mapEntry.Weight += weightIncrement + scoreBump
			} else {
				result := StashSearchResult{Url: `https://stashdb.org/scenes/` + stashscene.ID, Weight: weightIncrement + scoreBump, Title: stashscene.Title, Description: stashscene.Details, Date: stashscene.Date}
				if len(stashscene.Images) > 0 {
					result.ImageUrl = stashscene.Images[0].URL
				}
				for _, perf := range stashscene.Performers {
					result.Performers = append(result.Performers, StashSearchperformerResult{Name: perf.Performer.Name, Url: `https://stashdb.org/performers/` + perf.Performer.ID})
				}
				if stashscene.Duration > 0 {
					hours := stashscene.Duration / 3600 // calculate hours
					stashscene.Duration %= 3600         // remaining seconds after hours
					minutes := stashscene.Duration / 60 // calculate minutes
					stashscene.Duration %= 60           // remaining seconds after minutes

					// Format the time string
					result.Duration = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, stashscene.Duration)
				}

				results[stashscene.ID] = result
			}
		}
	}

	var fingerprints []string
	for _, file := range scene.Files {
		if file.Type == "video" {
			file.OsHash = "00000000000000000" + file.OsHash
			fingerprints = append(fingerprints, `"`+file.OsHash[len(file.OsHash)-16:]+`"`)
		}
	}
	if len(fingerprints) > 0 {
		fingerprintList := strings.Join(fingerprints, ",")
		fingerprintQuery := `
		{"input":{
					"studios": {
						"modifier": "EQUALS",
						"value": "` + stashStudioId + `"
					},
					"page": 1,
					"per_page": 25,
					"sort": "UPDATED_AT",
					"fingerprints": {"value": [` +
			fingerprintList +
			`], "modifier":"EQUALS"}
				}
			}`
		log.Infof("%s", fingerprintQuery)
		stashScenes := scrape.GetScenePage(fingerprintQuery)
		log.Infof("%s", stashScenes)
		updateResults(stashScenes, 200, performers)
	}

	performerList := strings.Join(performers, ",")
	performerQuery := `
	{"input":{
				"studios": {
					"modifier": "EQUALS",
					"value": "` + stashStudioId + `"
				},
				"page": 1,
				"per_page": 25,
				"sort": "UPDATED_AT",
				"performers": {"value": [` +
		performerList +
		`], "modifier":"INCLUDES_ALL"}
			}
		}`
	log.Infof("%s", performerQuery)
	stashScenes := scrape.GetScenePage(performerQuery)
	log.Infof("%s", stashScenes)
	updateResults(stashScenes, 100, performers)

	if len(stashScenes.Data.QueryScenes.Scenes) == 0 {
		performerQuery = strings.ReplaceAll(performerQuery, "INCLUDES_ALL", "INCLUDES")
		stashScenes := scrape.GetScenePage(performerQuery)
		updateResults(stashScenes, 10, performers)
	}
	//stashScene := scrape.GetStashDbScene(stashdbId)

	// var existingRef models.ExternalReference
	// existingRef.FindExternalId("stashdb scene", stashdbId)

	// jsonData, _ := json.MarshalIndent(stashScene.Data.Scene, "", "  ")

	// // chek if we have the performers, may not in the case of loading scenes from the parent studio
	// for _, performer := range stashScene.Data.Scene.Performers {
	// 	scrape.UpdatePerformer(performer.Performer)
	// }

	// var xbrLink []models.ExternalReferenceLink
	// xbrLink = append(xbrLink, models.ExternalReferenceLink{InternalTable: "scenes", InternalDbId: scene.ID, InternalNameId: scene.SceneID, ExternalSource: "stashdb scene", ExternalId: stashdbId, MatchType: 5})
	// ext := models.ExternalReference{ExternalSource: "stashdb scene", ExternalURL: "https://stashdb.org/scenes/" + stashdbId, ExternalId: stashdbId, ExternalDate: stashScene.Data.Scene.Updated, ExternalData: string(jsonData),
	// 	XbvrLinks: xbrLink}
	// ext.AddUpdateWithId()

	if len(results) == 0 {
		warnings = append(warnings, "No Stashdb Scenes Found")
	}
	var response StashSearchResponse
	response.Results = results
	response.Status = strings.Join(warnings, ", ")
	resp.WriteHeaderAndEntity(http.StatusOK, response)
}
func StashdbRunAll() {
	go func() {
		scrape.StashDb()

		externalreference.ApplySceneRules()
		externalreference.MatchAkaPerformers()
		externalreference.UpdateAllPerformerData()
		tlog := log.WithField("task", "scrape")
		tlog.Info("Stashdb Refresh Complete")

	}()
}
func findStashStudioId(scraper string) string {
	var site models.Site
	site.GetIfExist(scraper)

	db, _ := models.GetCommonDB()
	var refs []models.ExternalReferenceLink
	db.Preload("ExternalReference").Where(&models.ExternalReferenceLink{InternalTable: "sites", InternalNameId: scraper, ExternalSource: "stashdb studio"}).Find(&refs)

	for _, site := range refs {
		return site.ExternalId
	}

	config := models.BuildActorScraperRules()
	s := config.StashSceneMatching[scraper]
	log.Infof("%s", s)

	sitename := site.Name
	if i := strings.Index(sitename, " ("); i != -1 {
		sitename = sitename[:i]
	}
	log.Infof("%s", sitename)
	studio := scrape.FindStashdbStudio(sitename, "name")
	log.Infof("%s", studio)
	return studio.Data.Studio.ID
}
