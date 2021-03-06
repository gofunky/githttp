githttp
===========

[![Build Status](https://travis-ci.org/gofunky/githttp.svg)](https://travis-ci.org/gofunky/githttp)
[![GoDoc](https://godoc.org/github.com/gofunky/githttp?status.svg)](https://godoc.org/github.com/gofunky/githttp)
[![Go Report Card](https://goreportcard.com/badge/github.com/gofunky/githttp)](https://goreportcard.com/report/github.com/gofunky/githttp)
[![Codacy Badge](https://api.codacy.com/project/badge/Grade/eeef5dbed01a4f84a76c2bf96fb8a158)](https://www.codacy.com/app/gofunky/githttp?utm_source=github.com&amp;utm_medium=referral&amp;utm_content=gofunky/githttp&amp;utm_campaign=Badge_Grade)

A Smart Git Http server library in Go (golang)

### Example

```go
package main

import (
    "log"
    "net/http"
    "crypto/sha1"
    "github.com/gofunky/githttp"
)

func main() {
    // Get git handler to serve a directory of repos
    git, err := githttp.NewGitContext(githttp.GitOptions{
    	ProjectRoot: "my/repos",
    	AutoCreate: true,
    	ReceivePack: true,
    	UploadPack: true,
    	NoBare: true,
    	EventHandler: func(ev githttp.Event) {
    	    if ev.Error != nil {
    	    	log.Fatal(ev)
    	    }
    	},
    	Prep: func() githttp.Preprocesser {
    		return githttp.Preprocesser{
            Process:func(params *githttp.ProcessParams) error {
            	if params.IsNew {
            		// E.g., generate .gitignore file
            	}
            	return nil
    		},
            Path:func(rawPath string) (targetPath string, err error) {
            	// Lets hash the string
            	h := sha1.New()
            	h.Write([]byte(rawPath))
            	// Resulting target path will be "./my/repos/{hash(rawPath)}/
            	return string(h.Sum(nil)), nil
            },
    	}},
    })
    // Panic if the server context couldn't be created
    if err != nil {
    	panic(err)
    }

    // Attach handler to http server
    http.Handle("/", git)

    // Start HTTP server
    err = http.ListenAndServe(":8080", nil)
    if err != nil {
        panic(err)
    }
}
```

### Authentication example

```go
package main

import (
    "log"
    "net/http"

    "github.com/gofunky/githttp"
    "github.com/gofunky/githttp/auth"
)


func main() {
    // Get git handler to serve a directory of repos
    git, err := githttp.NewGitContext(githttp.GitOptions{
    	ProjectRoot: "my/repos",
    	ReceivePack: true,
    	UploadPack: true,
    })
    // Panic if the server context couldn't be created
    if err != nil {
    	panic(err)
    }

    // Build an authentication middleware based on a function
    authenticator := auth.Authenticator(func(info auth.AuthInfo) (bool, error) {
        // Disallow Pushes (making git server pull only)
        if info.Push {
            return false, nil
        }

        // Typically this would be a database lookup
        if info.Username == "admin" && info.Password == "password" {
            return true, nil
        }

        return false, nil
    })

    // Attach handler to http server
    // wrap authenticator around git handler
    http.Handle("/", authenticator(git))

    // Start HTTP server
    err = http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
```

