package scrape

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/mozillazg/go-slugify"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

var apikey string

func Adulttime(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, scraper string, name string, limitScraping bool, studioUrl string, masterSiteId string, additionalInfo interface{}) error {
	defer wg.Done()
	commonDb, _ := models.GetCommonDB()

	additionalInfoJSON, _ := json.Marshal(additionalInfo)
	var additionalScraperInfo AdditionAdulttimeScraperDetail
	json.Unmarshal(additionalInfoJSON, &additionalScraperInfo)

	scraperID := scraper
	siteID := name
	logScrapeStart(scraperID, siteID)

	url := "https://freetour.adulttime.com"
	method := "GET"
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	res, err := client.Do(req)
	if err == nil {
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		strbody := string(body)
		re := regexp.MustCompile(`"apiKey":"([^"]+)"`)
		matches := re.FindStringSubmatch(strbody)
		if len(matches) > 1 {
			apikey = matches[1]
		}
		//log.Info(strbody)
	}

	if singleSceneURL != "" {
		lastIndex := strings.LastIndex(singleSceneURL, "/")
		if lastIndex != -1 {
			adulttimeId := singleSceneURL[lastIndex+1:]

			url := "https://tsmkfa364q-dsn.algolia.net/1/indexes/*/queries?x-algolia-agent=Algolia%20for%20vanilla%20JavaScript%203.27.1%3BJS%20Helper%202.26.0&x-algolia-application-id=TSMKFA364Q&x-algolia-api-key=" + "baaf072fe117f721cc9afeb90905889c"
			method := "POST"
			payload := strings.NewReader(`{
				"requests": [{
					"indexName": "all_scenes",
					"params": "query=&hitsPerPage=20&page=0",
					"facetFilters": "clip_id:` + adulttimeId + `"}]
			}`)

			client := &http.Client{}
			req, err := http.NewRequest(method, url, payload)
			req.Header.Add("content-type", "application/json")
			req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
			req.Header.Add("Referer", "https://www.adulttime.com/")

			res, err := client.Do(req)
			if err == nil {
				defer res.Body.Close()
				body, _ := ioutil.ReadAll(res.Body)

				var data AdulttimeSceneRoot
				json.Unmarshal(body, &data)
				sc := processAdulttimeScrapedScene(data.Results[0].Hits[0], scraperID, "", commonDb, additionalScraperInfo)
				out <- sc

			}
		}
	} else {
		getAdulttimeScenes(scraperID, "", "", limitScraping, masterSiteId, commonDb, additionalScraperInfo, out)
		// scenes := getAdulttimeScenes(scraperID, "", "", limitScraping, masterSiteId, commonDb, additionalScraperInfo, out)
		// for _, scene := range scenes {
		// 	if checkIfWeWantScene(scene.HomepageURL) {
		// 		out <- scene
		// 	}
		// }
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}
func processAdulttimeScrapedScene(adulttimeScene AdulttimeSceneHit, scraperId string, masterSiteId string, commonDb *gorm.DB, additionalScraperInfo AdditionAdulttimeScraperDetail) models.ScrapedScene {
	var scene models.Scene
	clipid := strconv.Itoa(adulttimeScene.ClipID)
	scene.GetIfExist("adulttime-" + clipid)
	// re-get the clip without specifying the sitename, the sitename changes some fields
	response := callAdultimeSceneApi(scraperId, 0, "clip_id:"+clipid)
	if len(response.Results) > 0 && len(response.Results[0].Hits) > 0 {
		adulttimeScene = response.Results[0].Hits[0]
	}

	sc := models.ScrapedScene{}
	sc.MasterSiteId = masterSiteId
	sc.ScraperID = slugify.Slugify(adulttimeScene.Sitename + "-adulttime")
	sc.SceneType = "2D"
	if additionalScraperInfo.Site == "" {
		sc.Site = adulttimeScene.MainChannel.Name + " (Adulttime)"
	} else {
		sc.Site = additionalScraperInfo.Site
	}
	if additionalScraperInfo.Studio == "" {
		if len(adulttimeScene.Channels) > 1 {
			sc.Studio = adulttimeScene.Channels[1].Name
		} else {
			sc.Studio = adulttimeScene.MainChannel.Name
		}
	} else {
		sc.Studio = additionalScraperInfo.Studio
	}
	sc.HomepageURL = "https://members.adulttime.com/en/video/" + adulttimeScene.AvailableOnSite[0] + "/" + adulttimeScene.UrlTitle + "/" + clipid
	sc.SceneID = "adulttime-" + clipid

	sc.MembersUrl = sc.HomepageURL
	sc.Released = adulttimeScene.ReleaseDate
	sc.Title = adulttimeScene.Title
	sc.Synopsis = adulttimeScene.Description

	cover := ""
	resized := ""
	res := 0
	if adulttimeScene.PhotosetID != "" {
		photset := scrapeAdultimePhotosets(adulttimeScene.PhotosetID)
		if len(photset.Results) > 0 && len(photset.Results[0].Hits) > 0 {
			nextInc := 1.0
			nextPos := 0.0
			if len(photset.Results[0].Hits[0].SetPictures) > 15 {
				nextInc = float64(len(photset.Results[0].Hits[0].SetPictures)) / 15
			}
			sc.Gallery = append(sc.Gallery, "https://transform.gammacdn.com/photo_set/"+photset.Results[0].Hits[0].Picture)
			for idx, photo := range photset.Results[0].Hits[0].SetPictures {
				if idx >= int(nextPos) {
					sc.Gallery = append(sc.Gallery, "https://transform.gammacdn.com/photo_set"+photo.Thumb_Path)
					nextPos = nextPos + nextInc
				}
			}
		}
	}
	if len(adulttimeScene.Pictures) > 0 {
		for resolution, url := range adulttimeScene.Pictures {
			if url != "" {
				if cover == "" {
					cover = url
				}
				parts := strings.Split(resolution, "x")
				if len(parts) == 2 {
					size, _ := strconv.Atoi(parts[0])
					if size > res {
						res = size
						cover = url
					}
				}
				if resolution == "resized" {
					resized = url
				}
			}
		}
	}
	if resized != "" {
		sc.Covers = append(sc.Covers, "https://transform.gammacdn.com/movies"+resized)
	} else {
		sc.Covers = append(sc.Covers, "https://transform.gammacdn.com/movies"+cover)
	}

	if adulttimeScene.IsVR == 1 {
		sc.SceneType = "VR"
	}
	for _, tag := range adulttimeScene.Categories {
		sc.Tags = append(sc.Tags, tag.Name)
		if tag.Name == "Interactive Toys" {
			sc.HasScriptDownload = true
		}
	}

	// Cast
	sc.ActorDetails = make(map[string]models.ActorDetails)
	for _, model := range adulttimeScene.Actors {
		modelName := model.Name
		sc.Cast = append(sc.Cast, modelName)
		sc.ActorDetails[modelName] = models.ActorDetails{ProfileUrl: "https://members.adulttime.com/en/pornstar/view/" + model.UrlName + "/" + model.ActorID, Source: sc.ScraperID + " scrape", Gender: model.Gender}
	}

	sc.TrailerType = "scrape_json"
	params := models.TrailerScrape{SceneUrl: sc.MembersUrl, HtmlElement: ".Cms_SceneSignUrlData script", ExtractRegex: "window.defaultStateScene =  (.*);", RecordPath: "*.videos", ContentPath: "url", QualityPath: "format", KVHttpConfig: "adulttime-trailers"}
	strParams, _ := json.Marshal(params)
	sc.TrailerSrc = string(strParams)

	var cuepoints []models.ScrapedCuepoint
	for _, actionTag := range adulttimeScene.ActionTags {
		cuepoints = append(cuepoints, models.ScrapedCuepoint{Name: actionTag.Name, TimeStart: float64(actionTag.Timecode)})
	}
	sc.Cuepoints = cuepoints
	sc.AvailableOnSites = adulttimeScene.AvailableOnSite
	for _, channel := range adulttimeScene.Channels {
		if channel.ID != "" && !funk.Contains(sc.AvailableOnSites, channel.ID) {
			sc.AvailableOnSites = append(sc.AvailableOnSites, channel.ID)
		}
	}
	if adulttimeScene.ClipLength != "" {
		hours, _ := strconv.Atoi(adulttimeScene.ClipLength[0:2])
		mins, _ := strconv.Atoi(adulttimeScene.ClipLength[3:5])
		mins = mins + (hours * 60)
		secs, _ := strconv.Atoi(adulttimeScene.ClipLength[6:8])
		if secs > 29 {
			mins += 1
		}
		sc.Duration = mins

	}
	return sc
}

func init() {
	addAdulttimeScraper("adulttime-single_scene", "Adulttime - Other", "", "", "", nil)
	var scrapers config.ScraperList
	scrapers.Load()
	for _, scraper := range scrapers.XbvrScrapers.AdulttimeScrapers {
		addAdulttimeScraper(scraper.ID, scraper.Name, scraper.AvatarUrl, scraper.URL, scraper.MasterSiteId, scraper.AdditionalInfo)
	}
	for _, scraper := range scrapers.CustomScrapers.AdulttimeScrapers {
		addAdulttimeScraper(scraper.ID, scraper.Name, scraper.AvatarUrl, scraper.URL, scraper.MasterSiteId, scraper.AdditionalInfo)
	}
}
func addAdulttimeScraper(id string, name string, avatarURL string, adulttimeURL string, masterSiteId string, addtionalInfo interface{}) {
	if masterSiteId == "" {
		registerScraper(id, name+" (Adulttime)", avatarURL, "adulttime.com", func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return Adulttime(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, id, name, limitScraping, adulttimeURL, masterSiteId, addtionalInfo)
		})
	} else {
		registerAlternateScraper(id, name+" (Adulttime)", avatarURL, "adulttime.com", masterSiteId, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
			return Adulttime(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, id, name, limitScraping, adulttimeURL, masterSiteId, addtionalInfo)
		})
	}
}

