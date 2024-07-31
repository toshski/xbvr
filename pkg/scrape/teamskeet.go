package scrape

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

var teamskeetToken string

func Teamskeet(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, scraper string, name string, limitScraping bool) error {
	defer wg.Done()
	scraperID := scraper
	siteID := name
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.teamskeet.com")
	siteCollector := createCollector("www.teamskeet.com")
	siteCollector.MaxDepth = 5
	myCookie := &http.Cookie{
		Name:  "disclaimer_status",
		Value: "agree",
		Path:  "/", // Optional depending on your needs
		// You can set other cookie fields as needed, such as Domain, Expires, etc.
	}
	sceneCollector.SetCookies("https://www.teamskeet.com", []*http.Cookie{myCookie})
	siteCollector.SetCookies("https://www.teamskeet.com", []*http.Cookie{myCookie})

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {

	})
	siteCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		html, err := e.DOM.Html()
		if err != nil {
			log.Printf("Failed to get HTML: %s", err)
			return
		}
		filename := fmt.Sprintf("F:/temp/dump_%s.html", time.Now().Format("150405"))
		file, err := os.Create(filename)
		_, writeErr := file.WriteString(html + "\n")
		if writeErr != nil {
			log.Printf("Failed to write to file: %s", writeErr)
		}

	})

	sceneCollector.OnHTML(`htmlxxxxx`, func(e *colly.HTMLElement) {
	})

	siteCollector.OnHTML(`a.pagination-element__link`, func(e *colly.HTMLElement) {
		if !limitScraping {
			if strings.Contains(e.Text, "Next") {
				pageURL := e.Request.AbsoluteURL(e.Attr("href"))
				siteCollector.Visit(pageURL)
			}
		}
	})

	siteCollector.OnHTML(`div.thumb`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr(`a`, "href"))

		if strings.Contains(sceneURL, "trailers") {
			// If scene exist in database, there's no need to scrape
			if !funk.ContainsString(knownScenes, sceneURL) {

				sceneCollector.Visit(sceneURL)
			}
		}
	})

	if singleSceneURL != "" {
		if teamskeetToken == "" {
			teamskeetToken = refreshToken()
		}
		sceneCollector.Visit(singleSceneURL)
	} else {
		teamskeetToken = "https://ma-store.teamskeet.com/ts_movies/_search?size=40&from=0&sort=publishedDate%3Adesc"
		response := getData()

		if response != nil {
			var sceneList tsSceneListResponse
			json.Unmarshal(response, &sceneList)
			for _, hit := range sceneList.Hits.Hits {
				var scene models.Scene
				scene.GetIfExist("ts-" + hit.ID)
				if scene.ID > 0 {
					continue
				}

				sc := models.ScrapedScene{}
				sc.ScraperID = strings.ReplaceAll(hit.Source.Site.NickName, "-", "")
				sc.SceneType = "2D"
				sc.Studio = "Teamskeet"
				sc.Site = hit.Source.Site.SiteName + " (TS)"
				sc.HomepageURL = "https://app.teamskeet.com/movies/" + hit.ID
				sc.SceneID = "ts-" + hit.ID

				sc.MembersUrl = "https://app.teamskeet.com/movies/" + hit.ID
				sc.Released = time.Unix(hit.Source.PublishedDate, 0).Format("2006-01-02")
				sc.Title = hit.Source.Title
				re := regexp.MustCompile("<[^>]*>")
				sc.Synopsis = re.ReplaceAllString(hit.Source.Description, "")
				sc.Synopsis = strings.TrimSpace(sc.Synopsis)
				sc.Covers = append(sc.Covers, hit.Source.Img)

				for _, tag := range hit.Source.Tags {
					sc.Tags = append(sc.Tags, tag)
				}

				// trailer details
				sc.TrailerType = "teamskeet"
				params := models.TrailerScrape{SceneUrl: "http://192.168.237.11:7877?url=" + "https://api2.teamskeet.com/api/v1/movie/" + hit.ID + "/watch", ContentPath: "fallback"}
				//params := models.TrailerScrape{SceneUrl: "http://tmwproxy:7877?url=" + "https://api2.teamskeet.com/api/v1/movie/"+ hit.ID+ "/watch", RecordPath: "videos", ContentPath: "fallback", QualityPath: "res", EncodingPath: "encoding"}
				strParams, _ := json.Marshal(params)
				sc.TrailerSrc = string(strParams)

				// Cast
				sc.ActorDetails = make(map[string]models.ActorDetails)
				for _, model := range hit.Source.Models {
					if model.Gender == "female" {
						sc.Cast = append(sc.Cast, model.Name)
						// sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
						sc.ActorDetails[model.Name] = models.ActorDetails{Source: sc.ScraperID + " scrape", ImageUrl: model.Img}
					}
				}

				out <- sc

			}
		}
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func addTMWScraper(id string, name string, avatarURL string) {
	registerScraper(id, name+" (TS)", avatarURL, "www.teamskeet.com", func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
		return Teamskeet(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, id, name, limitScraping)
	})
}

