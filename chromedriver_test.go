package chromedriver

import (
	"testing"
	"time"

	goselenium "github.com/bunsenapp/go-selenium"
	"github.com/facebookgo/ensure"
)

func TestGoogle(t *testing.T) {
	capabilities := goselenium.Capabilities{}
	capabilities.SetBrowser(goselenium.ChromeBrowser())
	chromedriver, err := Start()
	ensure.Nil(t, err)
	defer chromedriver.StopOrFatal()
	driver, err := goselenium.NewSeleniumWebDriver(chromedriver.URL(), capabilities)
	ensure.Nil(t, err)
	_, err = driver.CreateSession()
	ensure.Nil(t, err)
	defer driver.DeleteSession()
	_, err = driver.Go("https://www.google.com")
	ensure.Nil(t, err)
	el, err := driver.FindElement(goselenium.ByCSSSelector("input[name='q']"))
	ensure.Nil(t, err)
	_, err = el.SendKeys("facebook")
	ensure.Nil(t, err)
	ensure.True(t, driver.Wait(
		goselenium.UntilElementPresent(goselenium.ByPartialLinkText("Facebook")),
		time.Minute,
		time.Millisecond*100,
	))
}