func getAdulttimeScenes(scraperId string, parentId string, tagId string, limitScraping bool, masterSiteId string, commonDb *gorm.DB, additionalScraperInfo AdditionAdulttimeScraperDetail, out chan<- models.ScrapedScene) []models.ScrapedScene {
	page := 0
	var scenes []models.ScrapedScene
	var siteJson gjson.Result
	if strings.HasPrefix(scraperId, "atl") {
		siteJson = gjson.ParseBytes(callAdultimeSiteApi("at"+strings.TrimSuffix(strings.TrimPrefix(scraperId, "atl"), "-adulttime"), scraperId))
	} else {
		siteJson = gjson.ParseBytes(callAdultimeSiteApi(strings.TrimSuffix(scraperId, "-adulttime"), scraperId))
	}
	if siteJson.Get("results.0.nbHits").Int() != 1 {
		log.Warnf("Expectedd only one entry %s", scraperId)
		return scenes
	}
	facetFilter := siteJson.Get("results.0.hits.0.defaultFiltersAlgolia.scenes.hierarchicalFacetsRefinements").String()
	if facetFilter == "" {
		facetFilter = siteJson.Get("results.0.hits.0.defaultFiltersAlgolia.scenes.disjunctiveFacetsRefinements.serie_name").String()
		if facetFilter != "" {
			facetFilter = strings.TrimPrefix(facetFilter, `["`)
			facetFilter = strings.TrimSuffix(facetFilter, `"]`)
			facetFilter = "serie_name:" + facetFilter
		} else {
			facetFilter = siteJson.Get("results.0.hits.0.defaultFiltersAlgolia.scenes.disjunctiveFacetsRefinements.sitename").String()
			if facetFilter != "" {
				facetFilter = strings.TrimPrefix(facetFilter, `["`)
				facetFilter = strings.TrimSuffix(facetFilter, `"]`)
				facetFilter = "sitename:" + facetFilter
			}
		}
	} else {
		facetFilter = strings.TrimPrefix(facetFilter, `{"network.lvl0":["`)
		facetFilter = strings.TrimSuffix(facetFilter, `"]}`)
		if strings.Contains(facetFilter, " > ") {
			facetFilter = `network.lvl1:` + facetFilter
		} else {
			facetFilter = `network.lvl0:` + facetFilter
		}
	}
	if facetFilter == "" {
		facetFilter = "sitename:" + strings.TrimSuffix(scraperId, "-adulttime")
	}
	searchCriteria := ""
	if facetFilter != "" {
		searchCriteria = facetFilter
	} else {
		log.Warnf("no facets defaultinig to sitename %s", scraperId)
	}

	if additionalScraperInfo.SearchCriteria != "" {
		searchCriteria = additionalScraperInfo.SearchCriteria
	}
	data := callAdultimeSceneApi(scraperId, page, searchCriteria)
	if len(data.Results) == 0 {
		log.Warnf("Invalid Query failed  %s", scraperId)
		return scenes
	}
	if len(data.Results[0].Hits) == 0 {
		log.Warnf("no scenes found  %s", scraperId)
	}
	for _, sceneHit := range data.Results[0].Hits {
		if checkIfWeWantScene(strconv.Itoa(sceneHit.ClipID)) {
			sc := processAdulttimeScrapedScene(sceneHit, scraperId, masterSiteId, commonDb, additionalScraperInfo)
			out <- sc
			scenes = append(scenes, sc)
		}
	}

	for limitScraping == false &&
		(page+1) < data.Results[0].NbPages {
		page += 1

		if additionalScraperInfo.SearchCriteria != "" {
			searchCriteria = additionalScraperInfo.SearchCriteria
		}

		data = callAdultimeSceneApi(scraperId, page, searchCriteria)
		for _, sceneHit := range data.Results[0].Hits {
			if checkIfWeWantScene(strconv.Itoa(sceneHit.ClipID)) {
				sc := processAdulttimeScrapedScene(sceneHit, scraperId, masterSiteId, commonDb, additionalScraperInfo)
				out <- sc
				scenes = append(scenes, sc)
			}
		}
	}
	return scenes
}