func init() {
	addTMWScraper("afterdark", "After Dark", "https://images.psmcdn.net/tsv4/site/logos/large/afterdark.jpg")
	addTMWScraper("analeuro", "Anal Euro", "https://images.psmcdn.net/tsv4/site/logos/large/analeuro.jpg")
	addTMWScraper("badmilfs", "BadMilfs", "https://images.psmcdn.net/tsv4/site/logos/large/badmilfs.jpg")
	addTMWScraper("bffs", "BFFS", "https://images.psmcdn.net/tsv4/site/logos/large/bffs.jpg")
	addTMWScraper("blackstepdad", "Black Step Dad", "https://images.psmcdn.net/tsv4/site/logos/large/blackstepdad.jpg")
	addTMWScraper("blackvalleygirls", "Black Valley Girls", "https://images.psmcdn.net/tsv4/site/logos/large/blackvalleygirls.jpg")
	addTMWScraper("bracefaced", "Brace Faced", "https://images.psmcdn.net/tsv4/site/logos/large/bracefaced.jpg")
	addTMWScraper("breedingmaterial", "Breeding Material", "https://images.psmcdn.net/tsv4/site/logos/large/breedingmaterial.jpg")
	addTMWScraper("cfnmteens", "CFNM Teens", "https://images.psmcdn.net/tsv4/site/logos/large/cfnmteens.jpg")
	addTMWScraper("dadcrush", "Dad Crush", "https://images.psmcdn.net/tsv4/site/logos/large/dadcrush.jpg")
	addTMWScraper("daddypounds", "Daddy Pounds", "https://images.psmcdn.net/tsv4/site/logos/large/daddypounds.jpg")
	addTMWScraper("daughterswap", "Daughter Swap", "https://images.psmcdn.net/tsv4/site/logos/large/daughterswap.jpg")
	addTMWScraper("dyked", "Dyked", "https://images.psmcdn.net/tsv4/site/logos/large/dyked.jpg")
	addTMWScraper("exxxtrasmall", "Exxxtra Small", "https://images.psmcdn.net/tsv4/site/logos/large/exxxtrasmall.jpg")
	addTMWScraper("familystrokes", "Family Strokes", "https://images.psmcdn.net/tsv4/site/logos/large/familystrokes.jpg")
	addTMWScraper("fostertapes", "Foster Tapes", "https://images.psmcdn.net/tsv4/site/logos/large/fostertapes.jpg")
	addTMWScraper("freakyfembots", "Freaky Fembots", "https://images.psmcdn.net/tsv4/site/logos/large/freakyfembots.jpg")
	addTMWScraper("freeusefantasy", "Freeuse Fantasy", "https://images.psmcdn.net/tsv4/site/logos/large/freeusefantasy.jpg")
	addTMWScraper("gingerpatch", "GingerPatch", "https://images.psmcdn.net/tsv4/site/logos/large/gingerpatch.jpg")
	addTMWScraper("glowupz", "Glowupz", "https://images.psmcdn.net/tsv4/site/logos/large/glowupz.jpg")
	addTMWScraper("herfreshmanyear", "Her Freshman Year", "https://images.psmcdn.net/tsv4/site/logos/large/herfreshmanyear.jpg")
	addTMWScraper("hijabhookup", "Hijab Hookup", "https://images.psmcdn.net/tsv4/site/logos/large/hijabhookup.jpg")
	addTMWScraper("hussiepass", "Hussie Pass", "https://images.psmcdn.net/tsv4/site/logos/large/hussiepass.jpg")
	addTMWScraper("imadeporn", "I Made Porn", "https://images.psmcdn.net/tsv4/site/logos/large/imadeporn.jpg")
	addTMWScraper("innocenthigh", "Innocent High", "https://images.psmcdn.net/tsv4/site/logos/large/innocenthigh.jpg")
	addTMWScraper("kissingsis", "Kissing Sis", "https://images.psmcdn.net/tsv4/site/logos/large/kissingsis.jpg")
	addTMWScraper("latinateam", "Latina Team", "https://images.psmcdn.net/tsv4/site/logos/large/latinateam.jpg")
	addTMWScraper("littleasians", "Little Asians", "https://images.psmcdn.net/tsv4/site/logos/large/littleasians.jpg")
	addTMWScraper("lusthd", "Lust HD", "https://images.psmcdn.net/tsv4/site/logos/large/lusthd.jpg")
	addTMWScraper("messyjessy1", "Messy Jessy", "https://images.psmcdn.net/tsv4/site/logos/large/messyjessy1.jpg")
	addTMWScraper("momswap", "Mom Swap", "https://images.psmcdn.net/tsv4/site/logos/large/momswap.jpg")
	addTMWScraper("mormongirlz", "Mormon Girlz", "https://images.psmcdn.net/tsv4/site/logos/large/mormongirlz.jpg")
	addTMWScraper("mybabysittersclub", "My BabySitters Club", "https://images.psmcdn.net/tsv4/site/logos/large/mybabysittersclub.jpg")
	addTMWScraper("notmygrandpa", "Not My Grandpa", "https://images.psmcdn.net/tsv4/site/logos/large/notmygrandpa.jpg")
	addTMWScraper("ourlittlesecret", "Our Little Secret", "https://images.psmcdn.net/tsv4/site/logos/large/ourlittlesecret.jpg")
	addTMWScraper("oyeloca", "Oye Loca", "https://images.psmcdn.net/tsv4/site/logos/large/oyeloca.jpg")
	addTMWScraper("pervdoctor", "PervDoctor", "https://images.psmcdn.net/tsv4/site/logos/large/pervdoctor.jpg")
	addTMWScraper("pervdriver", "PervDriver", "https://images.psmcdn.net/tsv4/site/logos/large/pervdriver.jpg")
	addTMWScraper("pervmom", "PervMom", "https://images.psmcdn.net/tsv4/site/logos/large/pervmom.jpg")
	addTMWScraper("pervnana", "PervNana", "https://images.psmcdn.net/tsv4/site/logos/large/pervnana.jpg")
	addTMWScraper("pervtherapy", "PervTherapy", "https://images.psmcdn.net/tsv4/site/logos/large/pervtherapy.jpg")
	addTMWScraper("petiteteens18", "Petite Teens 18", "https://images.psmcdn.net/tsv4/site/logos/large/petiteteens18.jpg")
	addTMWScraper("povlife", "POV Life", "https://images.psmcdn.net/tsv4/site/logos/large/povlife.jpg")
	addTMWScraper("rubateen", "Rub A Teen", "https://images.psmcdn.net/tsv4/site/logos/large/rubateen.jpg")
	addTMWScraper("selfdesire", "Self Desire", "https://images.psmcdn.net/tsv4/site/logos/large/selfdesire.jpg")
	addTMWScraper("sexandgrades", "Sex and Grades", "https://images.psmcdn.net/tsv4/site/logos/large/sexandgrades.jpg")
	addTMWScraper("sheseducedme", "She Seduced Me", "https://images.psmcdn.net/tsv4/site/logos/large/sheseducedme.jpg")
	addTMWScraper("shesnew", "She's New", "https://images.psmcdn.net/tsv4/site/logos/large/shesnew.jpg")
	addTMWScraper("shoplyfter", "Shoplyfter", "https://images.psmcdn.net/tsv4/site/logos/large/shoplyfter.jpg")
	addTMWScraper("shoplyftermylf", "Shoplyfter Mylf", "https://images.psmcdn.net/tsv4/site/logos/large/shoplyftermylf.jpg")
	addTMWScraper("sislovesme", "Sis Loves Me", "https://images.psmcdn.net/tsv4/site/logos/large/sislovesme.jpg")
	addTMWScraper("sisswap", "Sis Swap", "https://images.psmcdn.net/tsv4/site/logos/large/sisswap.jpg")
	addTMWScraper("solointerviews", "Solo Interviews", "https://images.psmcdn.net/tsv4/site/logos/large/solointerviews.jpg")
	addTMWScraper("spanish18", "Spanish 18", "https://images.psmcdn.net/tsv4/site/logos/large/spanish18.jpg")
	addTMWScraper("stayhomepov", "StayHomePOV", "https://images.psmcdn.net/tsv4/site/logos/large/stayhomepov.jpg")
	addTMWScraper("stepsiblings", "StepSiblings", "https://images.psmcdn.net/tsv4/site/logos/large/stepsiblings.jpg")
	addTMWScraper("submissived", "Submissived", "https://images.psmcdn.net/tsv4/site/logos/large/submissived.jpg")
	addTMWScraper("teamskeetallstars", "TeamSkeet AllStars", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetallstars.jpg")
	addTMWScraper("teamskeetclassics", "TeamSkeet Classics", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetclassics.jpg")
	addTMWScraper("teamskeetextras", "TeamSkeet Extras", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetextras.jpg")
	addTMWScraper("teamskeetfeatures", "TeamSkeet Features", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetfeatures.jpg")
	addTMWScraper("teamskeetlabs", "TeamSkeet Labs", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetlabs.jpg")
	addTMWScraper("teamskeetselects", "TeamSkeet Selects", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetselects.jpg")
	addTMWScraper("teamskeetvip", "TeamSkeet VIP", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetvip.jpg")
	addTMWScraper("teamskeetxadultprime", "TeamSkeet X Adult Prime", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxadultprime.jpg")
	addTMWScraper("teamskeetxamazingfilms", "TeamSkeet x Amazing Films", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxamazingfilms.jpg")
	addTMWScraper("teamskeetxaveryblack", "TeamSkeet X Avery Black", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxaveryblack.jpg")
	addTMWScraper("teamskeetxbaeb", "TeamSkeet X BAEB", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxbaeb.jpg")
	addTMWScraper("teamskeetxbananafever", "TeamSkeet X Banana Fever", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxbananafever.jpg")
	addTMWScraper("teamskeetxbang", "TeamSkeet X Bang", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxbang.jpg")
	addTMWScraper("teamskeetxbjraw", "TeamSkeet X BJ Raw", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxbjraw.jpg")
	addTMWScraper("teamskeetxbrandibraids", "TeamSkeet X Brandi Braids", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxbrandibraids.jpg")
	addTMWScraper("teamskeetxbrasilbimbos", "TeamSkeet X Brasil Bimbos", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxbrasilbimbos.jpg")
	addTMWScraper("teamskeetxbrattyfootgirls", "TeamSkeet X Bratty Foot Girls", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxbrattyfootgirls.jpg")
	addTMWScraper("teamskeetxbritstudioxxx", "TeamSkeet X BritStudioxxx", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxbritstudioxxx.jpg")
	addTMWScraper("teamskeetxcamsoda", "TeamSkeet X CamSoda", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxcamsoda.jpg")
	addTMWScraper("teamskeetxcannonproductions", "TeamSkeet x Cannon Productions", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxcannonproductions.jpg")
	addTMWScraper("teamskeetxclubcastings", "TeamSkeet X Club Castings", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxclubcastings.jpg")
	addTMWScraper("teamskeetxclubsweethearts", "TeamSkeet X Club Sweethearts", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxclubsweethearts.jpg")
	addTMWScraper("teamskeetxcumkitchen", "TeamSkeet X CumKitchen", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxcumkitchen.jpg")
	addTMWScraper("teamskeetxdoctaytay", "TeamSkeet X DocTayTay", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxdoctaytay.jpg")
	addTMWScraper("teamskeetxerotiquetvlive", "TeamSkeet X Erotique TV Live", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxerotiquetvlive.jpg")
	addTMWScraper("teamskeetxevaelfie", "TeamSkeet X Eva Elfie", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxevaelfie.jpg")
	addTMWScraper("teamskeetxevilangel", "TeamSkeet X EvilAngel", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxevilangel.jpg")
	addTMWScraper("teamskeetxfilthykings", "TeamSkeet X Filthy Kings", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxfilthykings.jpg")
	addTMWScraper("teamskeetxfit18", "TeamSkeet X Fit18", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxfit18.jpg")
	addTMWScraper("teamskeetxflorarodgers", "TeamSkeet X Flora Rodgers", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxflorarodgers.jpg")
	addTMWScraper("teamskeetxfuckingawesome", "TeamSkeet X Fucking Awesome", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxfuckingawesome.jpg")
	addTMWScraper("teamskeetxfuckingskinny", "TeamSkeet X Fucking Skinny", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxfuckingskinny.jpg")
	addTMWScraper("teamskeetxgotfilled", "TeamSkeet X GotFilled", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxgotfilled.jpg")
	addTMWScraper("teamskeetxgranddadz", "TeamSkeet X GrandDadz", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxgranddadz.jpg")
	addTMWScraper("teamskeetxharmonyfilms", "TeamSkeet x Harmony Films", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxharmonyfilms.jpg")
	addTMWScraper("teamskeetxherbcollins", "TeamSkeet X Herb Collins", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxherbcollins.jpg")
	addTMWScraper("teamskeetximmaybee", "TeamSkeet X Im May Bee", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetximmaybee.jpg")
	addTMWScraper("teamskeetximpuredesire", "TeamSkeet X Impure Desire", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetximpuredesire.jpg")
	addTMWScraper("teamskeetxjamesdeen", "TeamSkeet X James Deen", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxjamesdeen.jpg")
	addTMWScraper("teamskeetxjasonmoody", "TeamSkeet X Jason Moody", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxjasonmoody.jpg")
	addTMWScraper("teamskeetxjavhub", "TeamSkeet X JavHub", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxjavhub.jpg")
	addTMWScraper("teamskeetxJonathanJordan", "TeamSkeet X Jonathan Jordan", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxJonathanJordan.jpg")
	addTMWScraper("teamskeetxjoybear", "TeamSkeet X Joy Bear", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxjoybear.jpg")
	addTMWScraper("teamskeetxkatekoss", "TeamSkeet X Kate Koss", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxkatekoss.jpg")
	addTMWScraper("teamskeetxkrisskiss", "TeamSkeet X Kriss Kiss", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxkrisskiss.jpg")
	addTMWScraper("teamskeetxlaynalandry", "TeamSkeet X Layna Landry", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxlaynalandry.jpg")
	addTMWScraper("teamskeetxlunaxjames", "TeamSkeet X LunaXJames", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxlunaxjames.jpg")
	addTMWScraper("teamskeetxluxurygirl", "TeamSkeet X Luxury Girl", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxluxurygirl.jpg")
	addTMWScraper("teamskeetxmanko88", "TeamSkeet x Manko88", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxmanko88.jpg")
	addTMWScraper("teamskeetxModelMediaASIA", "Teamskeet X Model Media ASIA", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxModelMediaASIA.jpg")
	addTMWScraper("teamskeetxModelMediaUS", "Teamskeet X Model Media US", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxModelMediaUS.jpg")
	addTMWScraper("teamskeetxmollyredwolf", "TeamSkeet X Molly RedWolf", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxmollyredwolf.jpg")
	addTMWScraper("teamskeetxmrluckypov", "TeamSkeet X Mr Lucky POV", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxmrluckypov.jpg")
	addTMWScraper("teamskeetxog", "TeamSkeet X OG", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxog.jpg")
	addTMWScraper("teamskeetxonly3x", "TeamSkeet X Only 3X", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxonly3x.jpg")
	addTMWScraper("teamskeetxozfellatioqueens", "TeamSkeet X OZ Fellatio Queens", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxozfellatioqueens.jpg")
	addTMWScraper("teamskeetxpovperv", "TeamSkeet X POV Perv", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxpovperv.jpg")
	addTMWScraper("teamskeetxpovgod", "TeamSkeet X POVGod", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxpovgod.jpg")
	addTMWScraper("teamskeetxpurgatoryx", "TeamSkeet X PurgatoryX", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxpurgatoryx.jpg")
	addTMWScraper("teamskeetxrawattack", "TeamSkeet X Raw Attack", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxrawattack.jpg")
	addTMWScraper("teamskeetxreislin", "TeamSkeet X Reislin", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxreislin.jpg")
	addTMWScraper("teamskeetxrileycyriis", "TeamSkeet X Riley Cyriis", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxrileycyriis.jpg")
	addTMWScraper("teamskeetxscreampies", "TeamSkeet X Screampies", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxscreampies.jpg")
	addTMWScraper("teamskeetxsinematica", "TeamSkeet x SINematica", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxsinematica.jpg")
	addTMWScraper("teamskeetxSlutInspection", "TeamSkeet x Slut Inspection", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetXSlutInspection.jpg")
	addTMWScraper("teamskeetxspankmonster", "TeamSkeet X SpankMonster", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxspankmonster.jpg")
	addTMWScraper("teamskeetxsparksentertainment", "TeamSkeet x Sparks Entertainment", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxsparksentertainment.jpg")
	addTMWScraper("teamskeetxspizoo", "TeamSkeet X Spizoo", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxspizoo.jpg")
	addTMWScraper("teamskeetxstellasedona", "TeamSkeet X Stella Sedona", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxstellasedona.jpg")
	addTMWScraper("teamskeetxstephousehold", "TeamSkeet X StepHousehold", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxstephousehold.jpg")
	addTMWScraper("teamskeetxsweetiefox", "TeamSkeet X Sweetie Fox", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxsweetiefox.jpg")
	addTMWScraper("teamskeetxtabbyandnoname", "TeamSkeet X Tabby and No Name", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxtabbyandnoname.jpg")
	addTMWScraper("teamskeetxtenshigao", "TeamSkeet x Tenshigao", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxtenshigao.jpg")
	addTMWScraper("teamskeetxthicc18", "TeamSkeet X Thicc18", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxthicc18.jpg")
	addTMWScraper("teamskeetxtoughlovex", "TeamSkeet X ToughLoveX", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxtoughlovex.jpg")
	addTMWScraper("teamskeetxwilltilexxx", "TeamSkeet X Willtilexxx", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxwilltilexxx.jpg")
	addTMWScraper("teamskeetxxanderporn", "TeamSkeet x Xander Porn", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxxanderporn.jpg")
	addTMWScraper("teamskeetxyesgirlz", "TeamSkeet x YesGirlz", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxyesgirlz.jpg")
	addTMWScraper("teamskeetxyoungbusty", "TeamSkeet X YoungBusty", "https://images.psmcdn.net/tsv4/site/logos/large/teamskeetxyoungbusty.jpg")
	addTMWScraper("teencurves", "Teen Curves", "https://images.psmcdn.net/tsv4/site/logos/large/teencurves.jpg")
	addTMWScraper("teenpies", "Teen Pies", "https://images.psmcdn.net/tsv4/site/logos/large/teenpies.jpg")
	addTMWScraper("teenjoi", "TeenJoi", "https://images.psmcdn.net/tsv4/site/logos/large/teenjoi.jpg")
	addTMWScraper("teensdoporn", "Teens Do Porn", "https://images.psmcdn.net/tsv4/site/logos/large/teensdoporn.jpg")
	addTMWScraper("teensloveanal", "Teens Love Anal", "https://images.psmcdn.net/tsv4/site/logos/large/teensloveanal.jpg")
	addTMWScraper("teensloveblackcocks", "Teens Love Black Cocks", "https://images.psmcdn.net/tsv4/site/logos/large/teensloveblackcocks.jpg")
	addTMWScraper("teenslovemoney", "Teens Love Money", "https://images.psmcdn.net/tsv4/site/logos/large/teenslovemoney.jpg")
	addTMWScraper("teenyblack", "Teeny Black", "https://images.psmcdn.net/tsv4/site/logos/large/teenyblack.jpg")
	addTMWScraper("theloft", "The Loft", "https://images.psmcdn.net/tsv4/site/logos/large/theloft.jpg")
	addTMWScraper("therealworkout", "The Real Workout", "https://images.psmcdn.net/tsv4/site/logos/large/therealworkout.jpg")
	addTMWScraper("thickumz", "Thickumz", "https://images.psmcdn.net/tsv4/site/logos/large/thickumz.jpg")
	addTMWScraper("thisgirlsucks", "This Girl Sucks", "https://images.psmcdn.net/tsv4/site/logos/large/thisgirlsucks.jpg")
	addTMWScraper("tinysis", "Tiny Sis", "https://images.psmcdn.net/tsv4/site/logos/large/tinysis.jpg")
	addTMWScraper("tittyattack", "Titty Attack", "https://images.psmcdn.net/tsv4/site/logos/large/tittyattack.jpg")
}
func refreshToken() string {
	return ""
}
func checkTokenExpiry(jwtToken string) bool {
	// Split the token into its parts
	parts := strings.Split(jwtToken, ".")
	if len(parts) != 3 {
		return true
	}

	// Base64 decode the payload
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return true
	}
	// Unmarshal the JSON payload
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return true
	}

	// Extract and print the expiry time
	if exp, ok := claims["exp"].(float64); ok {
		expiryTime := time.Unix(int64(exp), 0)
		fmt.Printf("Token expires at: %s\n", expiryTime)

		// Check if the token is still valid
		if time.Now().Before(expiryTime) {
			return false
		} else {
			return true
		}
	} else {
		return true // exp not found
	}
}

type teamskeetConfig struct {
	RefreshToken string `json:"refresh_token"`
}

func requestRefresh() string {
	db, _ := models.GetCommonDB()

	tsConfig := teamskeetConfig{}
	var teamskeetConfig models.KV
	teamskeetConfig.Key = "teamskeet"
	db.Find(&teamskeetConfig)
	json.Unmarshal([]byte(teamskeetConfig.Value), &tsConfig)

	url := "https://auth.teamskeet.com/oauth/refresh"
	method := "POST"

	payload := strings.NewReader(`{"refresh_token":"` + tsConfig.RefreshToken + `"}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return ""
	}
	req.Header.Add("Access-Control-Allow-Origin", "*")
	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	var jsonResponse map[string]interface{}
	json.Unmarshal(body, &jsonResponse)
	if accessToken, ok := jsonResponse["access_token"].(string); ok {
		return accessToken
	} else {
		return ""
	}
}
func getData() []byte {
	expired := checkTokenExpiry(teamskeetToken)
	if expired {
		teamskeetToken = requestRefresh()
	}

	if teamskeetToken == "" {
		return nil
	}
	url := "https://ma-store.teamskeet.com/ts_movies/_search?size=40&from=0&sort=publishedDate%3Adesc"
	method := "POST"

	payload := strings.NewReader(`{"query":{"bool":{"must":[{"query_string":{"query":"(isUpcoming:false)"}}]}},"aggs":{"primaryTags":{"terms":{"field":"primary_tags.raw","exclude":"Latest"}},"rareTags":{"rare_terms":{"field":"primary_tags.raw"}}}}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return nil
	}
	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+teamskeetToken)
	req.Header.Add("host", "ma-store.teamskeet.com")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return body
}

// process requests to get the Trailer details
//var (
//    requestQueue  = make(chan string, 100) // Buffer may need adjustment
//    once          sync.Once
//    rateLimit = time.Tick(time.Second / 15) // Rate limit: 14 per second
//)

var (
	requestTimes = make([]time.Time, 0)
	mu           sync.Mutex
	once         sync.Once
)

func canProcessRequest() bool {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()

	// Filter out requests that are older than 10 seconds
	validTimes := make([]time.Time, 0)
	for _, t := range requestTimes {
		if now.Sub(t) <= 10*time.Second {
			validTimes = append(validTimes, t)
		}
	}
	requestTimes = validTimes

	if len(requestTimes) < 15 {
		// If we have processed less than 15 requests in the last 10 seconds, proceed
		requestTimes = append(requestTimes, now)
		return true
	}

	// Otherwise, do not process the request
	return false
}

func EnqueueTSWatchRequest(url string) models.VideoSourceResponse {
	once.Do(func() {
		// Initialize any global state here if necessary
	})

	if canProcessRequest() {
		log.Infof("will process %s", url)
		// Process the request in a new goroutine to not block the caller
		var r models.VideoSourceResponse
		r = ProcessTSWatchRequest(url) // Process and ignore the result for this simplified example
		return r                       // Assuming you need to return a placeholder or actual result
	} else {
		log.Infof("too many requests ignoreing  %s", url)
		// Request is ignored due to rate limit
		return models.VideoSourceResponse{VideoSources: []models.VideoSource{}}
	}
}

func ProcessTSWatchRequest(url string) models.VideoSourceResponse {
	trailers := models.VideoSourceResponse{
		VideoSources: []models.VideoSource{},
	}

	expired := checkTokenExpiry(teamskeetToken)
	if expired {
		teamskeetToken = requestRefresh()
	}

	if teamskeetToken == "" {
		return trailers
	}
	trailerurl := "https://api2.teamskeet.com/api/v1/movie/" + strings.TrimPrefix(url, "ts-") + "/watch"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, trailerurl, nil)

	if err != nil {
		fmt.Println(err)
		return trailers
	}

	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+teamskeetToken)
	req.Header.Add("host", "api2.teamskeet.com")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return trailers
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return trailers
	}
	var sceneList tsSceneListResponse
	json.Unmarshal(body, &sceneList)

	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return trailers
	}
	log.Infof("%s", body)
	var mp4Links []string
	findMP4Links(data, &mp4Links)
	for _, mp4Link := range mp4Links {
		if strings.HasSuffix(mp4Link, ".mp4") {
			trailers.VideoSources = append(trailers.VideoSources, models.VideoSource{URL: mp4Link, Quality: "mp4"})
		} else {
			if strings.HasSuffix(mp4Link, ".m3u8") {
				trailers.VideoSources = append(trailers.VideoSources, models.VideoSource{URL: mp4Link, Quality: "m3u8"})
			} else {
				trailers.VideoSources = append(trailers.VideoSources, models.VideoSource{URL: mp4Link, Quality: "Unknown"})
			}
		}
	}
	return trailers
}

func findMP4Links(data interface{}, links *[]string) {
	switch v := data.(type) {
	case []interface{}:
		for _, u := range v {
			findMP4Links(u, links)
		}
	case map[string]interface{}:
		for _, u := range v {
			if str, ok := u.(string); ok && (strings.HasSuffix(str, ".mp4") || strings.HasSuffix(str, ".m3u8")) {
				*links = append(*links, str)
			} else {
				findMP4Links(u, links)
			}
		}
	}
}

type tsSceneListResponse struct {
	Took         int                     `json:"took"`
	TimedOut     bool                    `json:"timed_out"`
	Shards       tsSceneListShards       `json:"_shards"`
	Hits         tsSceneListHits         `json:"hits"`
	Aggregations tsSceneListAggregations `json:"aggregations"`
}

type tsSceneListShards struct {
	Total      int `json:"total"`
	Successful int `json:"successful"`
	Skipped    int `json:"skipped"`
	Failed     int `json:"failed"`
}

type tsSceneListHits struct {
	Total    tsSceneListTotalHit `json:"total"`
	MaxScore *float64            `json:"max_score"` // Use pointer to handle null
	Hits     []tsSceneListHit    `json:"hits"`
}

type tsSceneListTotalHit struct {
	Value    int    `json:"value"`
	Relation string `json:"relation"`
}

type tsSceneListHit struct {
	Index  string            `json:"_index"`
	Type   string            `json:"_type"`
	ID     string            `json:"_id"`
	Score  *float64          `json:"_score"` // Use pointer to handle null
	Source tsSceneListSource `json:"_source"`
	Sort   []int64           `json:"sort"`
}

type tsSceneListSource struct {
	Title             string             `json:"title"`
	Description       string             `json:"description"`
	Trailer           string             `json:"trailer"`
	Video             string             `json:"video"`
	IsUpcoming        bool               `json:"isUpcoming"`
	Rank              int                `json:"rank"`
	Likes             int                `json:"likes"`
	Dislikes          int                `json:"dislikes"`
	PublishedDate     int64              `json:"publishedDate"`
	PublishedDateRank string             `json:"publishedDateRank"`
	Img               string             `json:"img"`
	VideoTrailer      string             `json:"videoTrailer"`
	Site              tsSceneListSite    `json:"site"`
	SiteLogo          string             `json:"siteLogo"`
	Tier              int                `json:"tier"`
	Type              string             `json:"type"`
	Models            []tsSceneListModel `json:"models"`
	Tags              []string           `json:"tags"`
	PrimaryTags       []string           `json:"primary_tags"`
}

type tsSceneListSite struct {
	SiteID    int    `json:"siteID"`
	ShortName string `json:"shortName"`
	NickName  string `json:"nickName"`
	SiteName  string `json:"siteName"`
}

type tsSceneListModel struct {
	Img    string `json:"img"`
	Gender string `json:"gender"`
	Name   string `json:"name"`
	ID     int    `json:"id"`
	Slug   string `json:"slug"`
}

type tsSceneListAggregations struct {
	PrimaryTags tsSceneListTagBucket `json:"primaryTags"`
	RareTags    tsSceneListTagBucket `json:"rareTags"`
}

type tsSceneListTagBucket struct {
	DocCountErrorUpperBound int                 `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int                 `json:"sum_other_doc_count"`
	Buckets                 []tsSceneListBucket `json:"buckets"`
}

type tsSceneListBucket struct {
	Key      string `json:"key"`
	DocCount int    `json:"doc_count"`
}
