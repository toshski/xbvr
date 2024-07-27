package scrape

import (
	"encoding/json"
	"net/url"
	"path"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"

	"github.com/xbapps/xbvr/pkg/models"
)

func NubilesSite(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, scraperID string, siteID string, URL string, singeScrapeAdditionalInfo string, limitScraping bool, sceneIdPrefix string) error {
	defer wg.Done()
	logScrapeStart(scraperID, siteID)
	sceneListPageCnt := 0

	parsedURL, _ := url.Parse(URL)
	sceneCollector := createCollector(parsedURL.Hostname())
	siteCollector := createCollector(parsedURL.Hostname())

	db, _ := models.GetDB()
	defer db.Close()

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "2D"
		sc.Studio = "Nubiles"
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "://"+parsedURL.Hostname(), "://members."+sceneIdPrefix+".com", 1)

		// Site ID
		sc.Site = siteID

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "/")
		sc.SceneID = slugify.Slugify(sceneIdPrefix) + "-" + tmp[5]

		// Title
		e.ForEach(`div.content-pane-title h2`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		// Cover URLs
		e.ForEach(`meta[property='og:image']`, func(id int, e *colly.HTMLElement) {
			sc.Covers = append(sc.Covers, e.Request.AbsoluteURL(e.Attr("content")))
		})

		// Gallery - removed links expiry
		//		e.ForEach(`div.content-pane-pick img`, func(id int, e *colly.HTMLElement) {
		//			sc.Gallery = append(sc.Gallery, e.Request.AbsoluteURL(e.Attr("src")))
		//		})

		// Synopsis
		e.ForEach(`div.content-pane-description`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		e.ForEach(`div.categories a`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, strings.TrimSpace(e.Text))
		})

		// trailer details
		sc.TrailerType = "load_json"
		params := models.TrailerScrape{SceneUrl: "http://nubilesproxy:7879?url=" + sc.HomepageURL, RecordPath: "videos", ContentPath: "src", QualityPath: "res", EncodingPath: "encoding"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// filenames, uses the trailer filenames as a bsee
		e.ForEach(`meta[property='og:video']`, func(id int, vid_e *colly.HTMLElement) {
			basename := path.Base(vid_e.Attr("content"))
			indexOfTrailer := strings.Index(basename, "_trailer")
			if indexOfTrailer == -1 {
				indexOfTrailer = strings.LastIndex(basename, "_")
			}
			if indexOfTrailer != -1 {
				basename = basename[:indexOfTrailer]
				if basename != "" {
					var filenames []string
					e.ForEach(`.edge-download-item .edge-download-item-dimensions`, func(id int, dl_ele *colly.HTMLElement) {
						dimensions := strings.TrimSpace(dl_ele.Text)
						indexOfX := strings.Index(dimensions, "x")
						if indexOfX != -1 {
							filenames = append(filenames, basename+"_"+dimensions[:indexOfX]+".mp4")
						}
					})
					sc.Filenames = filenames
				}
			}
		})

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`a.content-pane-performer`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		// Date
		e.ForEach(`div.content-pane-title span.date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(strings.Replace(e.Text, "Uploaded: ", "", -1), "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration

		out <- sc
	})

	if !limitScraping {
		siteCollector.OnHTML(`li.page-item a[aria-label='Next Page']`, func(e *colly.HTMLElement) {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			sceneListPageCnt++
			if sceneListPageCnt < 3000 {
				WaitBeforeVisit("nubiles-porn.com", siteCollector.Visit, pageURL)
			}
		})
	}

	siteCollector.OnHTML(`div.caption-header a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			WaitBeforeVisit("nubiles-porn.com", sceneCollector.Visit, sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		WaitBeforeVisit("nubiles-porn.com", siteCollector.Visit, URL)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}

	logScrapeFinished(scraperID, siteID)

	return nil
}

func addNubilesPornScraper(id string, name string, company string, avatarURL string, custom bool, siteURL string, sitesuffix string, firstpage string) {
	if avatarURL == "" {
		avatarURL = "https://static.nubiles-porn.com/assets/lightTheme/images/icons/NP_Icon23_01.svg"
	}

	suffixname := name
	if sitesuffix != "" {
		suffixname = name + " (" + sitesuffix + ")"
	}
	if firstpage == "" {
		firstpage = "https://" + siteURL + "/video/gallery"
	}
	registerScraper(id, suffixname, avatarURL, siteURL, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
		return NubilesSite(wg, updateSite, knownScenes, out, singleSceneURL, id, name, firstpage, singeScrapeAdditionalInfo, limitScraping, "nubiles-porn")
	})
}

