package scrape

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gocolly/colly/v2"
	"github.com/mozillazg/go-slugify"
	"github.com/nleeper/goment"
	"github.com/thoas/go-funk"
	"github.com/tidwall/gjson"
	"github.com/xbapps/xbvr/pkg/common"
	"github.com/xbapps/xbvr/pkg/models"
)

var vixenToken string

func Vixen(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, scraper string, name string, limitScraping bool) error {
	defer wg.Done()
	scraperID := scraper
	siteID := name
	logScrapeStart(scraperID, siteID)

	sceneCollector := createCollector("www.blacked.com", "www.blackedraw.com", "www.deeper.com", "www.tushy.com", "www.tushyraw.com", "www.vixen.com")
	siteCollector := createCollector("www.blacked.com", "www.blackedraw.com", "www.deeper.com", "www.tushy.com", "www.tushyraw.com", "www.vixen.com")
	fileCollector := createCollector("cdn.blacked.com", "cdn.blackedraw.com", "cdn.deeper.com", "cdn.tushy.com", "cdn.tushyraw.com", "cdn.vixen.com")
	siteCollector.MaxDepth = 5

	sceneCollector.OnHTML(`html`, func(e *colly.HTMLElement) {
		sc := models.ScrapedScene{}
		sc.ScraperID = scraperID
		sc.SceneType = "2D"
		sc.Studio = "Vixen"
		sc.Site = siteID
		sc.HomepageURL = strings.Split(e.Request.URL.String(), "?")[0]
		sc.MembersUrl = strings.ReplaceAll(sc.HomepageURL, "//www.", "//members.")

		var sceneId string
		// Cover
		e.ForEach(`[data-test-component="VideoCoverWrapper"] picture img`, func(id int, e *colly.HTMLElement) {
			coverurl := e.Attr(`src`)
			if coverurl != "" {
				sc.Covers = append(sc.Covers, coverurl)
				idRegEx := regexp.MustCompile(`videoimages\/(\d+)\/`)
				sceneId = idRegEx.FindStringSubmatch(coverurl)[1]
				sc.SceneID = slugify.Slugify("vixen-" + sceneId)
			}
		})

		// Title
		e.ForEach(`[data-test-component="VideoTitle"]`, func(id int, e *colly.HTMLElement) {
			sc.Title = strings.TrimSpace(e.Text)
		})

		e.ForEach(`script[type="application/json"]`, func(id int, e *colly.HTMLElement) {
			JsonMetadata := strings.TrimSpace(e.Text)

			// skip non video Metadata
			if gjson.Get(JsonMetadata, "props").Exists() {

				if gjson.Get(JsonMetadata, "props.pageProps.carouselImages").Exists() {
					images := gjson.Get(JsonMetadata, "props.pageProps.carouselImages")
					for _, img := range images.Array() {
						src := img.Get("src").String()
						galleryURL := e.Request.AbsoluteURL(src)

						parsedURL, err := url.Parse(galleryURL)
						if err != nil {
							log.Fatal("Error parsing URL:", err)
						}
						path := parsedURL.Path
						// Construct the local file path (adjust as needed)
						localFilePath := filepath.Join(common.MyFilesDir+`\`+scraperID, path)

						// Check if the file exists
						if !fileExists(localFilePath) {
							fileCollector.Visit(galleryURL)
						}
						sc.Gallery = append(sc.Gallery, "/myfiles/"+scraperID+path)
					}
				}
			}
		})

		// Date
		e.ForEach(`[data-test-component="ReleaseDateFormatted"]`, func(id int, e *colly.HTMLElement) {
			tmpDate, _ := goment.New(e.Text, "MMMM DD, YYYY")
			sc.Released = tmpDate.Format("YYYY-MM-DD")
		})
		// Duration
		e.ForEach(`[data-test-component="RunLengthFormatted"]`, func(id int, e *colly.HTMLElement) {
			parts := strings.Split(e.Text, ":")
			dur := 0
			secs := 0
			switch len(parts) {
			case 2: // mm:ss
				dur, _ = strconv.Atoi(parts[0])
				secs, _ = strconv.Atoi(parts[1])
				if secs > 30 {
					dur++
				}
			case 3: // hh:mm:ss
				hrs, _ := strconv.Atoi(parts[0])
				mins, _ := strconv.Atoi(parts[0])
				secs, _ = strconv.Atoi(parts[2])
				dur = hrs*60 + mins
				if secs > 30 {
					dur++
				}
			}
			sc.Duration = dur
		})

		// Synopsis
		e.ForEach(`[data-test-component="VideoDescription"]`, func(id int, e *colly.HTMLElement) {
			sc.Synopsis = strings.TrimSpace(e.Text)
		})

		// Tags
		results := GetCateegoriesAndChapters(sceneId, scraperID)
		for _, category := range results.categories {
			sc.Tags = append(sc.Tags, category)
		}
		var cuepoints []models.ScrapedCuepoint
		for _, cuepoint := range results.chapters {
			cuepoints = append(cuepoints, models.ScrapedCuepoint{Name: cuepoint.title, TimeStart: float64(cuepoint.seconds)})
		}
		sc.Cuepoints = cuepoints

		// trailer details
		sc.TrailerType = scraperID
		sc.TrailerSrc = sceneId

		// Cast
		sc.ActorDetails = make(map[string]models.ActorDetails)
		e.ForEach(`[data-test-component="VideoModels"] a`, func(id int, e *colly.HTMLElement) {
			sc.Cast = append(sc.Cast, strings.TrimSpace(e.Text))
			//sc.ActorDetails[strings.TrimSpace(e.Text)] = models.ActorDetails{Source: sc.ScraperID + " scrape", ProfileUrl: e.Request.AbsoluteURL(e.Attr("href"))}
		})

		// Filenames
		prefix := strings.ToUpper(strings.ReplaceAll(scraper, "raw", "_raw"))
		sc.Filenames = append(sc.Filenames, fmt.Sprintf("%s_%s_2160P.mp4", prefix, sceneId))
		sc.Filenames = append(sc.Filenames, fmt.Sprintf("%s_%s_1080P.mp4", prefix, sceneId))
		sc.Filenames = append(sc.Filenames, fmt.Sprintf("%s_%s_720P.mp4", prefix, sceneId))

		out <- sc
	})

	siteCollector.OnHTML(`a[data-test-component="PageNumberLink"]`, func(e *colly.HTMLElement) {
		if !limitScraping {
			pageURL := e.Request.AbsoluteURL(e.Attr("href"))
			WaitBeforeVisit("www."+scraperID+".com", siteCollector.Visit, pageURL)
		}
	})

	siteCollector.OnHTML(`a[href^="/videos/"]`, func(e *colly.HTMLElement) {
		sceneURL := e.Request.AbsoluteURL(e.Attr("href"))

		// If scene exist in database, there's no need to scrape
		if !funk.ContainsString(knownScenes, sceneURL) {
			WaitBeforeVisit("www."+scraperID+".com", sceneCollector.Visit, sceneURL)
		}
	})

	fileCollector.OnResponse(func(r *colly.Response) {
		log.Println("Visited", r.Request.URL)

		// Create the file
		fileName := common.MyFilesDir + `\` + scraperID + r.Request.URL.Path
		fileName = strings.ReplaceAll(fileName, `/`, `\`)
		imgpath := filepath.Dir(fileName)
		os.MkdirAll(imgpath, os.FileMode(0755))
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
	})

	if singleSceneURL != "" {
		sceneCollector.Visit(singleSceneURL)
	} else {
		WaitBeforeVisit("www."+scraperID+".com", siteCollector.Visit, "https://www."+scraperID+".com/videos")
	}

	if updateSite {
		updateSiteLastUpdate(scraperID)
	}
	logScrapeFinished(scraperID, siteID)
	return nil

}

func addVixenScraper(id string, name string, avatarURL string, site string) {
	registerScraper(id, name, avatarURL, site, func(wg *sync.WaitGroup, updateSite bool, knownScenes []string, out chan<- models.ScrapedScene, singleSceneURL string, singeScrapeAdditionalInfo string, limitScraping bool) error {
		return Vixen(wg, updateSite, knownScenes, out, singleSceneURL, singeScrapeAdditionalInfo, id, name, limitScraping)
	})
}

func init() {
	addVixenScraper("blacked", "Blacked", "/myfiles/blacked_favicon.ico", "www.blacked.com")
	addVixenScraper("blackedraw", "Blacked Raw", "/myfiles/blackedraw_favicon.ico", "www.blackedraw.com")
	addVixenScraper("deeper", "Deeper", "/myfiles/deeper_favicon.ico", "www.deeper.com")
	addVixenScraper("tushy", "Tushy", "/myfiles/tushy_favicon.ico", "www.tushy.com")
	addVixenScraper("tushyraw", "Tushy Raw", "/myfiles/tushyraw_favicon.ico", "www.tushyraw.com")
	addVixenScraper("vixen", "Vixen", "/myfiles/vixen_favicon.ico", "www.vixen.com")
}
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

type chapter struct {
	title   string
	seconds int
}
type categoriesAndChapters struct {
	categories []string
	chapters   []chapter
}

func GetCateegoriesAndChapters(sceneId string, scraper string) categoriesAndChapters {
	var categories []string
	var cueppoints []chapter
	query := fmt.Sprintf(`{"query":"query getRelatedVideos($videoSlug: ID!) { findOneVideo(input: {videoId: $videoSlug}) { id: uuid videoId slug categories {name} chapters { video { title seconds } }}  } ","operationName":"getRelatedVideos","variables":{"videoSlug":%s}}`, sceneId)
	sceneJson := string(callVixenGraphql(query, scraper, true))

	if gjson.Get(sceneJson, "data.findOneVideo").Exists() {
		videoJson := gjson.Get(sceneJson, "data.findOneVideo.categories")
		videoJson.ForEach(func(key, value gjson.Result) bool {
			category := value.Get("name")
			categories = append(categories, category.String())
			return true // return true to continue iterating
		})
		chaptersJson := gjson.Get(sceneJson, "data.findOneVideo.chapters.video")
		chaptersJson.ForEach(func(key, value gjson.Result) bool {
			cuepointTitle := value.Get("title").String()
			seconds := value.Get("seconds").Int()
			cueppoints = append(cueppoints, chapter{title: cuepointTitle, seconds: int(seconds)})
			return true // return true to continue iterating
		})
	}
	results := categoriesAndChapters{categories: categories, chapters: cueppoints}
	return results
}

func ProcessVixenWatchRequest(sceneId string, scraper string) models.VideoSourceResponse {
	trailers := models.VideoSourceResponse{
		VideoSources: []models.VideoSource{},
	}
	log.Infof("Geting Video Tokens for %s: %s", scraper, sceneId)

	query := fmt.Sprintf(`{"query":"query getToken($videoId: ID!, $device: Device!) { generateVideoToken(input: {videoId: $videoId, device: $device}) { p270 {token} p360 {token} p480 {token} p480l {token} p720 {token} p1080 {token} p2160 {token} hls {token} } } ","operationName":"getToken","variables":{"videoId":"%s","device":"desktop"}}`, sceneId)
	query = fmt.Sprintf(`{"operationName":"getToken","variables":{"videoId":"%s","device":"desktop"},"query":"query getToken($videoId: ID!, $device: Device!) { generateVideoToken(input: {videoId: $videoId, device: $device}) { p270 { token } p360 { token } p480 { token } p480l { token } p720 { token } p1080 { token } p2160 { token } hls { token } } } "}`, sceneId)
	sceneJson := string(callVixenGraphql(query, scraper, true))

	if gjson.Get(sceneJson, "data").Exists() {
		generateVideoToken := gjson.Get(sceneJson, "data.generateVideoToken")
		generateVideoToken.ForEach(func(key, value gjson.Result) bool {
			entryKey := key.String()
			if value.IsObject() {
				token := value.Get("token").String()
				trailers.VideoSources = append(trailers.VideoSources, models.VideoSource{URL: token, Quality: entryKey})
			}
			return true // return true to continue iterating
		})
	}
	return trailers
}

func callVixenGraphql(query string, scraper string, members bool) []byte {
	// Create an HTTP POST request to send the GraphQL query to the endpoint
	domainHead := "www."
	if members {
		domainHead = "members."
	}
	req, err := http.NewRequest("POST", "https://"+domainHead+"vixenplus.com/graphql", bytes.NewBuffer([]byte(query)))
	if err != nil {
		log.Infof("error geting new request in %s: %s", scraper, err)
	}

	if members {
		storedCookies := getCookies()
		//		req.AddCookie(&http.Cookie{Name: `access_token`, Value: storedCookies.AccessToken})
		//		req.AddCookie(&http.Cookie{Name: `refresh_token`, Value: storedCookies.RefreshTooken})
		//		req.AddCookie(&http.Cookie{Name: `sid`, Value: storedCookies.SID})
		//		req.AddCookie(&http.Cookie{Name: `vuid`, Value: storedCookies.VUID})
		if !checkTokenExpiry(storedCookies.AccessToken) {
			req.Header.Add("Authorization", "Bearer "+storedCookies.AccessToken)
		}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Connection", "keep-alive")

	callClient := func() []byte {
		var bodyBytes []byte
		defer func() {
			if r := recover(); r != nil {
				return
			}
		}()
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Infof("error client.do  in callStashDb %s", err)
		}

		defer resp.Body.Close()

		bodyBytes, _ = io.ReadAll(resp.Body)
		return bodyBytes
	}
	return callClient()
}

type vixenConfig struct {
	AccessToken   string `json:"accessToken"`
	RefreshTooken string `json:"refreshToken"`
	SID           string `json:"sid"`
	VUID          string `json:"vuid"`
}

func getCookies() vixenConfig {
	db, _ := models.GetCommonDB()

	config := vixenConfig{}
	var vixenConfig models.KV
	vixenConfig.Key = "vixen"
	db.Find(&vixenConfig)
	json.Unmarshal([]byte(vixenConfig.Value), &config)
	return config
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
