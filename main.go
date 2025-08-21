package main

import (
	"errors"
	"flag"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	pb "github.com/jyane/jpkg/proto"
)

var (
	mode                 = flag.String("mode", "", "Mode install|update")
	jpkgManifestFilePath = flag.String("jpkg-manifest-file", "jpkg-manifest.txtpb", "File path to jpkg manifest file")
	jpkgLockFilePath     = flag.String("jpkg-lock-file", "jpkg-lock.txtpb", "File path to jpkg lock file")
	baseDir              = flag.String("base-dir", "repos/", "Directory to install repositories")
)

func repoDir(r *pb.Repository) (string, error) {
	if r.GetDirectory() != "" {
		return r.GetDirectory(), nil
	}
	u, err := url.Parse(r.GetUrl())
	if err != nil {
		return "", err
	}
	repoPath := strings.Trim(u.Path, "/")
	name := path.Base(repoPath)
	name = strings.TrimSuffix(name, ".git")
	return name, nil
}

func clone(dir string, repoUrl string) (string, error) {
	repo, err := git.PlainClone(dir, &git.CloneOptions{
		URL:      repoUrl,
		Progress: os.Stdout,
	})
	if err != nil {
		return "", err
	}
	ref, err := repo.Head()
	if err != nil {
		return "", err
	}
	return ref.Hash().String(), nil
}

func checkout(dir string, hash string) error {
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return err
	}
	w, err := repo.Worktree()
	if err != nil {
		return err
	}
	if err := w.Checkout(&git.CheckoutOptions{Hash: plumbing.NewHash(hash)}); err != nil {
		return err
	}
	return nil
}

func install() {
	path := resolveJpkgFile(*jpkgManifestFilePath, *jpkgLockFilePath)
	log.Printf("Installing packages from %s", path)
	p, err := readJpkgFile(path)
	if err != nil {
		log.Fatalf("Failed to parse package file: %s, %v", path, err)
	}
	for _, r := range p.GetRepositories() {
		dir, err := repoDir(r)
		if err != nil {
			log.Fatalf("Failed to get directory to clone: %s, %v", r.GetUrl(), err)
		}
		log.Printf("Cloning %s to %s", r.GetUrl(), *baseDir+dir)
		hash, err := clone(*baseDir+dir, r.GetUrl())
		if err != nil {
			log.Fatalf("Failed to clone: %s, %v", r.GetUrl(), err)
		}
		hashToCheckout := ""
		if r.GetHash() == "" {
			hashToCheckout = hash
		} else {
			hashToCheckout = r.GetHash()
		}
		if err := checkout(*baseDir+dir, hashToCheckout); err != nil {
			log.Fatalf("Failed to checkout: %s, %v", r.GetUrl(), err)
		}
		log.Printf("Checked out %s to %s", r.GetUrl(), hashToCheckout)
		r.Directory = dir
		r.Hash = hashToCheckout
	}
	if err := writeJpkgLockFile(*jpkgLockFilePath, p); err != nil {
		log.Fatalf("Failed to save lock file: %s, %v", *jpkgLockFilePath, err)
	}
}

func pull(path string) (string, error) {
	r, err := git.PlainOpen(path)
	if err != nil {
		return "", err
	}
	w, err := r.Worktree()
	if err != nil {
		return "", err
	}
	if err := w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Progress:   os.Stdout,
	}); !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return "", err
	}
	ref, err := r.Head()
	if err != nil {
		return "", err
	}
	return ref.Hash().String(), nil
}

func update() {
	p, err := readJpkgFile(*jpkgManifestFilePath)
	if err != nil {
		log.Fatalf("Failed to read file %s, %v", *jpkgManifestFilePath, err)
	}
	for _, r := range p.GetRepositories() {
		if r.GetHash() != "" {
			log.Printf("Skipping as the repository %s is locked at %s", r.GetUrl(), r.GetHash())
			continue
		}
		dir, err := repoDir(r)
		if err != nil {
			log.Fatalf("Failed to get directory to update: %s, %v", r.GetUrl(), err)
		}
		log.Printf("Pulling %s %s", r.GetUrl(), *baseDir+dir)
		hash, err := pull(*baseDir + dir)
		if err != nil {
			log.Fatalf("Failed to pull repository %s, %v", r.GetUrl(), err)
		}
		log.Printf("Updated or already updated repository=%s, hash=%s", r.GetUrl(), hash)
		r.Directory = dir
		r.Hash = hash
	}
	if err := writeJpkgLockFile(*jpkgLockFilePath, p); err != nil {
		log.Fatalf("Failed to save lock file %s, %v", *jpkgLockFilePath, err)
	}
}

func main() {
	flag.Parse()
	switch *mode {
	case "install":
		install()
	case "update":
		update()
	default:
		flag.Usage()
	}
}