func init() {
	//	registerScraper("nubiles-porn", "Nubiles-Porn", "https://static.nubiles-porn.com/assets/lightTheme/images/icons/NP_Icon23_01.svg", "nubiles-porn.com", NubilesPornVR)
	//	registerScraper("nubiles-net", "Nubiles-Net", "https://static.nubiles.net/assets/legacyTheme/images/icons/NubilesNet.svg", "nubiles.net", NubilesNet)

	addNubilesPornScraper("nubiles-porn", "Nubiles Porn", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/icons/NP_Icon23_01.svg", false, "nubiles-porn.com", "Nubiles-Porn", "https://nubiles-porn.com/video/gallery/website/7")
	addNubilesPornScraper("nubiles-net", "Nubiles.net", "Nubiles", "https://static.nubiles.net/assets/legacyTheme/images/icons/NubilesNet.svg", false, "nubiles.net", "Nubiles-Porn", "")
	addNubilesPornScraper("badteenspunished", "Bad Teens Punished", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/btp_logo.png", false, "badteenspunished.com", "Nubiles-Porn", "")
	addNubilesPornScraper("bountyhunterporn", "Bounty Hunter Porn", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/bhx_logo.png", false, "bountyhunterporn.com", "Nubiles-Porn", "")
	addNubilesPornScraper("caughtmycoach", "Caught My Coach", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/logos/cmc_logo_stacked.svg", false, "caughtmycoach.com", "Nubiles-Porn", "")
	addNubilesPornScraper("cheatingsis", "Cheating Sis", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/ch_logo.svg", false, "cheatingsis.com", "Nubiles-Porn", "")
	addNubilesPornScraper("cumswappingsis", "Cum Swapping Sis", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/icons/css_icon.svg", false, "cumswappingsis.com", "Nubiles-Porn", "")
	addNubilesPornScraper("driverxxx", "Driver XXX", "Nubiles", "https://cdn.stashdb.org/images/65/ac/65ac067f-25d2-4b62-9910-308e32a20b44", false, "driverxxx.com", "Nubiles-Porn", "")
	addNubilesPornScraper("badteenspunished", "Bad Teens Punished", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/btp_logo.png", false, "badteenspunished.com", "Nubiles-Porn", "")
	addNubilesPornScraper("daddyslilangel", "Daddy's Lil Angel", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/dla_logo.png", false, "daddyslilangel.com", "Nubiles-Porn", "")
	addNubilesPornScraper("teacherfucksteens", "Teacher Fucks Teens", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/tft_logo.png", false, "teacherfucksteens.com", "Nubiles-Porn", "")
	addNubilesPornScraper("nubiles-casting", "Nubiles Casting", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/nc_logo.png", false, "nubiles-casting.com", "Nubiles-Porn", "")
	addNubilesPornScraper("petiteballerinasfucked", "Petite Ballerinas Fucked", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/pbf_logo.png", false, "petiteballerinasfucked.com", "Nubiles-Porn", "")
	addNubilesPornScraper("stepsiblingscaught", "Step Siblings Caught", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/ssc_logo.png", false, "stepsiblingscaught.com", "Nubiles-Porn", "")
	addNubilesPornScraper("princesscum", "Princess Cum", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/pc_logo.png", false, "princesscum.com", "Nubiles-Porn", "")
	addNubilesPornScraper("petitehdporn", "Petite HD Porn", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/phdp_logo.png", false, "petitehdporn.com", "Nubiles-Porn", "")
	addNubilesPornScraper("nubilesunscripted", "Nubiles Unscripted", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/nu_logo.png", false, "nubilesunscripted.com", "Nubiles-Porn", "")
	addNubilesPornScraper("myfamilypies", "My Family Pies", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/mfp_logo.png", false, "myfamilypies.com", "Nubiles-Porn", "")
	addNubilesPornScraper("nubileset", "NubilesET", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/logos/ne_white_text.png", false, "nubileset.com", "Nubiles-Porn", "")
	addNubilesPornScraper("momsteachsex", "Moms Teach Sex", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/mts_logo.png", false, "momsteachsex.com", "Nubiles-Porn", "")
	addNubilesPornScraper("familyswap", "Family Swap", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/icons/fsx.svg", false, "familyswap.xxx", "Nubiles-Porn", "")
	addNubilesPornScraper("detentiongirls", "Detention Girls", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/logos/dg_logo_white_text.png", false, "detentiongirls.com", "Nubiles-Porn", "")
	addNubilesPornScraper("youngermommy", "Younger Mommy", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/icons/yl_icon.svg", false, "youngermommy.com", "Nubiles-Porn", "")
	addNubilesPornScraper("smashed", "Smashed", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/banner_logos/sx_logo_white.svg", false, "smashed.xxx", "Nubiles-Porn", "")
	addNubilesPornScraper("lilsis", "Lil Sis", "Nubiles", "https://static.nubiles-porn.com/assets/lightTheme/images/logos/ls_logo_stacked.svg", false, "lilsis.com", "Nubiles-Porn", "")
}
