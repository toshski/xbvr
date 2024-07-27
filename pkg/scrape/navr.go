package scrape

import (
	"encoding/json"
	"html"
	"strconv"
	"strings"
	"sync"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/robertkrimen/otto"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/models"
)

func NaughtyAmericaVR(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool, siteURL string, siteId string, siteName string, vr bool) error {
	defer wg.Done()
	scraperID := siteId
	siteID := siteName
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.naughtyamerica.com")
	siteCollector := createCollector("www.naughtyamerica.com")

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID

		vr = false
		e.ForEach(`#toogle-container a.flag-bg`, func(id int, e *colly.HTMLElement) {
			// check for vr in none vr sites
			if e.Text == "VR" {
				vr = true
			}
		})

		if vr {
			sc.SceneType = "VR"
		} else {
			sc.SceneType = "2D"
		}
		sc.Studio = "NaughtyAmerica"
		sc.Site = siteID
		sc.Title = ""
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.Replace(sc.HomepageURL, "https://www.naughtyamerica.com/", "https://members.naughtyamerica.com/", 1)

		// Scene ID - get from URL
		tmp := strings.Split(sc.HomepageURL, "-")
		sc.SiteID = tmp[len(tmp)-1]
		sc.SceneID = slugify.Slugify(sc.Site) + "-" + sc.SiteID
		sc.SceneID = "na-" + sc.SiteID

		// Title
		sc.Title = strings.TrimSpace(e.ChildText(`div.scene-info a.site-title`)) + " - " + strings.TrimSpace(e.ChildText(`Title`))

		// Date
		e.ForEach(`div.date-tags span.entry-date`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})

		// Duration
		e.ForEach(`div.date-tags div.duration`, func(id int, e *colly.HTMLElement) {
			r := strings.NewReplacer("|", "", "min", "")
			tmpDuration, err := strconv.Atoi(strings.TrimSpace(r.Replace(e.Text)))
			if err == nil {
				sc.Duration = tmpDuration
			}
		})

		// trailer details
		sc.TrailerType = "url"

		// Filenames & Covers
		// There's a different video element for the four most recent scenes
		// New video element
		e.ForEach(`dl8-video`, func(id int, e *colly.HTMLElement) {
			// images5.naughtycdn.com/cms/nacmscontent/v1/scenes/2cst/nikkijaclynmarco/scene/horizontal/1252x708c.jpg
			base := strings.Split(strings.Replace(e.Attr("poster"), "//", "", -1), "/")
			if len(base) < 7 {
				return
			}
			baseName := base[5] + base[6]
			defaultBaseName := "nam" + base[6]

			filenames := []string{"_180x180_3dh.mp4", "_smartphonevr60.mp4", "_smartphonevr30.mp4", "_vrdesktopsd.mp4", "_vrdesktophd.mp4", "_180_sbs.mp4", "_6kvr264.mp4", "_6kvr265.mp4", "_8kvr265.mp4"}

			for i := range filenames {
				sc.Filenames = append(sc.Filenames, baseName+filenames[i], defaultBaseName+filenames[i])
			}

			base[8] = "horizontal"
			base[9] = "1182x777c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))

			base[8] = "vertical"
			base[9] = "1182x1788c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))
			if vr {
				sc.TrailerSrc = `https://videos.naughtycdn.com/` + base[5] + `/trailers/vr/` + base[5] + base[6] + `/` + base[5] + base[6] + `teaser_vrdesktophd.mp4`
			} else {
				sc.TrailerSrc = `https://videos.naughtycdn.com/` + base[5] + `/trailers/` + base[5] + base[6] + `trailer_720.mp4`
			}
		})

		// Old video element
		e.ForEach(`a.play-trailer img.start-card.desktop-only`, func(id int, e *colly.HTMLElement) {
			// images5.naughtycdn.com/cms/nacmscontent/v1/scenes/2cst/nikkijaclynmarco/scene/horizontal/1252x708c.jpg
			srcset := e.Attr("data-srcset")
			if srcset == "" {
				srcset = e.Attr("srcset")
			}
			base := strings.Split(strings.Replace(srcset, "//", "", -1), "/")
			if len(base) < 7 {
				return
			}
			baseName := base[5] + base[6]
			defaultBaseName := "nam" + base[6]

			filenames := []string{"_180x180_3dh.mp4", "_smartphonevr60.mp4", "_smartphonevr30.mp4", "_vrdesktopsd.mp4", "_vrdesktophd.mp4", "_180_sbs.mp4", "_6kvr264.mp4", "_6kvr265.mp4", "_8kvr265.mp4"}

			for i := range filenames {
				sc.Filenames = append(sc.Filenames, baseName+filenames[i], defaultBaseName+filenames[i])
			}

			base[8] = "horizontal"
			base[9] = "1182x777c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))

			base[8] = "vertical"
			base[9] = "1182x1788c.jpg"
			sc.Covers = append(sc.Covers, "https://"+strings.Join(base, "/"))
			if vr {
				sc.TrailerSrc = `https://videos.naughtycdn.com/` + base[5] + `/trailers/vr/` + base[5] + base[6] + `/` + base[5] + base[6] + `teaser_vrdesktophd.mp4`
			} else {
				sc.TrailerSrc = `https://videos.naughtycdn.com/` + base[5] + `/trailers/` + base[5] + base[6] + `trailer_720.mp4`
			}
		})

		if len(sc.Covers) == 0 {
			e.ForEach(`a.play-trailer img.start-card.live-scene`, func(id int, e *colly.HTMLElement) {
				sc.Covers = append(sc.Covers, "https:"+e.Attr("src"))
			})
		}
		// trailer details for my proxy
		sc.TrailerType = "load_json"
		link := "http://myproxy:7878?url=" + sc.HomepageURL
		if sc.TrailerSrc != "" {
			link = link + "&trailer=" + sc.TrailerSrc
		}

		params := models.TrailerScrape{SceneUrl: link, RecordPath: "videos", ContentPath: "src", QualityPath: "res", EncodingPath: "encoding"}
		strParams, _ := json.Marshal(params)
		sc.TrailerSrc = string(strParams)

		// Gallery
		e.ForEach(`div.contain-scene-images.desktop-only a.thumbnail`, func(id int, e *colly.HTMLElement) {
			if id > 0 {
				sc.Gallery = append(sc.Gallery, strings.Replace(e.Request.AbsoluteURL(e.Attr("href")), "dynamic", "", -1))
			}
		})

		// Synopsis
		e.ForEach(`div.synopsis`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(strings.Replace(e.Text, "Synopsis", "", -1))
		})

		// Tags
		e.ForEach(`div.categories a.cat-tag`, func(id int, e *colly.HTMLElement) {
			sc.Tags = append(sc.Tags, e.Text)
		})

		// Cast (extract from JavaScript)
		e.ForEach(`script`, func(id int, e *colly.HTMLElement) {
			if strings.Contains(e.Text, "femaleStar") {
				vm := otto.New()

				script := e.Text
				script = strings.Replace(script, "window.dataLayer", "dataLayer", -1)
				script = strings.Replace(script, "dataLayer = dataLayer || []", "dataLayer = []", -1)
				script = script + "\nout = []; dataLayer.forEach(function(v) { if (v.femaleStar) { out.push(v.femaleStar); } });"
				vm.Run(script)

				out, _ := vm.Get("out")
				outs, _ := out.ToString()

				sc.Cast = strings.Split(html.UnescapeString(outs), ",")
			}
		})
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`a.scene-title`, func(id int, e *colly.HTMLElement) {
			for _, actor := range sc.Cast {
				if strings.EqualFold(actor, e.Text) {
					sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: strings.SplitN(e.Request.AbsoluteURL(e.Attr("href")), "?", 2)[0]}
				}
			}
		})

		out <- sc
	})

	var visited []string
	siteCollector.OnHTML(`ul[class=pagination] li a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			if !funk.Contains(visited, pageURL) {
				visited = append(visited, pageURL)
				WaitBeforeVisit("www.naughtyamerica.com", siteCollector.Visit, pageURL)
			}
		}
	})

	siteCollector.OnHTML(`div[class=site-list] div[class=scene-item] a.contain-img`, func(e *colly.HTMLElement) {
		sceneURL := strings.Split(e.Request.AbsoluteURL(e.Attr("href")), "?")[0]

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			WaitBeforeVisit("www.naughtyamerica.com", sceneCollector.Visit, sceneURL)
		}
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		WaitBeforeVisit("www.naughtyamerica.com", siteCollector.Visit, siteURL)
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func addNAScraper(id string, name string, avatarURL string, custom bool, siteURL string, vr bool) {
	if avatarURL == "" {
		avatarURL = "https://cdn.stashdb.org/images/bd/30/bd30cc1b-f840-4bba-b573-c2e0ef2e9dfd"
	}

	registerScraper(id, name+" (NA)", avatarURL, siteURL, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
		return NaughtyAmericaVR(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, limitScraping, siteURL, id, name, vr)
	})
}

func init() {
	addNAScraper(slugify.Slugify("My Friend's Hot Mom"), "My Friend's Hot Mom", "https://cdn.stashdb.org/images/30/42/3042ed44-504d-4827-8b31-3ea5f5711e59", false, "https://www.naughtyamerica.com/site/my-friend-s-hot-mom/", false)
	addNAScraper(slugify.Slugify("FANS"), "FANS", "https://cdn.stashdb.org/images/37/90/3790f695-34f7-4ce5-bea6-e502abcb3e21", false, "https://www.naughtyamerica.com/site/fans?&custom1=31546", false)
	addNAScraper(slugify.Slugify("Thundercock"), "Thundercock", "https://cdn.stashdb.org/images/f8/19/f81946c3-4c56-43cc-ad76-6d3f9fde2191", false, "https://www.naughtyamerica.com/site/Thundercock", false)
	addNAScraper(slugify.Slugify("Real Pornstars VR"), "Real Pornstars VR", "https://sm.naughtycdn.com/assets/logos/horizontal_logos/RPVR.webp", false, "https://www.naughtyamerica.com/site/real-pornstars", false)
	addNAScraper(slugify.Slugify("T&A"), "T&A", "https://cdn.stashdb.org/images/1b/26/1b2666ea-47de-4066-91a0-35773ecafed7", false, "https://www.naughtyamerica.com/site/t-a", false)
	addNAScraper(slugify.Slugify("Mom's Money"), "Mom's Money", "https://cdn.stashdb.org/images/20/96/2096eadd-dc91-4fe2-9169-4b5071ba8e1a", false, "https://www.naughtyamerica.com/site/mom-s-money", false)
	addNAScraper(slugify.Slugify("After School"), "After School", "https://cdn.stashdb.org/images/8d/eb/8deba0c9-dbc8-4c58-99d6-516d63e56250", false, "https://www.naughtyamerica.com/site/after-school", false)
	addNAScraper(slugify.Slugify("The Dressing Room"), "The Dressing Room", "https://cdn.stashdb.org/images/cc/ff/ccff8dbd-55d2-4d53-8aaa-837afb74782d", false, "https://www.naughtyamerica.com/site/dressing-room", false)
	addNAScraper(slugify.Slugify("The Gym"), "The Gym", "https://cdn.stashdb.org/images/82/f0/82f075a6-c3ab-4af5-8e43-dda7735c2c78", false, "https://www.naughtyamerica.com/site/gym", false)
	addNAScraper(slugify.Slugify("The Office"), "The Office", "https://cdn.stashdb.org/images/c3/c2/c3c2b9de-5e63-41fc-9693-99096777a99d", false, "https://www.naughtyamerica.com/site/office", false)
	addNAScraper(slugify.Slugify("The Dorm Room"), "The Dorm Room", "https://cdn.stashdb.org/images/d9/3f/d93f3241-5d76-4ba4-a3e9-9540474ce71c", false, "https://www.naughtyamerica.com/site/dorm-room", false)
	addNAScraper(slugify.Slugify("Fuck My Ass"), "Fuck My Ass", "https://cdn.stashdb.org/images/b0/31/b031d365-d829-4a37-8fe0-f8496d6ca33b", false, "https://www.naughtyamerica.com/site/fuck-my-ass", false)
	addNAScraper(slugify.Slugify("My Girlfriend"), "My Girlfriend", "https://cdn.stashdb.org/images/77/a2/77a2e2b6-35a0-4657-995f-639541f577f5", false, "https://www.naughtyamerica.com/site/my-girlfriend", false)
	addNAScraper(slugify.Slugify("PSE Porn Star Experience"), "PSE Porn Star Experience", "https://cdn.stashdb.org/images/a4/2b/a42b290f-3223-4946-aafa-3c4711239184", false, "https://www.naughtyamerica.com/site/pse-porn-star-experience", false)
	addNAScraper(slugify.Slugify("Spring Break"), "Spring Break", "https://cdn.stashdb.org/images/12/c1/12c1025d-a500-4992-97fa-fa2272505e1a", false, "https://www.naughtyamerica.com/site/spring-break", false)
	addNAScraper(slugify.Slugify("Summer Vacation"), "Summer Vacation", "https://cdn.stashdb.org/images/04/1a/041a0a5d-6855-4f5b-8b0c-98ceb4712d82", false, "https://www.naughtyamerica.com/site/summer-vacation", false)
	addNAScraper(slugify.Slugify("Super Sluts"), "Super Sluts", "https://cdn.stashdb.org/images/b6/02/b602e8a0-ab3f-4c46-9a05-7cf516003a02", false, "https://www.naughtyamerica.com/site/super-sluts", false)
	addNAScraper(slugify.Slugify("The Spa"), "The Spa", "https://cdn.stashdb.org/images/eb/c5/ebc5e871-0c34-4066-ac2f-bc1fb01302a9", false, "https://www.naughtyamerica.com/site/spa", false)
	addNAScraper(slugify.Slugify("True Sex Stories"), "True Sex Stories", "https://cdn.stashdb.org/images/e6/cb/e6cbfead-0c4f-4fa4-8757-17ef9fb6f4d9", false, "https://www.naughtyamerica.com/site/true-sex-stories", false)
	addNAScraper(slugify.Slugify("Mrs. Creampie"), "Mrs. Creampie", "https://cdn.stashdb.org/images/1a/bd/1abd66d6-0cd4-44e8-8db8-61947c1cf6d3", false, "https://www.naughtyamerica.com/site/mrs-creampie-1", false)

	addNAScraper(slugify.Slugify("College Sugar Babes"), "College Sugar Babes", "https://cdn.stashdb.org/images/45/4a/454a2e59-7efe-4deb-85db-afdf47e15dfb", false, "https://www.naughtyamerica.com/site/college-sugar-babes", false)
	addNAScraper(slugify.Slugify("Classroom"), "Classroom", "https://cdn.stashdb.org/images/47/0e/470e42e3-e088-4378-860a-8b6372e63d32", false, "https://www.naughtyamerica.com/site/classroom", false)
	addNAScraper(slugify.Slugify("Party Girls"), "Party Girls", "https://cdn.stashdb.org/images/d9/0f/d90f77d2-c442-4696-8ee2-51b6c20df89b", false, "https://www.naughtyamerica.com/site/party-girls", false)
	addNAScraper(slugify.Slugify("My First Sex Teacher"), "My First Sex Teacher", "https://cdn.stashdb.org/images/9a/fb/9afb9f9f-15bd-4a2a-9342-e48595bd5fa8", false, "https://www.naughtyamerica.com/site/my-first-sex-teacher", false)
	addNAScraper(slugify.Slugify("Watch Your Wife"), "Watch Your Wife", "https://cdn.stashdb.org/images/ef/27/ef278d6f-7831-4376-8ce6-dfc0657a75ea", false, "https://www.naughtyamerica.com/site/watch-your-wife", false)
	addNAScraper(slugify.Slugify("Open Family"), "Open Family", "https://cdn.stashdb.org/images/4e/4b/4e4bebaf-01e6-471e-adea-158ac71ab5de", false, "https://www.naughtyamerica.com/site/open-family", false)
	addNAScraper(slugify.Slugify("Naughty Office"), "Naughty Office", "https://cdn.stashdb.org/images/c3/d3/c3d3eb23-35b1-472c-b6b7-34373db7a18e", false, "https://www.naughtyamerica.com/site/naughty-office", false)
	addNAScraper(slugify.Slugify("My Sister's Hot Friend"), "My Sister's Hot Friend", "https://cdn.stashdb.org/images/8c/4f/8c4fe3da-6324-49f8-8b42-4c9861621f70", false, "https://www.naughtyamerica.com/site/my-sister-s-hot-friend", false)
	addNAScraper(slugify.Slugify("Latina Stepmom"), "Latina Stepmom", "https://cdn.stashdb.org/images/65/1a/651a4e30-48ff-4ad6-9fde-ad2585692a94", false, "https://www.naughtyamerica.com/site/latina-step-mom", false)
	addNAScraper(slugify.Slugify("Asian 1 On 1"), "Asian 1 On 1", "https://cdn.stashdb.org/images/2f/34/2f34a10d-098b-4067-8fbe-01dc5763a9e1", false, "https://www.naughtyamerica.com/site/asian-1-on-1", false)
	addNAScraper(slugify.Slugify("Big Cock Hero"), "Big Cock Hero", "https://cdn.stashdb.org/images/09/aa/09aa6523-2bf3-4c03-8db6-a998292069d7", false, "https://www.naughtyamerica.com/site/big-cock-hero", false)
	addNAScraper(slugify.Slugify("Slut Stepsister"), "Slut Stepsister", "https://cdn.stashdb.org/images/1e/ff/1eff0cb3-e9d3-4d5a-ac1e-dd3b8ed0713c", false, "https://www.naughtyamerica.com/site/slut-step-sister", false)
	addNAScraper(slugify.Slugify("Sleazy Stepdad"), "Sleazy Stepdad", "https://cdn.stashdb.org/images/7b/7b/7b7bcb19-e038-41d7-ae37-eaec23801c4c", false, "https://www.naughtyamerica.com/site/sleazy-stepdad", false)
	addNAScraper(slugify.Slugify("Show My BF"), "Show My BF", "https://cdn.stashdb.org/images/74/93/7493fa96-7187-43bc-9153-f7729449026a", false, "https://www.naughtyamerica.com/site/show-my-bf", false)
	addNAScraper(slugify.Slugify("Slut Step Mom"), "Slut Step Mom", "https://cdn.stashdb.org/images/cf/b3/cfb39f42-3a5a-4de8-a337-752d9ed70b89", false, "https://www.naughtyamerica.com/site/slut-step-mom", false)
	addNAScraper(slugify.Slugify("Teens Love Cream"), "Teens Love Cream", "https://cdn.stashdb.org/images/4f/ba/4fba80c0-c4b0-4f09-b19c-cab33685f2df", false, "https://www.naughtyamerica.com/site/teens-love-cream", false)
	addNAScraper(slugify.Slugify("Seduced By A Cougar"), "Seduced By A Cougar", "https://cdn.stashdb.org/images/39/6b/396b7c17-2a6e-44bb-8b51-17da825d0a08", false, "https://www.naughtyamerica.com/site/seduced-by-a-cougar", false)
	addNAScraper(slugify.Slugify("My Daughter's Hot Friend"), "My Daughter's Hot Friend", "https://cdn.stashdb.org/images/35/13/3513dfbf-af68-4044-b287-9164d77048ca", false, "https://www.naughtyamerica.com/site/my-daughter-s-hot-friend", false)
	addNAScraper(slugify.Slugify("LA Sluts"), "LA Sluts", "https://cdn.stashdb.org/images/b1/6c/b16c08cf-ef42-4f97-b046-ed9b8298ad92", false, "https://www.naughtyamerica.com/site/la-sluts", false)
	addNAScraper(slugify.Slugify("My Wife Is My Pornstar"), "My Wife Is My Pornstar", "https://cdn.stashdb.org/images/8d/5b/8d5b480a-46f0-47a2-94f8-47abfe53262b", false, "https://www.naughtyamerica.com/site/my-wife-is-my-pornstar", false)

	addNAScraper(slugify.Slugify("Wives on Vacation"), "Wives on Vacation", "https://cdn.stashdb.org/images/12/60/12606642-9bb4-49d0-9700-63f2265c3ce0", false, "https://www.naughtyamerica.com/site/wives-on-vacation", false)
	addNAScraper(slugify.Slugify("Naughty Weddings"), "Naughty Weddings", "https://cdn.stashdb.org/images/a1/49/a14954fb-b117-4ca0-bf3e-f005193a759b", false, "https://www.naughtyamerica.com/site/naughty-weddings", false)
	addNAScraper(slugify.Slugify("Dirty Wives Club"), "Dirty Wives Club", "https://cdn.stashdb.org/images/9c/29/9c298ab2-4371-4112-beca-3db7de473921", false, "https://www.naughtyamerica.com/site/dirty-wives-club", false)
	addNAScraper(slugify.Slugify("My Dad's Hot Girlfriend"), "My Dad's Hot Girlfriend", "https://cdn.stashdb.org/images/98/d1/98d1649f-f6b2-46bf-8a08-9f6c25efc329", false, "https://www.naughtyamerica.com/site/my-dad-s-hot-girlfriend", false)
	addNAScraper(slugify.Slugify("My Girl Loves Anal"), "My Girl Loves Anal", "https://cdn.stashdb.org/images/e6/97/e6972900-09c5-47ab-9d4f-5238b3b029a1", false, "https://www.naughtyamerica.com/site/my-girl-loves-anal", false)
	addNAScraper(slugify.Slugify("Anal College"), "Anal College", "https://cdn.stashdb.org/images/1e/4e/1e4ee06f-d391-4b6d-929d-958238eae824", false, "https://www.naughtyamerica.com/site/anal-college", false)
	addNAScraper(slugify.Slugify("Lesbian Girl on Girl"), "Lesbian Girl on Girl", "https://cdn.stashdb.org/images/00/cb/00cb2e1c-06bc-4d3d-9ea0-4a7d23512db1", false, "https://www.naughtyamerica.com/site/lesbian-girl-on-girl", false)
	addNAScraper(slugify.Slugify("I Have a Wife"), "I Have a Wife", "https://cdn.stashdb.org/images/5e/44/5e44388d-4fa2-4e01-ae56-ae6955010934", false, "https://www.naughtyamerica.com/site/i-have-a-wife", false)
	addNAScraper(slugify.Slugify("Naughty Bookworms"), "Naughty Bookworms", "https://cdn.stashdb.org/images/d6/1c/d61ce02a-20f9-4a13-bdc5-6bd6e16e9c1f", false, "https://www.naughtyamerica.com/site/naughty-bookworms", false)
	addNAScraper(slugify.Slugify("Housewife 1 on 1"), "Housewife 1 on 1", "https://cdn.stashdb.org/images/13/29/1329a5b3-b617-424f-a604-3bd736fc7021", false, "https://www.naughtyamerica.com/site/housewife-1-on-1", false)
	addNAScraper(slugify.Slugify("My Wife's Hot Friend"), "My Wife's Hot Friend", "https://cdn.stashdb.org/images/98/c9/98c9cb88-392f-435a-b542-2896131ab22b", false, "https://www.naughtyamerica.com/site/my-wife-s-hot-friend", false)
	addNAScraper(slugify.Slugify("Latin Adultery"), "Latin Adultery", "https://cdn.stashdb.org/images/09/1d/091dd871-b0b5-4fcc-94d5-d6a765ee2c9c", false, "https://www.naughtyamerica.com/site/latin-adultery", false)
	addNAScraper(slugify.Slugify("Ass Masterpiece"), "Ass Masterpiece", "https://cdn.stashdb.org/images/e1/10/e110639f-1bf8-4938-b04d-0fec40c115f7", false, "https://www.naughtyamerica.com/site/ass-masterpiece", false)
	addNAScraper(slugify.Slugify("2 Chicks Same Time"), "2 Chicks Same Time", "https://cdn.stashdb.org/images/67/6b/676b918d-b87d-4b3f-80e2-11f7dea03a8e", false, "https://www.naughtyamerica.com/site/2-chicks-same-time", false)
	addNAScraper(slugify.Slugify("My Friend's Hot Girl"), "My Friend's Hot Girl", "https://cdn.stashdb.org/images/1d/19/1d1954b9-c138-40e7-80bf-908459d341c7", false, "https://www.naughtyamerica.com/site/my-friend-s-hot-girl", false)
	addNAScraper(slugify.Slugify("Neighbor Affair"), "Neighbor Affair", "https://cdn.stashdb.org/images/eb/4f/eb4f531e-d19f-4087-b407-7255aaf8c59e", false, "https://www.naughtyamerica.com/site/neighbor-affair", false)
	addNAScraper(slugify.Slugify("My Girlfriend's Busty Friend"), "My Girlfriend's Busty Friend", "https://cdn.stashdb.org/images/6c/04/6c041ae9-f37f-4a50-be3a-b48e9b94529c", false, "https://www.naughtyamerica.com/site/my-girlfriend-s-busty-friend", false)
	addNAScraper(slugify.Slugify("Naughty Athletics"), "Naughty Athletics", "https://cdn.stashdb.org/images/22/73/2273d2f0-bbfb-45cb-994e-220f8509a1a3", false, "https://www.naughtyamerica.com/site/naughty-athletics", false)
	addNAScraper(slugify.Slugify("My Naughty Massage"), "My Naughty Massage", "https://cdn.stashdb.org/images/fe/fc/fefc755a-65ac-4a21-8acb-3901cdb4f177", false, "https://www.naughtyamerica.com/site/my-naughty-massage", false)
	addNAScraper(slugify.Slugify("Fast Times"), "Fast Times", "https://cdn.stashdb.org/images/da/c2/dac22ccb-af80-4ada-94c8-b67bdb19915a", false, "https://www.naughtyamerica.com/site/fast-times", false)

	addNAScraper(slugify.Slugify("The Passenger"), "The Passenger", "https://cdn.stashdb.org/images/61/e0/61e0d42c-4314-4108-8c30-afa5d5cb12f4", false, "https://www.naughtyamerica.com/site/the-passenger", false)
	addNAScraper(slugify.Slugify("MILF Sugar Babes"), "MILF Sugar Babes", "https://cdn.stashdb.org/images/75/7c/757c1219-db39-4b8a-9c14-e35840c1691a", false, "https://www.naughtyamerica.com/site/milf-sugar-babes", false)
	addNAScraper(slugify.Slugify("Perfect Fucking Strangers"), "Perfect Fucking Strangers", "https://cdn.stashdb.org/images/b3/bc/b3bcb975-f4f6-455d-ae36-90989f002d79", false, "https://www.naughtyamerica.com/site/perfect-fucking-strangers", false)
	addNAScraper(slugify.Slugify("American Daydreams"), "American Daydreams", "https://cdn.stashdb.org/images/c4/04/c404a072-affb-4565-b667-49a1ea8d3c81", false, "https://www.naughtyamerica.com/site/american-daydreams", false)
	addNAScraper(slugify.Slugify("Socal Coeds"), "Socal Coeds", "https://cdn.stashdb.org/images/a1/f1/a1f1de77-9b28-4286-81ae-0fdad146cea4", false, "https://www.naughtyamerica.com/site/socal-coeds", false)
	addNAScraper(slugify.Slugify("Naughty Country Girls"), "Naughty Country Girls", "https://cdn.stashdb.org/images/05/15/0515d22e-aaa2-42c6-aa24-312c0fae0584", false, "https://www.naughtyamerica.com/site/naughty-country-girls", false)
	addNAScraper(slugify.Slugify("Diary of a MILF"), "Diary of a MILF", "https://cdn.stashdb.org/images/94/92/94923fd6-2396-4a16-beeb-041cb3f852be", false, "https://www.naughtyamerica.com/site/diary-of-a-milf", false)
	addNAScraper(slugify.Slugify("Naughty Rich Girls"), "Naughty Rich Girls", "https://cdn.stashdb.org/images/85/ee/85ee625f-d90a-422e-973d-589d4a814bfd", false, "https://www.naughtyamerica.com/site/naughty-rich-girls", false)
	addNAScraper(slugify.Slugify("My Naughty Latin Maid"), "My Naughty Latin Maid", "https://cdn.stashdb.org/images/77/9d/779d4bac-db63-40bf-91fe-ef40ae225ee6", false, "https://www.naughtyamerica.com/site/my-naughty-latin-maid", false)
	addNAScraper(slugify.Slugify("Naughty America"), "Naughty America", "https://cdn.stashdb.org/images/70/54/7054e798-8aba-4da1-b043-274ba5f77b3d", false, "https://www.naughtyamerica.com/site/naughty-america", false)
	addNAScraper(slugify.Slugify("Diary of a Nanny"), "Diary of a Nanny", "https://cdn.stashdb.org/images/0e/42/0e42eddd-8896-4c79-8286-c7208d0f0370", false, "https://www.naughtyamerica.com/site/diary-of-a-nanny", false)
	addNAScraper(slugify.Slugify("Naughty Flipside"), "Naughty Flipside", "https://cdn.stashdb.org/images/d9/5b/d95be1d3-62a5-443e-a373-a34bab6baf79", false, "https://www.naughtyamerica.com/site/naughty-flipside", false)
	addNAScraper(slugify.Slugify("Tonight's Fuck"), "Tonight's Fuck", "https://cdn.stashdb.org/images/40/f2/40f23a7f-049f-4033-8baa-0ac3f6e23b86", false, "https://www.naughtyamerica.com/site/tonight-s-fuck", false)

	addNAScraper(slugify.Slugify("Live Party Girl"), "Live Party Girl", "https://cdn.stashdb.org/images/21/ca/21cae785-5101-47bb-8899-6800d0051fa6", false, "https://www.naughtyamerica.com/site/live-party-girl", false)
	addNAScraper(slugify.Slugify("Live Naughty Student"), "Live Naughty Student", "https://cdn.stashdb.org/images/3f/b7/3fb799b6-8bb4-43a4-be15-e13997b9af45", false, "https://www.naughtyamerica.com/site/live-naughty-student", false)
	addNAScraper(slugify.Slugify("Live Naughty Secretary"), "Live Naughty Secretary", "https://cdn.stashdb.org/images/dc/79/dc7999ec-fb67-4d32-a862-0bd1a9f06ded", false, "https://www.naughtyamerica.com/site/live-naughty-secretary", false)
	addNAScraper(slugify.Slugify("Live Gym Cam"), "Live Gym Cam", "https://cdn.stashdb.org/images/03/85/0385caeb-8d84-4760-a6a8-666bbdbf27ea", false, "https://www.naughtyamerica.com/site/live-gym-cam", false)
	addNAScraper(slugify.Slugify("Live Naughty Teacher"), "Live Naughty Teacher", "https://cdn.stashdb.org/images/00/c6/00c69cc8-ff3a-493b-9bcf-0ebaef4822c0", false, "https://www.naughtyamerica.com/site/live-naughty-teacher", false)
	addNAScraper(slugify.Slugify("Live Naughty Milf"), "Live Naughty Milf", "https://cdn.stashdb.org/images/55/19/55197584-753a-4192-961f-9e31678fc1c6", false, "https://www.naughtyamerica.com/site/live-naughty-milf", false)
	addNAScraper(slugify.Slugify("Live Naughty Nurse"), "Live Naughty Nurse", "https://cdn.stashdb.org/images/d3/a2/d3a2a082-08cc-427a-aff5-a5ebaa46e52f", false, "https://www.naughtyamerica.com/site/live-naughty-nurse", false)

	//addNAScraper(slugify.Slugify("Watch Your Mom"), "Watch Your Mom", "https://cdn.stashdb.org/images/39/50/3950401d-11a3-45af-9829-adc7227b5732", false, "https://www.naughtyamerica.com/site/watch-your-mom", false)
	//addNAScraper("naughtyamericavr", "NaughtyAmerica VR", "https://mcdn.vrporn.com/files/20170718100937/naughtyamericavr-vr-porn-studio-vrporn.com-virtual-reality.png", false, "https://www.naughtyamerica.com/vr-porn", true)
	//addNAScraper(slugify.Slugify("Big Cock Bully"), "Big Cock Bully", "https://cdn.stashdb.org/images/a4/45/a445fd2f-ce8d-4c03-b7aa-008c09dd485d", false, "https://www.naughtyamerica.com/site/big-cock-bully", false)
	//addNAScraper(slugify.Slugify("MILF Sugar Babes Classic"), "MILF Sugar Babes Classic", "https://cdn.stashdb.org/images/75/7f/757fc7d2-0716-4791-86eb-9b8cfd05616b", false, "https://www.naughtyamerica.com/site/milf-sugar-babes", false)

}