func checkIfWeWantScene(SceneUrl string) bool {
	parts := strings.Split(SceneUrl, "/")
	var scene models.Scene
	scene.GetIfExist("adulttime-" + parts[len(parts)-1])
	if scene.ID == 0 {
		return true
	}
	if scene.NeedsUpdate {
		return true
	}
	return false
}

func callAdultimeSceneApi(scraperId string, page int, searchCriteria string) AdulttimeSceneRoot {
	url := "https://tsmkfa364q-dsn.algolia.net/1/indexes/*/queries?x-algolia-agent=Algolia%20for%20vanilla%20JavaScript%203.27.1%3BJS%20Helper%202.26.0&x-algolia-application-id=TSMKFA364Q&x-algolia-api-key=" + "baaf072fe117f721cc9afeb90905889c"
	method := "POST"

	payload := strings.NewReader(`{
		"requests": [{
			"indexName": "all_scenes_latest_desc",
			"params": "query=&hitsPerPage=60&page=` + strconv.Itoa(page) + `",
			"facetFilters": ["upcoming:0", "segment:adulttime", "` + searchCriteria + `"]}]
	}`)
	// log.Info(`{
	//  	"requests": [{
	//  		"indexName": "all_scenes_latest_desc",
	//  		"params": "query=&hitsPerPage=60&page=` + strconv.Itoa(page) + `",
	//  		"facetFilters": ["upcoming:0", "segment:adulttime", "` + searchCriteria + `"]}]
	//  }`)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Add("Referer", "https://www.adulttime.com/")

	var data AdulttimeSceneRoot
	res, err := client.Do(req)
	if err == nil {
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		// log.Info("------------------------------------")
		// log.Info(string(body))
		json.Unmarshal(body, &data)
	}
	return data
}

