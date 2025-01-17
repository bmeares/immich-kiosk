package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/charmbracelet/log"
	"github.com/labstack/echo/v4"

	"github.com/damongolding/immich-kiosk/config"
	"github.com/damongolding/immich-kiosk/immich"
	"github.com/damongolding/immich-kiosk/utils"
)

var (
	KioskVersion  string
	ExampleConfig []byte
	baseConfig    config.Config
)

type PageData struct {
	// KioskVersion the current build version of Kiosk
	KioskVersion string
	// ImageData image as base64 data
	ImageData string
	// ImageData blurred image as base64 data
	ImageBlurData string
	// Date image date
	ImageDate string
	// instance config
	config.Config
}

type ErrorData struct {
	Title   string
	Message string
}

type ClockData struct {
	ClockTime string
	ClockDate string
}

func init() {
	err := baseConfig.Load()
	if err != nil {
		log.Fatal(err)
	}
}

// Home home endpoint
func Home(c echo.Context) error {

	if log.GetLevel() == log.DebugLevel {
		fmt.Println()
	}

	requestId := fmt.Sprintf("[%s]", c.Response().Header().Get(echo.HeaderXRequestID))

	// create a copy of the global config to use with this instance
	instanceConfig := baseConfig

	queries, err := utils.CombineQueries(c.Request().URL.Query(), c.Request().Referer())
	if err != nil {
		log.Error("err combining queries", "err", err)
	}

	if len(queries) > 0 {
		instanceConfig = instanceConfig.ConfigWithOverrides(queries)
	}

	log.Debug(requestId, "path", c.Request().URL.String(), "instanceConfig", instanceConfig)

	pageData := PageData{
		KioskVersion: KioskVersion,
		Config:       instanceConfig,
	}

	return c.Render(http.StatusOK, "index.tmpl", pageData)

}

// NewImage new image endpoint
func NewImage(c echo.Context) error {

	if log.GetLevel() == log.DebugLevel {
		fmt.Println()
	}

	requestId := fmt.Sprintf("[%s]", c.Response().Header().Get(echo.HeaderXRequestID))

	// create a copy of the global config to use with this instance
	instanceConfig := baseConfig

	queries, err := utils.CombineQueries(c.Request().URL.Query(), c.Request().Referer())
	if err != nil {
		log.Error("err combining queries", "err", err)
	}

	if len(queries) > 0 {
		instanceConfig = instanceConfig.ConfigWithOverrides(queries)
	}

	log.Debug(requestId, "path", c.Request().URL.String(), "config", instanceConfig)

	immichImage := immich.NewImage(baseConfig)

	switch {
	case instanceConfig.Album != "":
		randomAlbumImageErr := immichImage.GetRandomImageFromAlbum(instanceConfig.Album, requestId)
		if randomAlbumImageErr != nil {
			log.Error("err getting image from album", "err", randomAlbumImageErr)
			return c.Render(http.StatusOK, "error.tmpl", ErrorData{Title: "Error getting image from album", Message: "Is album ID correct?"})
		}
		break
	case instanceConfig.Person != "":
		randomPersonImageErr := immichImage.GetRandomImageOfPerson(instanceConfig.Person, requestId)
		if randomPersonImageErr != nil {
			log.Error("err getting image of person", "err", randomPersonImageErr)
			return c.Render(http.StatusOK, "error.tmpl", ErrorData{Title: "Error getting image of person", Message: "Is person ID correct?"})
		}
		break
	default:
		randomImageErr := immichImage.GetRandomImage(requestId)
		if randomImageErr != nil {
			log.Error("err getting random image", "err", randomImageErr)
			return c.Render(http.StatusOK, "error.tmpl", ErrorData{Title: "Error getting random image", Message: "Is Immich running? Are your config settings correct?"})
		}
	}

	imageGet := time.Now()
	imgBytes, err := immichImage.GetImagePreview()
	if err != nil {
		return err
	}
	log.Debug(requestId, "Got image in", time.Since(imageGet).Seconds())

	// if user wants the raw image data send it
	if c.Request().URL.Query().Has("raw") {
		return c.Blob(http.StatusOK, immichImage.OriginalMimeType, imgBytes)
	}

	imageConvertTime := time.Now()
	img, err := utils.ImageToBase64(imgBytes)
	if err != nil {
		return err
	}
	log.Debug(requestId, "Converted image in", time.Since(imageConvertTime).Seconds())

	var imgBlur string

	if instanceConfig.BackgroundBlur {
		imageBlurTime := time.Now()
		imgBlurBytes, err := utils.BlurImage(imgBytes)
		if err != nil {
			return err
		}
		imgBlur, err = utils.ImageToBase64(imgBlurBytes)
		if err != nil {
			return err
		}
		log.Debug(requestId, "Blurred image in", time.Since(imageBlurTime).Seconds())
	}

	// Image METADATA
	var imageDate string

	var imageTimeFormat string
	if instanceConfig.ImageTimeFormat == "12" {
		imageTimeFormat = time.Kitchen
	} else {
		imageTimeFormat = time.TimeOnly
	}

	imageDateFormat := instanceConfig.ImageDateFormat
	if imageDateFormat == "" {
		imageDateFormat = "02/01/2006"
	}

	switch {
	case (instanceConfig.ShowImageDate && instanceConfig.ShowImageTime):
		imageDate = fmt.Sprintf("%s %s", immichImage.LocalDateTime.Format(imageDateFormat), immichImage.LocalDateTime.Format(imageTimeFormat))
		break
	case instanceConfig.ShowImageDate:
		imageDate = fmt.Sprintf("%s", immichImage.LocalDateTime.Format(imageDateFormat))
		break
	case instanceConfig.ShowImageTime:
		imageDate = fmt.Sprintf("%s", immichImage.LocalDateTime.Format(imageTimeFormat))
		break
	}

	data := PageData{
		ImageData:     img,
		ImageBlurData: imgBlur,
		ImageDate:     imageDate,
		Config:        instanceConfig,
	}

	return c.Render(http.StatusOK, "image.tmpl", data)
}

// Clock clock endpoint
func Clock(c echo.Context) error {

	if log.GetLevel() == log.DebugLevel {
		fmt.Println()
	}

	requestId := fmt.Sprintf("[%s]", c.Response().Header().Get(echo.HeaderXRequestID))

	// create a copy of the global config to use with this instance
	instanceConfig := baseConfig

	queries, err := utils.CombineQueries(c.Request().URL.Query(), c.Request().Referer())
	if err != nil {
		log.Error("err combining queries", "err", err)
	}

	if len(queries) > 0 {
		instanceConfig = instanceConfig.ConfigWithOverrides(queries)
	}

	log.Debug(requestId, "path", c.Request().URL.String(), "config", instanceConfig)

	t := time.Now()

	clockTimeFormat := "15:04"
	if instanceConfig.TimeFormat == "12" {
		clockTimeFormat = time.Kitchen
	}

	clockDateFormat := instanceConfig.DateFormat
	if clockDateFormat == "" {
		clockDateFormat = "02/01/2006"
	}

	var data ClockData

	switch {
	case (instanceConfig.ShowTime && instanceConfig.ShowDate):
		data.ClockTime = t.Format(clockTimeFormat)
		data.ClockDate = t.Format(clockDateFormat)
		break
	case instanceConfig.ShowTime:
		data.ClockTime = t.Format(clockTimeFormat)
		break
	case instanceConfig.ShowDate:
		data.ClockDate = t.Format(clockDateFormat)
	}

	return c.Render(http.StatusOK, "clock.tmpl", data)
}
