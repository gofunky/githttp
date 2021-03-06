package githttp

import (
	"fmt"
	gogit "gopkg.in/src-d/go-git.v4"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type (
	// GitHTTP exposes the interfaces of the git server.
	GitHTTP interface {
		Init() (*gitContext, error)
		ServeHTTP(w http.ResponseWriter, r *http.Request)
	}

	// gitContext is the context on that the git server operates on.
	gitContext struct {
		options GitOptions
	}

	// GitOptions contains the the git server options.
	GitOptions struct {
		// Root directory to serve repos from
		ProjectRoot string

		// Path to git binary
		GitBinPath string

		// Access rules
		UploadPack  bool
		ReceivePack bool

		// To disable bare init
		NoBare bool

		// Implicit generation
		AutoCreate bool

		// May be used to create a common context for the preprocessing funcs
		Prep func() Preprocesser

		// Event handling functions
		EventHandler func(ev Event)
	}
)

// NewGitContext is the factory for the git server.
func NewGitContext(options GitOptions) (context GitHTTP, err error) {
	if options.ProjectRoot == "" {
		return nil, ErrMissingArgument
	}
	if options.GitBinPath == "" {
		binary, lookErr := exec.LookPath("git")
		if lookErr != nil {
			return nil, ErrMissingArgument
		}
		options.GitBinPath = binary
	}
	return &gitContext{
		options: options,
	}, nil
}

// Implement the http.Handler interface
func (g *gitContext) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.requestHandler(w, r)
	return
}

// Build root directory if doesn't exist
func (g *gitContext) Init() (*gitContext, error) {
	if err := os.MkdirAll(g.options.ProjectRoot, os.ModePerm); err != nil {
		return nil, err
	}
	return g, nil
}

// Publish event if EventHandler is set
func (g *gitContext) event(e Event) {
	if g.options.EventHandler != nil {
		g.options.EventHandler(e)
	} else {
		fmt.Printf("EVENT: %q\n", e.Error)
	}
}

// Actual command handling functions

func (g *gitContext) serviceRPC(hr HandlerReq) error {
	w, r, rpc, dir := hr.w, hr.r, hr.RPC, hr.Dir

	access, err := g.hasAccess(r, dir, rpc, true)
	if err != nil {
		return err
	}

	if access == false {
		return &ErrorNoAccess{hr.Dir}
	}

	// Reader that decompresses if necessary
	reader, err := requestReader(r)
	if err != nil {
		return err
	}
	defer reader.Close()

	// Reader that scans for events
	rpcReader := &RpcReader{
		Reader: reader,
		Rpc:    rpc,
	}

	// Set content type
	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-result", rpc))

	args := []string{rpc, "--stateless-rpc", "."}
	cmd := exec.Command(g.options.GitBinPath, args...)
	cmd.Dir = dir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	defer stdout.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	// Scan's git command's output for errors
	gitReader := &GitReader{
		Reader: stdout,
	}

	// Copy input to git binary
	io.Copy(stdin, rpcReader)
	stdin.Close()

	// Write git binary's output to http response
	io.Copy(w, gitReader)

	// Wait till command has completed
	mainError := cmd.Wait()

	if mainError == nil {
		mainError = gitReader.GitError
	}

	// Fire events
	for _, e := range rpcReader.Events {
		// Set directory to current repo
		e.Dir = dir
		e.Request = hr.r
		e.Error = mainError

		// Fire event
		g.event(e)
	}

	// Because a response was already written,
	// the header cannot be changed
	return nil
}

func (g *gitContext) getInfoRefs(hr HandlerReq) error {
	w, r, dir := hr.w, hr.r, hr.Dir
	serviceName := getServiceType(r)
	access, err := g.hasAccess(r, dir, serviceName, false)
	if err != nil {
		return err
	}

	if !access {
		g.updateServerInfo(dir)
		hdrNocache(w)
		return sendFile("text/plain; charset=utf-8", hr)
	}

	args := []string{serviceName, "--stateless-rpc", "--advertise-refs", "."}
	refs, err := g.gitCommand(dir, args...)
	if err != nil {
		return err
	}

	hdrNocache(w)
	w.Header().Set("Content-Type", fmt.Sprintf("application/x-git-%s-advertisement", serviceName))
	w.WriteHeader(http.StatusOK)
	w.Write(packetWrite("# service=git-" + serviceName + "\n"))
	w.Write(packetFlush())
	w.Write(refs)

	return nil
}