func ScrapeAdulttimeActor(actorId uint, actorURL string, linkType string) {
	actorUrlParts := strings.Split(actorURL, "/")
	adulttimeId := actorUrlParts[len(actorUrlParts)-1]
	apiURL := "https://tsmkfa364q-dsn.algolia.net/1/indexes/*/queries?x-algolia-agent=Algolia%20for%20vanilla%20JavaScript%203.27.1%3BJS%20Helper%202.26.0&x-algolia-application-id=TSMKFA364Q&x-algolia-api-key=" + "baaf072fe117f721cc9afeb90905889c"
	method := "POST"
	payload := strings.NewReader(`{
		"requests": [
			{
				"indexName": "all_actors",
				"params": "query=&page=0&facetFilters=%5B%22actor_id%3A` + adulttimeId + `%22%5D"
			} ] } 
	`)

	client := &http.Client{}
	req, err := http.NewRequest(method, apiURL, payload)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Add("Referer", "https://www.adulttime.com/")

	var data AdulttimeActorRoot
	res, err := client.Do(req)
	if err == nil {
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)

		json.Unmarshal(body, &data)
	}

	if len(data.Results[0].Hits) == 0 {
		return
	}

	var actor models.Actor
	actor.GetIfExistByPK(uint(actorId))

	profilePic := ""
	largestRes := 0
	if len(data.Results[0].Hits[0].Pictures) > 0 {
		for resolution, url := range data.Results[0].Hits[0].Pictures {
			if url != "" {
				parts := strings.Split(resolution, "x")
				if profilePic == "" {
					profilePic = url + "?width=" + parts[0] + "&height=" + parts[1]
				}
				if len(parts) == 2 {
					size, _ := strconv.Atoi(parts[0])
					if size > largestRes {
						largestRes = size
						profilePic = url + "?width=" + parts[0] + "&height=" + parts[1]
					}
				}
			}
		}
	}
	actor.AddToImageArray("https://transform.gammacdn.com/actors" + profilePic)
	if actor.ImageUrl == "" {
		actor.ImageUrl = "https://transform.gammacdn.com/actors" + profilePic
	}
	actor.Save()

	var extref models.ExternalReference
	var extreflink models.ExternalReferenceLink

	commonDb, _ := models.GetCommonDB()
	commonDb.Preload("ExternalReference").
		Where(&models.ExternalReferenceLink{ExternalSource: linkType, InternalDbId: uint(actorId)}).
		First(&extreflink)
	extref = extreflink.ExternalReference

	if extref.ID == 0 {
		actor.Save()
		dataJson, _ := json.Marshal(data)

		extrefLink := []models.ExternalReferenceLink{{InternalTable: "actors", InternalDbId: actor.ID, InternalNameId: actor.Name, ExternalSource: linkType, ExternalId: actorURL}}
		extref = models.ExternalReference{ID: extref.ID, XbvrLinks: extrefLink, ExternalSource: linkType, ExternalId: actorURL, ExternalURL: actorURL, ExternalDate: time.Now(), ExternalData: string(dataJson)}
		extref.AddUpdateWithId()
	} else {
		extref.ExternalDate = time.Now()
		extref.AddUpdateWithId()
	}
}

