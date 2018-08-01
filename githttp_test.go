package githttp

import (
	"errors"
	"gopkg.in/src-d/go-git.v4"
	gogit "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	gitRefs    = "/info/refs?service="
	uploadPack = "git-upload-pack"
	testFile   = "test.file"
)

func TestNewGitContext(t *testing.T) {
	type args struct {
		options GitOptions
		uri     string
		check   func() error
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		wantRespErr bool
		wantResp    string
		prepare     func()
		cleanup     func()
	}{
		{
			name: "Basic git access test",
			args: args{
				options: GitOptions{
					ProjectRoot: "./testdata",
					AutoCreate:  true,
					ReceivePack: true,
					UploadPack:  true,
				},
				uri: "/my/repo" + gitRefs + uploadPack,
				check: func() error {
					if _, err := os.Stat("./testdata/my/repo/HEAD"); os.IsNotExist(err) {
						return errors.New("git HEAD does not exist")
					}
					return nil
				},
			},
			cleanup: func() {
				err := os.RemoveAll("./testdata/my/")
				if err != nil {
					panic(err)
				}
			},
			wantResp: "service=git-upload-pack",
		},
		{
			name: "Existing git access test",
			args: args{
				options: GitOptions{
					ProjectRoot: "./testdata",
					ReceivePack: true,
					UploadPack:  true,
				},
				uri: "/existing/repo" + gitRefs + uploadPack,
				check: func() error {
					if _, err := os.Stat("./testdata/existing/repo/HEAD"); os.IsNotExist(err) {
						return errors.New("git HEAD does not exist")
					}
					return nil
				},
			},
			prepare: func() {
				_, err := initRepo("./testdata/existing/repo", true, false)
				if err != nil {
					panic(err)
				}
			},
			cleanup: func() {
				err := os.RemoveAll("./testdata/existing/")
				if err != nil {
					panic(err)
				}
			},
			wantResp: "service=git-upload-pack",
		},
		{
			name: "Nonexisting git access test",
			args: args{
				options: GitOptions{
					ProjectRoot: "./testdata",
					ReceivePack: true,
					UploadPack:  true,
				},
				uri: "/invalid/repo" + gitRefs + uploadPack,
				check: func() error {
					if _, err := os.Stat("./testdata/invalid/repo/HEAD"); os.IsNotExist(err) {
						return nil
					}
					return errors.New("git HEAD unexpectedly exists")
				},
			},
			wantRespErr: true,
		},
		{
			name: "Existing nonbare git access test",
			args: args{
				options: GitOptions{
					ProjectRoot: "./testdata",
					AutoCreate:  true,
					NoBare:      true,
					ReceivePack: true,
					UploadPack:  true,
				},
				uri: "/nonbare/repo" + gitRefs + uploadPack,
				check: func() error {
					if _, err := os.Stat("./testdata/nonbare/repo/.git/HEAD"); os.IsNotExist(err) {
						return errors.New("git HEAD does not exist")
					}
					return nil
				},
			},
			prepare: func() {
				_, err := initRepo("./testdata/nonbare/repo", false, true)
				if err != nil {
					panic(err)
				}
			},
			cleanup: func() {
				err := os.RemoveAll("./testdata/nonbare/")
				if err != nil {
					panic(err)
				}
			},
			wantResp: "refs/heads/master",
		},
		{
			name: "Nonbare git access test",
			args: args{
				options: GitOptions{
					ProjectRoot: "./testdata",
					AutoCreate:  true,
					NoBare:      true,
					ReceivePack: true,
					UploadPack:  true,
				},
				uri: "/nonbare/new/repo" + gitRefs + uploadPack,
				check: func() error {
					if _, err := os.Stat("./testdata/nonbare/new/repo/.git/HEAD"); os.IsNotExist(err) {
						return errors.New("git HEAD does not exist")
					}
					return nil
				},
			},
			cleanup: func() {
				err := os.RemoveAll("./testdata/nonbare/")
				if err != nil {
					panic(err)
				}
			},
			wantResp: "service=git-upload-pack",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
			}

			rr := httptest.NewRecorder()
			gotContext, err := NewGitContext(tt.args.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGitContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			gotContext.Init()

			req, err := http.NewRequest("GET", tt.args.uri, nil)
			if err != nil {
				panic(err)
			}

			http.Handle(tt.name, gotContext)
			gotContext.ServeHTTP(rr, req)

			// Check the status code is what we expect.
			if status := rr.Code; (status != http.StatusOK) != tt.wantRespErr {
				t.Errorf("handler returned wrong status code: got %v want %v",
					status, http.StatusOK)
			}

			// Check the response body is what we expect.
			if !strings.Contains(rr.Body.String(), tt.wantResp) {
				t.Errorf("handler returned unexpected body: got %v want %v",
					rr.Body.String(), tt.wantResp)
			}

			// Check the subtest case.
			if tt.args.check != nil {
				if err := tt.args.check(); err != nil {
					t.Errorf("test case failed its check test: %v", err)
				}
			}

			if tt.cleanup != nil {
				tt.cleanup()
			}
		})
	}
}

func initRepo(path string, bare bool, withTestFile bool) (*git.Repository, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(absPath, os.ModePerm)
	if err != nil {
		return nil, err
	}
	repo, err := gogit.PlainInit(absPath, bare)
	if err != nil {
		return nil, err
	}
	if withTestFile {
		w, err := repo.Worktree()
		if err != nil {
			return nil, err
		}
		testFilePath := filepath.Join(absPath, testFile)
		err = ioutil.WriteFile(testFilePath, []byte("hello world!"), os.ModePerm)
		if err != nil {
			return nil, err
		}
		_, err = w.Add(testFile)
		if err != nil {
			return nil, err
		}
		_, err = w.Commit("example go-git commit", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "John Doe",
				Email: "john@doe.org",
				When:  time.Now(),
			},
		})
		if err != nil {
			return nil, err
		}
	}
	return repo, nil
}
