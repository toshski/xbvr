package scrape

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

func NewSensations(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
	defer wg.Done()
	scraperID := "newsensations"
	siteID := "NewSensations"
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("newsensations.com")
	siteCollector := createCollector("newsensations.com")
	fileCollector := createCollector("newsensations.com", "nsnetworkmembers.newsensations.com")

	storedCookies := getCookies()
	c := siteCollector.Cookies(domain)
	cookie1 := http.Cookie{Name: `pcar%5fTWVtYmVycyBBcmVh`, Value: storedCookies.PCar, Domain: "newsensations.com", Path: "/"}
	c = append(c, &cookie1)
	cookie2 := http.Cookie{Name: `psso%5fTWVtYmVycyBBcmVh`, Value: storedCookies.PSSO, Domain: "newsensations.com", Path: "/"}
	c = append(c, &cookie2)
	cookie3 := http.Cookie{Name: `username`, Value: storedCookies.Username, Domain: "newsensations.com", Path: "/", Expires: time.Now().AddDate(1, 0, 0)}
	c = append(c, &cookie3)
	siteCollector.SetCookies("https://"+"newsensations.com", c)
	sceneCollector.SetCookies("https://"+"newsensations.com", c)
	fileCollector.SetCookies("https://"+"newsensations.com", c)

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "2D"
		sc.Studio = "New Sensations"
		sc.Site = siteID
		sc.HomepageURL = e.Request.URL.String()
		queryParams := e.Request.URL.Query()
		sc.SiteID = queryParams.Get("id")

		// Title
		//		abort := false
		e.ForEach(`div.indRight h2`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
			//if strings.Contains(strings.ToLower(sc.Title), "compilations") || strings.Contains(strings.ToLower(sc.Title), "interviews") || strings.Contains(strings.ToLower(sc.Title), "bts") {
			//	abort = true
			//			}
		})

		//		if !abort {
		// Cover URLs
		e.ForEach(`img[id="default_poster"]`, func(id int, e *colly.HTMLElement) {
			coverUrl := e.Attr("src")
			parsedURL, err := url.Parse(coverUrl)
			if err != nil {
				log.Fatal("Error parsing URL:", err)
			}
			path := parsedURL.Path
			// Construct the local file path (adjust as needed)
			localFilePath := filepath.Join(common.MyFilesDir+`\NewSensations`, path)

			// Check if the file exists
			if !fileExists(localFilePath) {
				fileCollector.Visit(coverUrl)
			}

			sc.Covers = append(sc.Covers, "/myfiles/NewSensations"+path)
			// sc.Gallery = append(sc.Gallery, "/myfiles/NewSensations"+path)
		})

		// Gallery
		//e.ForEach(`div.video-detail__gallery a.image-container`, func(id int, e *colly.HTMLElement) {
		//	sc.Gallery = append(sc.Gallery, e.Attr("href"))
		//})

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`span.update_models a`, func(id int, e *colly.HTMLElement) {
			if strings.TrimSpace(e.Text) != "" {
				actorUrl := e.Request.AbsoluteURL(e.Attr("href"))
				//parsedActorURL, _ := url.Parse(actorUrl)
				//actorName := strings.TrimSpace(e.Text + " (" + parsedActorURL.Query().Get(("id")) + ")")
				actorName := strings.TrimSpace(e.Text)
				sc.Cast = append(sc.Cast, actorName)
				sc.ActorDetails[actorName] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: actorUrl}
			}
		})

		// Tags
		e.ForEach(`div.textLink a[href*='category.php']`, func(id int, e *colly.HTMLElement) {
			tag := strings.TrimSpace(e.Text)
			if tag != "" {
				sc.Tags = append(sc.Tags, strings.ToLower(tag))
			}
		})

		// Synposis
		e.ForEach(`div.description	`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(e.Text), "Description:"))
		})

		// Release date / Duration
		e.ForEach(`div.datePhotos`, func(id int, e *colly.HTMLElement) {
			dateRegex := regexp.MustCompile(`\d{2}/\d{2}/\d{4}`)
			lengthRegex := regexp.MustCompile(`\d+.Minutes`)

			// Find date
			dateMatch := dateRegex.FindString(strings.TrimSpace(e.Text))
			if dateMatch != "" {
				tmpDate, _ := goment.New(strings.TrimSpace(dateMatch), "MM/DD/YYYY")
				sc.Released = tmpDate.Format("YYYY-MM-DD")
			}

			// Find video length
			lengthMatch := lengthRegex.FindString(strings.TrimSpace(e.Text))
			if lengthMatch != "" {
				// Extract the number of minutes from the matched string
				minutesRegex := regexp.MustCompile(`\d+`)
				minutesMatch := minutesRegex.FindString(lengthMatch)
				sc.Duration, _ = strconv.Atoi(strings.TrimSpace(minutesMatch))
			}
		})

		e.ForEach(`div.stremVideo[id="download_select"] a`, func(id int, e *colly.HTMLElement) {
			videourl, err := url.Parse(e.Attr("href"))
			if err == nil {
				if videourl.Host == "nsnetworkmembers.newsensations.com" {
					sc.Filenames = append(sc.Filenames, filepath.Base(videourl.Path))
					sc.Filenames = append(sc.Filenames, sc.SiteID+" - "+filepath.Base(videourl.Path))
				}
			}
		})

		sc.TrailerType = "newsensations"
		sc.TrailerSrc = sc.HomepageURL

		if sc.SiteID != "" {
			sc.SceneID = fmt.Sprintf("newsensations-%v", sc.SiteID)

			// save only if we got a SceneID
			//if sc.SiteID == "9821" {
			out <- sc
			//}
		}
		//		}
	})

	fileCollector.OnResponse(func(r *colly.Response) {
		log.Println("Visited", r.Request.URL)

		// Create the file
		fileName := common.MyFilesDir + `\NewSensations` + r.Request.URL.Path
		// Check if the directory exists
		dir := filepath.Dir(fileName)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			// Create the directory
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				log.Fatal("Error creating directory: ", err)
				return
			}
		}
		file, err := os.Create(fileName)
		if err != nil {
			log.Fatal("Could not create file:", err)
		}
		defer file.Close()

		// Write the response body to file
		_, err = file.Write(r.Body)
		if err != nil {
			log.Fatal("Could not write to file:", err)
		}

		log.Println("File saved:", fileName)
	})

	siteCollector.OnHTML(`div.pagination a`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			siteCollector.Visit(pageURL)
		}
	})

	siteCollector.OnHTML(`div.videoBlock div.captions h4 a`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			sceneCollector.Visit(sceneURL)
		}
	})

	siteCollector.OnHTML(`html`, func(e *colly.HTMLElement) {

	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		siteCollector.Visit("https://newsensations.com/members/category.php?id=5&s=d")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil
}

func init() {
	registerScraper("newsensations", "New Sensations", "https://newsensations.com/tour_ns/favicon.ico", "newsensations.com", NewSensations)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

type newsensationsConfig struct {
	Username string `json:"username"`
	PCar     string `json:"pcar"`
	PSSO     string `json:"psso"`
}

func getCookies() newsensationsConfig {
	db, _ := models.GetCommonDB()

	nsConfig := newsensationsConfig{}
	var newsensationsConfig models.KV
	newsensationsConfig.Key = "newsensations"
	db.Find(&newsensationsConfig)
	json.Unmarshal([]byte(newsensationsConfig.Value), &nsConfig)
	return nsConfig
}

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
func EnqueueNSWatchRequest(url string) models.VideoSourceResponse {
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

	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Println(err)
		return trailers
	}

	req.Header.Add("Accept", "application/json, text/plain, */*")
	req.Header.Add("host", "newsensations.com")

	storedCookies := getCookies()
	req.AddCookie(&http.Cookie{Name: `username`, Value: storedCookies.Username, Domain: "newsensations.com", Path: "/"})
	req.AddCookie(&http.Cookie{Name: `pcar%5fTWVtYmVycyBBcmVh`, Value: storedCookies.PCar, Domain: "newsensations.com", Path: "/"})
	req.AddCookie(&http.Cookie{Name: `psso%5fTWVtYmVycyBBcmVh`, Value: storedCookies.PSSO, Domain: "newsensations.com", Path: "/"})

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return trailers
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return trailers
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return trailers
	}

	doc.Find(`div.stremVideo[id="download_select"] a`).Each(func(i int, s *goquery.Selection) {
		videourl, _ := s.Attr("href")
		if strings.Contains(videourl, "https://nsnetworkmembers.newsensations.com") {
			quality := strings.TrimSpace(s.Text())
			re := regexp.MustCompile(`MP4 - (\d+.)`)
			matches := re.FindStringSubmatch(quality)
			if len(matches) >= 2 {
				quality = matches[1]
			}
			trailers.VideoSources = append(trailers.VideoSources, models.VideoSource{URL: strings.TrimSuffix(videourl, "&d=1"), Quality: quality})
		}
	})

	return trailers
}
