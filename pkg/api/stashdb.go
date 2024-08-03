package api

import (
	"encoding/json"
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