func scrapeAdultimePhotosets(photosetId string) AdulttimePhotosetRoot {
	url := "https://tsmkfa364q-dsn.algolia.net/1/indexes/*/queries?x-algolia-agent=Algolia%20for%20vanilla%20JavaScript%203.27.1%3BJS%20Helper%202.26.0&x-algolia-application-id=TSMKFA364Q&x-algolia-api-key=" + apikey
	method := "POST"
	payload := strings.NewReader(`{ "requests": [ { "indexName": "all_photosets", "params": "query=&page=0&facetFilters=%5B%22set_id%3A` + photosetId + `%22%5D" } ] }`)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Add("Referer", "https://www.adulttime.com/")

	var data AdulttimePhotosetRoot
	res, err := client.Do(req)
	if err == nil {
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		json.Unmarshal(body, &data)
	}
	return data
}

func callAdultimeSiteApi(site string, scraperId string) []byte {
	url := "https://tsmkfa364q-dsn.algolia.net/1/indexes/*/queries?x-algolia-agent=Algolia%20for%20vanilla%20JavaScript%203.27.1%3BJS%20Helper%202.26.0&x-algolia-application-id=TSMKFA364Q&x-algolia-api-key=" + "baaf072fe117f721cc9afeb90905889c"
	method := "POST"

	payload := strings.NewReader(`{
    "requests": [
        {
            "indexName": "all_channels",
            "params": "query=&hitsPerPage=10&page=0",
            "facetFilters": "segment:adulttime, slug:` + site + `"
	    } 
	] 
	}`)
	// log.Info(`{
	// "requests": [
	//     {
	//         "indexName": "all_channels",
	//         "params": "query=&hitsPerPage=10&page=0",
	//         "facetFilters": "segment:adulttime, slug:` + site + `"
	// } ] }`)
	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0")
	req.Header.Add("Referer", "https://www.adulttime.com/")

	res, err := client.Do(req)
	if err == nil {
		defer res.Body.Close()
		body, _ := ioutil.ReadAll(res.Body)
		var s models.Site
		j := string(body)
		searchparams := gjson.Get(j, "results.0.hits.0.defaultFiltersAlgolia.scenes").String()
		s.GetIfExist(scraperId)
		if s.ID == "" {
			log.Warnf("Could not update site data %s", site)
		}
		s.Adulttime = searchparams
		s.Save()
		return body
	}
	return nil
}

