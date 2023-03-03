package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/gin-gonic/gin"
)

var pathTable = make(map[string]*url.URL)

var (
	ErrApplyPatch = errors.New("failed to apply patch")
)

type ForwardPatch struct {
	Path string `json:"path"`
	Dest string `json:"dest"`
}

func addForward(c *gin.Context) {
	var patch ForwardPatch
	if err := c.ShouldBindJSON(&patch); err != nil {
		logger.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrApplyPatch})
		return
	}
	if err := applyPath(patch.Path, patch.Dest); err != nil {
		logger.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrApplyPatch})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "patch applied"})
}

func applyPath(path, dest string) error {
	if _, ok := pathTable[path]; ok {
		return errors.New("path already exists")
	}
	var url *url.URL
	var err error
	if url, err = validatePath(dest); err != nil {
		return err
	}
	logger.Printf("Adding path %s -> %s\n", path, dest)
	pathTable[path] = url
	return nil
}

func validatePath(rawurl string) (*url.URL, error) {
	if rawurl == "" {
		return nil, errors.New("path cannot be empty")
	}
	url, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	if url.Scheme == "" {
		return nil, errors.New("path must contain a scheme")
	}
	if url.Host == "" {
		return nil, errors.New("path must contain a host")
	}
	if url.Port() == "" {
		if url.Scheme == "http" {
			url.Host = fmt.Sprintf("%s:80", url.Host)
		} else if url.Scheme == "https" {
			url.Host = fmt.Sprintf("%s:443", url.Host)
		} else {
			return nil, errors.New("path must contain a port")
		}
	}
	if resp, err := http.DefaultClient.Get(url.String()); err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, errors.New("path is not reachable")
	}
	return url, nil
}

func getForward(c *gin.Context) {
	logger.Println("Forwarding request...")
	path := c.Param("dest")
	dest, ok := pathTable[path]
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "path not found"})
		return
	}
	c.Request = rewrite(dest, c.Request)
	c.Request.Header.Set("X-Forwarded-For", c.ClientIP())
	c.Request.Header.Set("X-Forwarded-Host", c.Request.Host)
	c.Request.Header.Set("X-Forwarded-Proto", c.Request.Proto)

	proxy := httputil.NewSingleHostReverseProxy(dest)
	proxy.ServeHTTP(c.Writer, c.Request)
}

func rewrite(dest *url.URL, req *http.Request) *http.Request {
	req.Host = dest.Host
	req.RequestURI = dest.RequestURI()
	req.URL.Host = dest.Host
	req.URL.Scheme = dest.Scheme
	req.URL.Path = dest.Path
	req.URL.RawQuery = dest.RawQuery
	req.URL.Fragment = dest.Fragment
	return req
}

type Patch struct {
	Path string `json:"path"`
	Dest string `json:"dest"`
}

type PatchList struct {
	Patches []Patch `json:"patches"`
}

func ParsePatches(raw string) (PatchList, error) {
	var patches []Patch
	if err := json.Unmarshal([]byte(raw), &patches); err != nil {
		return PatchList{Patches: patches}, err
	}
	return PatchList{Patches: patches}, nil
}

func (p PatchList) Apply() error {
	if os.Getenv("ENVIRONMENT") == "DEVELOPMENT" {
		fmt.Printf("Patches: %v\n", p.Patches)
	}
	for _, patch := range p.Patches {
		if err := applyPath(patch.Path, patch.Dest); err != nil {
			return err
		}
	}
	return nil
}
