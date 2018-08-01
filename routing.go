package githttp

import (
	"net/http"
	"os"
	"regexp"
	"strings"
)

type Service struct {
	Method  string
	Handler func(HandlerReq) error
	RPC     string
}

type HandlerReq struct {
	w    http.ResponseWriter
	r    *http.Request
	RPC  string
	Dir  string
	File string
}

// Routing regexes
var (
	_serviceRPCUpload  = regexp.MustCompile("(.*?)/git-upload-pack$")
	_serviceRPCReceive = regexp.MustCompile("(.*?)/git-receive-pack$")
	_getInfoRefs       = regexp.MustCompile("(.*?)/info/refs$")
	_getHead           = regexp.MustCompile("(.*?)/HEAD$")
	_getAlternates     = regexp.MustCompile("(.*?)/objects/info/alternates$")
	_getHTPPAlternates = regexp.MustCompile("(.*?)/objects/info/http-alternates$")
	_getInfoPacks      = regexp.MustCompile("(.*?)/objects/info/packs$")
	_getInfoFile       = regexp.MustCompile("(.*?)/objects/info/[^/]*$")
	_getLooseObject    = regexp.MustCompile("(.*?)/objects/[0-9a-f]{2}/[0-9a-f]{38}$")
	_getPackFile       = regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.pack$")
	_getIdxFile        = regexp.MustCompile("(.*?)/objects/pack/pack-[0-9a-f]{40}\\.idx$")
)

func (g *gitContext) services() map[*regexp.Regexp]Service {
	return map[*regexp.Regexp]Service{
		_serviceRPCUpload:  {"POST", g.serviceRPC, "upload-pack"},
		_serviceRPCReceive: {"POST", g.serviceRPC, "receive-pack"},
		_getInfoRefs:       {"GET", g.getInfoRefs, ""},
		_getHead:           {"GET", g.getTextFile, ""},
		_getAlternates:     {"GET", g.getTextFile, ""},
		_getHTPPAlternates: {"GET", g.getTextFile, ""},
		_getInfoPacks:      {"GET", g.getInfoPacks, ""},
		_getInfoFile:       {"GET", g.getTextFile, ""},
		_getLooseObject:    {"GET", g.getLooseObject, ""},
		_getPackFile:       {"GET", g.getPackFile, ""},
		_getIdxFile:        {"GET", g.getIdxFile, ""},
	}
}

// getService return's the service corresponding to the
// current http.Request's URL
// as well as the name of the repo
func (g *gitContext) getService(path string) (string, *Service) {
	for re, service := range g.services() {
		if m := re.FindStringSubmatch(path); m != nil {
			return m[1], &service
		}
	}

	// No match
	return "", nil
}

// Request handling function
func (g *gitContext) requestHandler(w http.ResponseWriter, r *http.Request) {
	// Get service for URL
	repo, service := g.getService(r.URL.Path)

	// No url match
	if service == nil {
		renderNotFound(w)
		return
	}

	// Bad method
	if service.Method != r.Method {
		renderMethodNotAllowed(w, r)
		return
	}

	// RPC type
	rpc := service.RPC

	// Get specific file
	file := strings.Replace(r.URL.Path, repo+"/", "", 1)

	// Resolve directory
	dir, err := g.getGitDir(repo)

	// Repo not found on disk
	if err != nil {
		renderNotFound(w)
		return
	}

	// Build request info for handler
	hr := HandlerReq{w, r, rpc, dir, file}

	// Call handler
	if err := service.Handler(hr); err != nil {
		if os.IsNotExist(err) {
			renderNotFound(w)
			return
		}
		switch err.(type) {
		case *ErrorNoAccess:
			renderNoAccess(w)
			return
		}
		http.Error(w, err.Error(), 500)
	}
}
