package scrape

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/config"
	"github.com/xbapps/xbvr/pkg/models"
)

func Tmw(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, scraper string, name string, limitScraping bool) error {
	defer wg.Done()
	scraperID := scraper
	siteID := name
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("teenmegaworld.net")
	siteCollector := createCollector("teenmegaworld.net")
	siteCollector.MaxDepth = 5

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "2D"
		if scraper == "tmwvrnet" {
			sc.SceneType = "VR"
		}
		sc.Studio = "TeenMegaWorld"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.SceneID = slugify.Slugify("tmw-" + strings.TrimSuffix(sc.HomepageURL[strings.LastIndex(sc.HomepageURL, "/")+1:], ".html"))
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://tmwvrnet.com/trailers/", "https://"+config.Config.ScraperSettings.TMWVRNet.TmwMembersDomain+"/scenes/", 1)
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://teenmegaworld.net/trailers/", "https://"+config.Config.ScraperSettings.TMWVRNet.TmwMembersDomain+"/scenes/", 1)
		sc.MembersUrl = strings.Replace(sc.MembersUrl, ".html", "_vids.html", 1)

		// Date & Duration
		e.ForEach(`.video-info-data`, func(id int, e *colly.HTMLElement) {
			log.Infof("tmw date %s", e.ChildText(`.video-info-date`))
			tmpDate, _ := goment.New(e.ChildText(`.video-info-date`), "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(strings.Replace(e.ChildText(`.video-info-time`), " min", "", -1)))
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		e.ForEach(`title`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover
		coverurl := e.ChildAttr(`meta[property="og:image"]`, "content")
		if coverurl != "" {
			sc.Covers = append(sc.Covers, coverurl)
		}
		// Gallery
		e.ForEach(`div.photo-list img`, func(id int, e *colly.HTMLElement) {
			galleryURL := e.Request.AbsoluteURL(e.Attr("src"))
			if galleryURL == "" || galleryURL == "https://teenmegaworld.net/assets/vr/public/tour1/images/th5.jpg" {
				return
			}
			srcset := strings.Split(e.Attr("srcset"), ",")
			lastSrc := srcset[len(srcset)-1]
			if lastSrc != "" {
				galleryURL = e.Request.AbsoluteURL(strings.TrimSpace(strings.Split(lastSrc, " ")[0]))
			}
			sc.Gallery = append(sc.Gallery, galleryURL)
		})

		// Synopsis
		e.ForEach(`p.video-description-text`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`div.video-tag-list a`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			if tag != "" {
				sc.Tags = append(sc.Tags, tag)
			}
		})

		// trailer details
		sc.TrailerType = "load_json"
		params := models.TrailerScrape{SceneUrl: "http://192.168.237.11:7879?url=" + sc.HomepageURL, RecordPath: "videos", ContentPath: "src", QualityPath: "res", EncodingPath: "encoding"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`.video-actor-list a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		// Filenames
		// NOTE: no way to guess filename

		out <- sc
	})

	siteCollector.OnHTML(`a.pagination-element__link`, func(e *colly.HTMLElement) {
		if !limitScraping {
			if strings.Contains(e.Text, "Next") {
				pageURL := e.Request.AbsoluteURL(e.Attr("href"))
				WaitBeforeVisit("teenmegaworld.net", siteCollector.Visit, pageURL)
			}
		}
	})

	siteCollector.OnHTML(`div.thumb`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.ChildAttr(`a`, "href"))

		if strings.Contains(sceneURL, "trailers") {
			// If scene exist in database, there's no need to scrape
			if !funk.ContainsString(knownScenes, sceneURL) {
				WaitBeforeVisit("teenmegaworld.net", sceneCollector.Visit, sceneURL)
			}
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		WaitBeforeVisit("teenmegaworld.net", siteCollector.Visit, "https://teenmegaworld.net/sites/"+scraperID)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func addTMWScraper(id string, name string, avatarURL string) {
	registerScraper(id, name+" (TMW)", avatarURL, "teenmegaworld.net", func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
		return Tmw(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, id, name, limitScraping)
	})
}

func init() {
	addTMWScraper("aboutgirlslove", "About Girls Love", "https://cdn.stashdb.org/images/4c/55/4c55d1ac-7282-426b-9d8e-877a9e35579d")
	addTMWScraper("atmovs", "ATMovs", "https://cdn.stashdb.org/images/15/bd/15bd48c1-52f7-46e9-abcc-a3036c0f97f4")
	addTMWScraper("lollyhardcore", "Lolly Hardcore", "https://cdn.stashdb.org/images/97/cc/97cc5d0a-ed99-41bb-acea-18f819638816")
	addTMWScraper("hometeenvids", "Home Teen Vids", "https://cdn.stashdb.org/images/b9/58/b958d524-c731-45d5-8dd6-65109afdc9d9")
	addTMWScraper("hometoyteens", "Home Toy Teens", "https://cdn.stashdb.org/images/15/1d/151dac13-6da3-4a4e-9442-94cd10e47942")
	addTMWScraper("privateteenvideo", "Private Teen Video", "https://cdn.stashdb.org/images/bc/43/bc43a8c4-c853-418c-9a79-da7eda283170")
	addTMWScraper("18firstsex", "18 First Sex", "https://cdn.stashdb.org/images/5a/ba/5aba684c-a1c6-428e-b730-2b4c423a101d")
	addTMWScraper("gag-n-gape", "Gag-n-Gape", "https://cdn.stashdb.org/images/b2/cb/b2cbab68-5d10-443e-a4a9-d82311227db7")
	addTMWScraper("soloteengirls", "Solo Teen Girls", "https://cdn.stashdb.org/images/48/02/48023d70-d20b-40e0-8408-54a1d4aa6623")
	addTMWScraper("old-n-young", "Old-n-Young", "https://cdn.stashdb.org/images/64/08/64086f80-4674-4744-b851-b578a221e638")
	addTMWScraper("watchmefucked", "Watch Me Fucked", "https://cdn.stashdb.org/images/d8/bc/d8bce90a-e6eb-4352-a519-4f6eea87d409")
	addTMWScraper("wow-orgasms", "WOW Orgasms", "https://cdn.stashdb.org/images/7f/9c/7f9c46d7-1fba-467c-b346-e75a325a6ce4")
	addTMWScraper("anal-angels", "Anal Angels", "https://cdn.stashdb.org/images/0a/5c/0a5c6f0c-f929-45d4-beeb-434e89758b9d")
	addTMWScraper("teensexmovs", "Teen Sex Movs", "https://cdn.stashdb.org/images/9e/7e/9e7e964f-20f2-4b14-9845-155135a3123b")
	addTMWScraper("dirty-doctor", "Dirty Doctor", "https://cdn.stashdb.org/images/51/1b/511bcff1-cf29-4bcd-af8c-9b91b540fc59")
	addTMWScraper("trickymasseur", "Tricky Masseur", "https://cdn.stashdb.org/images/23/c3/23c3ba95-7f23-4b3b-bf09-b0f843d103ca")
	addTMWScraper("dirty-coach", "Dirty Coach", "https://cdn.stashdb.org/images/bb/bd/bbbd35b9-965c-491d-8fdc-1175dcc01acc")
	addTMWScraper("fuckstudies", "Fuck Studies", "https://cdn.stashdb.org/images/86/27/86277bd1-0cf2-46c5-be81-63799c45fdb5")
	addTMWScraper("beauty-angels", "Beauty Angels", "https://cdn.stashdb.org/images/b9/21/b92165b9-24b2-4133-8006-bd8e4613b5f3")
	addTMWScraper("firstbgg", "First BGG", "https://cdn.stashdb.org/images/52/2c/522cef80-defe-4cb4-b930-34c57ba4aa23")
	addTMWScraper("teensexmania", "Teen Sex Mania", "https://cdn.stashdb.org/images/0f/f5/0ff5991c-5db9-43e7-8bf5-b7ed3783ffbc")
	addTMWScraper("x-angels", "X-Angels", "https://cdn.stashdb.org/images/33/23/3323818d-d3ca-4b32-81a3-7fc3fb601d34")
	addTMWScraper("creampie-angels", "Creampie Angels", "https://cdn.stashdb.org/images/11/77/11770573-2770-416c-a7ad-9aacf0b7f1cc")
	addTMWScraper("nubilegirlshd", "Nubile Girls HD", "https://cdn.stashdb.org/images/2c/20/2c200f37-259d-4f27-8d35-54157a57d4a9")
	addTMWScraper("nylonsx", "NylonsX", "https://cdn.stashdb.org/images/a6/28/a628d64d-4924-41b5-85de-10cca3956387")
	addTMWScraper("exgfbox", "Ex GF Box", "https://cdn.stashdb.org/images/7f/04/7f048a3d-bef9-4068-84f4-09bbdb6e8386")
	addTMWScraper("teenstarsonly", "Teen Stars Only", "https://cdn.stashdb.org/images/d0/80/d080a92b-55c6-4f52-9f32-0e27195604c9")
	addTMWScraper("squirtingvirgin", "Squirting Virgin", "https://cdn.stashdb.org/images/3e/33/3e33165e-8df9-409b-b6ac-a3b809955164")
	addTMWScraper("teens3some", "Teens3Some", "https://cdn.stashdb.org/images/9c/9a/9c9a9d94-3d6b-4ad0-9be6-958ccf435c73")
	addTMWScraper("rawcouples", "Raw Couples", "https://cdn.stashdb.org/images/30/28/3028bb9e-6e23-4825-8877-f0db8290c103")
	addTMWScraper("tmwpov", "TmwPOV", "https://cdn.stashdb.org/images/9c/c6/9cc63549-3752-4f05-bf44-a99e8ab56c0a")
	addTMWScraper("tmwvrnet", "TmwVRnet", "https://cdn-vr.sexlikereal.com/images/studio_creatives/logotypes/1/26/logo_crop_1623330575.png")
	addTMWScraper("beauty4k", "Beauty 4K", "https://cdn.stashdb.org/images/76/69/76691b88-d8c4-421c-aeac-58da540ca1f9")
	addTMWScraper("ohmyholes", "Oh! My Holes", "https://ohmyholes.com/assets/omh/_default/images/favicon/apple-touch-icon.png")
	addTMWScraper("anal-beauty", "Anal Beauty", "https://cdn.stashdb.org/images/8b/47/8b4752f4-620a-47bd-90bd-243d325b4600")
	addTMWScraper("noboring", "noboring", "https://cdn.stashdb.org/images/8b/47/8b4752f4-620a-47bd-90bd-243d325b4600")
}