type AdulttimeSceneRoot struct {
	Results []AdulttimeSceneResult `json:"results"`
}

type AdulttimeSceneResult struct {
	Hits                []AdulttimeSceneHit          `json:"hits"`
	NbHits              int                          `json:"nbHits"`
	Page                int                          `json:"page"`
	NbPages             int                          `json:"nbPages"`
	HitsPerPage         int                          `json:"hitsPerPage"`
	ExhaustiveNbHits    bool                         `json:"exhaustiveNbHits"`
	ExhaustiveTypo      bool                         `json:"exhaustiveTypo"`
	Exhaustive          AdulttimeExhaustive          `json:"exhaustive"`
	Query               string                       `json:"query"`
	Params              string                       `json:"params"`
	Index               string                       `json:"index"`
	ProcessingTimeMS    int                          `json:"processingTimeMS"`
	ProcessingTimingsMS AdulttimeProcessingTimingsMS `json:"processingTimingsMS"`
}

type AdulttimeSceneHit struct {
	Clip4KSize        string                   `json:"clip_4k_size"`
	SubtitleID        interface{}              `json:"subtitle_id"` // Use interface{} for null values
	ClipID            int                      `json:"clip_id"`
	Title             string                   `json:"title"`
	Description       string                   `json:"description"`
	ClipType          string                   `json:"clip_type"`
	ClipLength        string                   `json:"clip_length"`
	ClipPath          string                   `json:"clip_path"`
	SourceClipID      int                      `json:"source_clip_id"`
	Length            int                      `json:"length"`
	ReleaseDate       string                   `json:"release_date"`
	Upcoming          int                      `json:"upcoming"`
	MovieID           int                      `json:"movie_id"`
	MovieTitle        string                   `json:"movie_title"`
	MovieDesc         string                   `json:"movie_desc"`
	MovieDateCreated  string                   `json:"movie_date_created"`
	Compilation       string                   `json:"compilation"`
	SiteID            int                      `json:"site_id"`
	Sitename          string                   `json:"sitename"`
	SitenamePretty    string                   `json:"sitename_pretty"`
	Segment           string                   `json:"segment"`
	SerieID           int                      `json:"serie_id"`
	SerieName         string                   `json:"serie_name"`
	StudioID          int                      `json:"studio_id"`
	StudioName        string                   `json:"studio_name"`
	CategoryIDs       string                   `json:"category_ids"`
	Directors         []interface{}            `json:"directors"` // Use interface{} for empty arrays
	NetworkName       string                   `json:"network_name"`
	NetworkID         string                   `json:"network_id"`
	Original          string                   `json:"original"`
	SegmentSiteID     string                   `json:"segment_site_id"`
	Scrubbers         AdulttimeScrubbers       `json:"scrubbers"`
	Trailers          map[string]string        `json:"trailers"`
	UrlTitle          string                   `json:"url_title"`
	UrlMovieTitle     string                   `json:"url_movie_title"`
	LengthRange15Min  string                   `json:"length_range_15min"`
	PhotosetID        string                   `json:"photoset_id"`
	PhotosetName      string                   `json:"photoset_name"`
	PhotosetUrlName   string                   `json:"photoset_url_name"`
	Network           AdulttimeNetwork         `json:"network"`
	Date              int64                    `json:"date"`
	Actors            []AdulttimeActor         `json:"actors"`
	FemaleActors      []AdulttimeActor         `json:"female_actors"`
	Categories        []AdulttimeCategory      `json:"categories"`
	MasterCategories  []interface{}            `json:"master_categories"` // Use interface{} for empty arrays
	AwardWinning      []interface{}            `json:"award_winning"`     // Use interface{} for empty arrays
	Male              int                      `json:"male"`
	Female            int                      `json:"female"`
	Shemale           int                      `json:"shemale"`
	PicturesQualifier []string                 `json:"pictures_qualifier"`
	Pictures          map[string]string        `json:"pictures"`
	DownloadFileSizes map[string]int           `json:"download_file_sizes"`
	DownloadSizes     []string                 `json:"download_sizes"`
	AvailableOnSite   []string                 `json:"availableOnSite"`
	ContentTags       []string                 `json:"content_tags"`
	Lesbian           int                      `json:"lesbian"`
	Bisex             int                      `json:"bisex"`
	Trans             int                      `json:"trans"`
	HasSubtitle       int                      `json:"hasSubtitle"`
	HasPpu            int                      `json:"hasPpu"`
	Channels          []AdulttimeChannel       `json:"channels"`
	MainChannel       AdulttimeChannel         `json:"mainChannel"`
	RatingRank        interface{}              `json:"rating_rank"` // Use interface{} for null values
	RatingsUp         int                      `json:"ratings_up"`
	RatingsDown       int                      `json:"ratings_down"`
	Plays365Days      int                      `json:"plays_365days"`
	Plays30Days       int                      `json:"plays_30days"`
	Plays7Days        int                      `json:"plays_7days"`
	Plays24Hours      int                      `json:"plays_24hours"`
	EngagementScore   float64                  `json:"engagement_score"`
	NetworkPriority   int                      `json:"network_priority"`
	ActionTags        []AdulttimeActionTag     `json:"action_tags"`
	Views             int                      `json:"views"`
	SingleSiteViews   int                      `json:"single_site_views"`
	VRFormat          interface{}              `json:"vr_format"` // Use interface{} for null or missing values
	IsVR              int                      `json:"isVR"`
	VideoFormats      []AdulttimeVideoFormat   `json:"video_formats"`
	ObjectID          string                   `json:"objectID"`
	HighlightResult   AdulttimeHighlightResult `json:"_highlightResult"`
}