func (g *gitContext) getInfoPacks(hr HandlerReq) error {
	hdrCacheForever(hr.w)
	return sendFile("text/plain; charset=utf-8", hr)
}

func (g *gitContext) getLooseObject(hr HandlerReq) error {
	hdrCacheForever(hr.w)
	return sendFile("application/x-git-loose-object", hr)
}

func (g *gitContext) getPackFile(hr HandlerReq) error {
	hdrCacheForever(hr.w)
	return sendFile("application/x-git-packed-objects", hr)
}

func (g *gitContext) getIdxFile(hr HandlerReq) error {
	hdrCacheForever(hr.w)
	return sendFile("application/x-git-packed-objects-toc", hr)
}

func (g *gitContext) getTextFile(hr HandlerReq) error {
	hdrNocache(hr.w)
	return sendFile("text/plain", hr)
}

// Logic helping functions

func sendFile(contentType string, hr HandlerReq) error {
	w, r := hr.w, hr.r
	reqFile := path.Join(hr.Dir, hr.File)

	f, err := os.Stat(reqFile)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", f.Size()))
	w.Header().Set("Last-Modified", f.ModTime().Format(http.TimeFormat))
	http.ServeFile(w, r, reqFile)

	return nil
}

func (g *gitContext) getGitDir(repoPath string) (targetPath string, err error) {
	options := g.options
	root := options.ProjectRoot

	if root == "" {
		cwd, err := os.Getwd()

		if err != nil {
			return "", err
		}

		root = cwd
	}

	var subDir string

	// Create preprocessing context
	var prepper = &Preprocesser{}
	if options.Prep != nil {
		optPrep := options.Prep()
		prepper = &optPrep
	}

	if !prepper.IsPathNil() {
		subDir, err = prepper.Path(repoPath)
		if err != nil {
			return "", err
		}
	} else {
		subDir = repoPath
	}
	localPath := path.Join(root, subDir)
	absPath, err := filepath.Abs(localPath)
	if err != nil {
		return "", err
	}
	var isNew bool
	var repo *gogit.Repository
	if checkRepo, err := gogit.PlainOpen(absPath); err == nil {
		repo = checkRepo
	} else {
		// If AutoCreate is false, just bail
		if !options.AutoCreate {
			return "", err
		}
		// If AutoCreate is true, attempt to create and initialise the directory
		err = os.MkdirAll(localPath, os.ModePerm)
		if err != nil {
			return "", err
		}
		repo, err = gogit.PlainInit(absPath, !options.NoBare)
		if err != nil {
			return "", err
		}
		isNew = true
	}
	if !prepper.IsProcessNil() {
		err := prepper.Process(&ProcessParams{
			RepositoryPath: repoPath,
			LocalPath:      absPath,
			IsNew:          isNew,
			Repository:     repo,
		})
		if err != nil {
			return "", err
		}
	}

	return localPath, nil
}

func (g *gitContext) hasAccess(r *http.Request, dir string, rpc string, checkContentType bool) (bool, error) {
	if checkContentType {
		if r.Header.Get("Content-Type") != fmt.Sprintf("application/x-git-%s-request", rpc) {
			return false, nil
		}
	}

	if !(rpc == "upload-pack" || rpc == "receive-pack") {
		return false, nil
	}
	if rpc == "receive-pack" {
		return g.options.ReceivePack, nil
	}
	if rpc == "upload-pack" {
		return g.options.UploadPack, nil
	}

	return g.getConfigSetting(rpc, dir)
}

func (g *gitContext) getConfigSetting(serviceName string, dir string) (bool, error) {
	serviceName = strings.Replace(serviceName, "-", "", -1)
	setting, err := g.getGitConfig("http."+serviceName, dir)
	if err != nil {
		return false, nil
	}

	if serviceName == "uploadpack" {
		return setting != "false", nil
	}

	return setting == "true", nil
}

func (g *gitContext) getGitConfig(configName string, dir string) (string, error) {
	args := []string{"config", configName}
	out, err := g.gitCommand(dir, args...)
	if err != nil {
		return "", err
	}
	return string(out)[0 : len(out)-1], nil
}

func (g *gitContext) updateServerInfo(dir string) ([]byte, error) {
	args := []string{"update-server-info"}
	return g.gitCommand(dir, args...)
}

func (g *gitContext) gitCommand(dir string, args ...string) ([]byte, error) {
	command := exec.Command(g.options.GitBinPath, args...)
	command.Dir = dir

	return command.Output()
}
