package webserver

import (
	"github.com/gin-gonic/gin"
	"regexp"
	"strconv"
)

type WebRoute struct {
	Path    string
	Method  string
	Handler func(ctx *gin.Context)
}

type WebService interface {
	GinRoutes() []WebRoute
	AltRoutes() []WebRoute
	Middlewares() []func(ctx *gin.Context)
	Init(gin *gin.Engine) error
}

/*
simple user agent string parser
*/

type UserAgent struct {
	Family string
	Major  uint64
	Minor  uint64
	Patch  uint64
}

//check if UA is one of UAvalues
func (ua *UserAgent) Is(UAvalues ...string) bool {
	for _, val := range UAvalues {
		if ua.Family == val {
			return true
		}
	}
	return false
}

type UserAgentRegexp struct {
	Family   string
	UaRegexp string
}

const (
	UA_CHROME_MOBILE = "Chrome Mobile"
	UA_CHROME        = "Chrome"
	UA_YANDEX        = "Yandex Browser"
	UA_MIUI          = "MiuiBrowser"
	UA_WEBVIEW       = "Chrome Mobile WebView"
	UA_CHROME_IOS    = "Chrome Mobile iOS"
	UA_FIREFOX       = "Firefox"
	UA_OTHER         = "OTHER"
)

/*
based on https://github.com/ua-parser/uap-core
*/
var uaList = []UserAgentRegexp{
	{UA_MIUI, `(MiuiBrowser)/(\d+)\.(\d+)\.(\d+)`},
	{UA_YANDEX, `(YaBrowser)/(\d+)\.(\d+)\.(\d+)`},
	{UA_WEBVIEW, `Version/.+(Chrome)/(\d+)\.(\d+)\.(\d+)\.(\d+)`},
	{UA_WEBVIEW, `; wv\).+(Chrome)/(\d+)\.(\d+)\.(\d+)\.(\d+)`},
	{UA_CHROME_IOS, `(CriOS)/(\d+)\.(\d+)\.(\d+)\.(\d+)`},
	{UA_CHROME_MOBILE, `(Chrome)/(\d+)\.(\d+)\.(\d+)\.(\d+) Mobile(?:[ /]|$)`},
	{UA_CHROME, `(Chromium|Chrome)/(\d+)\.(\d+)(?:\.(\d+)|)(?:\.(\d+)|)`},
	{UA_FIREFOX, `(Firefox)/(\d+)\.(\d+)`},
}

var uaRegexp []*regexp.Regexp

//todo the best way is to use https://github.com/ua-parser/uap-go
//this is simplified version that's enough to our purpose
func DetectUA(UAstring string) UserAgent {
	if len(uaRegexp) == 0 {
		for _, uaDescriptor := range uaList {
			uaRegexp = append(uaRegexp, regexp.MustCompile(uaDescriptor.UaRegexp))
		}
	}
	for _, regexp := range uaRegexp {
		matches := regexp.FindStringSubmatchIndex(UAstring)
		if len(matches) > 0 {
			var ua UserAgent
			ua.Family = string(regexp.ExpandString(nil, "$1", UAstring, matches))
			ua.Major, _ = strconv.ParseUint(string(regexp.ExpandString(nil, "$2", UAstring, matches)), 10, 64)
			ua.Minor, _ = strconv.ParseUint(string(regexp.ExpandString(nil, "$3", UAstring, matches)), 10, 64)
			ua.Patch, _ = strconv.ParseUint(string(regexp.ExpandString(nil, "$4", UAstring, matches)), 10, 64)
			return ua
		}
	}
	return UserAgent{
		Family: UA_OTHER,
	}
}

/*
 encode string compatible with Javascript DecodeURIComponent

func EncodeURIComponent(str string) (encodedString string) {

	repl := map[string]string {
		"+": "%20",
		"%21": "!",
		"%27": "'",
		"%28": "(",
		"%29": ")",
		"%2A": "*",
	}
	encodedString = url.QueryEscape(str)
	for oldValue, newValue := range repl {
		encodedString = strings.Replace(encodedString, oldValue, newValue, -1)
	}
	return
}
*/