type AdulttimeExhaustive struct {
	NbHits bool `json:"nbHits"`
	Typo   bool `json:"typo"`
}

type AdulttimeProcessingTimingsMS struct {
	Request AdulttimeRequest `json:"_request"`
}

type AdulttimeRequest struct {
	RoundTrip int `json:"roundTrip"`
}

type AdulttimeScrubbers struct {
	Full    AdulttimeScrubber `json:"full"`
	Trailer []interface{}     `json:"trailer"` // Use interface{} for potentially empty arrays
}

type AdulttimeScrubber struct {
	URL         string `json:"url"`
	ThumbWidth  string `json:"thumbWidth"`
	ThumbHeight string `json:"thumbHeight"`
}

type AdulttimeNetwork struct {
	Lvl0 string `json:"lvl0"`
	Lvl1 string `json:"lvl1"`
}

type AdulttimeActor struct {
	ActorID string `json:"actor_id"`
	Name    string `json:"name"`
	Gender  string `json:"gender"`
	UrlName string `json:"url_name"`
}

type AdulttimeCategory struct {
	CategoryID string `json:"category_id"`
	Name       string `json:"name"`
	UrlName    string `json:"url_name"`
}

type AdulttimeChannel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type AdulttimeActionTag struct {
	Name     string `json:"name"`
	Timecode int    `json:"timecode"`
}

