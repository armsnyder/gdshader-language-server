// Copyright (c) 2026 Adam Snyder <https://armsnyder.com> and contributors
// SPDX-License-Identifier: MIT

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"time"
)

// GitHubFS is an [fs.FS] implementation that reads files from a GitHub
// repository.
type GitHubFS struct {
	Client *http.Client
	Repo   string
	Ref    string
}

// Open implements [fs.ReadDirFS].
func (g *GitHubFS) Open(name string) (fs.File, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", g.Repo, name, url.QueryEscape(g.Ref)), http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.object+json")

	resp, err := g.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(resp.Status)
	}

	entry := &gitHubEntry{client: g.Client}

	if err := json.NewDecoder(resp.Body).Decode(&entry.data); err != nil {
		return nil, fmt.Errorf("decode response body: %w", err)
	}

	return entry, nil
}

var _ fs.FS = (*GitHubFS)(nil)

type gitHubEntry struct {
	data struct {
		Name        string         `json:"name"`
		Path        string         `json:"path"`
		Size        int64          `json:"size"`
		DownloadURL string         `json:"download_url"`
		Type        string         `json:"type"`
		Content     string         `json:"content"`
		Encoding    string         `json:"encoding"`
		Entries     []*gitHubEntry `json:"entries"`
	}

	client       *http.Client
	downloadResp *http.Response
}

// Info implements [fs.DirEntry].
func (g *gitHubEntry) Info() (fs.FileInfo, error) { return g, nil }

// Type implements [fs.DirEntry].
func (g *gitHubEntry) Type() fs.FileMode { return g.Mode().Type() }

// IsDir implements [fs.FileInfo].
func (g *gitHubEntry) IsDir() bool { return g.data.Type == "dir" }

// ModTime implements [fs.FileInfo].
func (g *gitHubEntry) ModTime() time.Time { return time.Time{} }

// Mode implements [fs.FileInfo].
func (g *gitHubEntry) Mode() fs.FileMode {
	if g.data.Type == "dir" {
		return fs.ModeDir | 0o700
	}
	return 0o600
}

// Name implements [fs.FileInfo].
func (g *gitHubEntry) Name() string { return g.data.Name }

// Size implements [fs.FileInfo].
func (g *gitHubEntry) Size() int64 { return g.data.Size }

// Sys implements [fs.FileInfo].
func (g *gitHubEntry) Sys() any { return nil }

// Close implements [fs.ReadDirFile].
func (g *gitHubEntry) Close() error {
	if g.downloadResp != nil {
		return g.downloadResp.Body.Close()
	}
	return nil
}

// Read implements [fs.ReadDirFile].
func (g *gitHubEntry) Read(b []byte) (int, error) {
	if g.data.Type == "dir" {
		return 0, &fs.PathError{Op: "read", Path: g.data.Path, Err: errors.New("is a directory")}
	}

	if g.downloadResp == nil {
		resp, err := g.client.Get(g.data.DownloadURL)
		if err != nil {
			return 0, err
		}
		if resp.StatusCode != http.StatusOK {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			return 0, errors.New(resp.Status)
		}
		g.downloadResp = resp
	}

	return g.downloadResp.Body.Read(b)
}

// Stat implements [fs.ReadDirFile].
func (g *gitHubEntry) Stat() (fs.FileInfo, error) {
	return g, nil
}

// ReadDir implements [fs.ReadDirFile].
func (g *gitHubEntry) ReadDir(n int) ([]fs.DirEntry, error) {
	if g.data.Type != "dir" {
		return nil, &fs.PathError{Op: "readdir", Path: g.data.Path, Err: errors.New("not a directory")}
	}

	if len(g.data.Entries) == 0 {
		return nil, io.EOF
	}

	if n <= 0 || n > len(g.data.Entries) {
		n = len(g.data.Entries)
	}

	entries := g.data.Entries[:n]
	g.data.Entries = g.data.Entries[n:]

	dirEntries := make([]fs.DirEntry, len(entries))
	for i, entry := range entries {
		dirEntries[i] = entry
	}

	return dirEntries, nil
}

var (
	_ fs.ReadDirFile = (*gitHubEntry)(nil)
	_ fs.FileInfo    = (*gitHubEntry)(nil)
	_ fs.DirEntry    = (*gitHubEntry)(nil)
)