type AdulttimeVideoFormat struct {
	Codec      string `json:"codec"`
	Format     string `json:"format"`
	Size       int    `json:"size"`
	Slug       string `json:"slug"`
	TrailerURL string `json:"trailer_url"`
}

type AdulttimeHighlightResult struct {
	Title AdulttimeHighlight `json:"title"`
}

type AdulttimeHighlight struct {
	Value            string   `json:"value"`
	MatchLevel       string   `json:"matchLevel"`
	FullyHighlighted bool     `json:"fullyHighlighted"`
	MatchedWords     []string `json:"matchedWords"`
}

type AdulttimeActorRoot struct {
	Results []AdulttimeActorResult `json:"results"`
}

type AdulttimeActorResult struct {
	Hits                []AdulttimeActorHit          `json:"hits"`
	NbHits              int                          `json:"nbHits"`
	Page                int                          `json:"page"`
	NbPages             int                          `json:"nbPages"`
	HitsPerPage         int                          `json:"hitsPerPage"`
	ExhaustiveNbHits    bool                         `json:"exhaustiveNbHits"`
	ExhaustiveTypo      bool                         `json:"exhaustiveTypo"`
	Exhaustive          AdulttimeExhaustive          `json:"exhaustive"`
	Query               string                       `json:"query"`
	Params              string                       `json:"params"`
	Index               string                       `json:"index"`
	ProcessingTimeMS    int                          `json:"processingTimeMS"`
	ProcessingTimingsMS AdulttimeProcessingTimingsMS `json:"processingTimingsMS"`
}

type AdulttimeActorHit struct {
	ActorId     int                 `json:"actor_id"`
	Name        string              `json:"name"`
	Gender      string              `json:"gender"`
	Description string              `json:"description"`
	URLName     string              `json:"url_name"`
	Categories  []AdulttimeCategory `json:"categories"`
	Pictures    map[string]string   `json:"pictures"`
}

type AdulttimePhotosetRoot struct {
	Results []AdulttimePhotosetResult `json:"results"`
}

type AdulttimePhotosetResult struct {
	Hits                []AdulttimePhotosetHit       `json:"hits"`
	NbHits              int                          `json:"nbHits"`
	Page                int                          `json:"page"`
	NbPages             int                          `json:"nbPages"`
	HitsPerPage         int                          `json:"hitsPerPage"`
	ExhaustiveNbHits    bool                         `json:"exhaustiveNbHits"`
	ExhaustiveTypo      bool                         `json:"exhaustiveTypo"`
	Exhaustive          AdulttimeExhaustive          `json:"exhaustive"`
	Query               string                       `json:"query"`
	Params              string                       `json:"params"`
	Index               string                       `json:"index"`
	ProcessingTimeMS    int                          `json:"processingTimeMS"`
	ProcessingTimingsMS AdulttimeProcessingTimingsMS `json:"processingTimingsMS"`
}

type AdulttimePhotosetHit struct {
	SetId       int                        `json:"set_id"`
	Picture     string                     `json:"picture"`
	SetPictures []AdulttimePhotosetPicture `json:"set_pictures"`
}
type AdulttimePhotosetPicture struct {
	Thumb_Path string `json:"thumb_path"`
}

type AdditionAdulttimeScraperDetail struct {
	SearchCriteria string `json:"search_criteria"`
	Studio         string `json:"studio"`
	Site           string `json:"site"`
}
