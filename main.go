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
)

// Not identical to proto.Merge
func merge(manifest *pb.JpkgFile, lock *pb.JpkgFile) *pb.JpkgFile {
	ret := &pb.JpkgFile{
		Directory: manifest.GetDirectory(),
	}
	for _, mr := range manifest.GetRepositories() {
		r := mr
		for _, lr := range lock.GetRepositories() {
			if mr.GetUrl() == lr.GetUrl() {
				r = lr
			}
		}
		ret.Repositories = append(ret.Repositories, r)
	}
	return ret
}

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
	log.Printf("Installing packages...")
	manifest, err := readJpkgFile(*jpkgManifestFilePath)
	if err != nil {
		log.Fatalf("Failed to parse manifest file: %s, %v", *jpkgManifestFilePath, err)
	}
	p := manifest
	if fileAvailable(*jpkgLockFilePath) {
		lock, err := readJpkgFile(*jpkgLockFilePath)
		if err != nil {
			log.Fatalf("Failed to parse lock file: %s, %v", *jpkgLockFilePath, err)
		}
		p = merge(manifest, lock)
	}
	for _, r := range p.GetRepositories() {
		dir, err := repoDir(r)
		if err != nil {
			log.Fatalf("Failed to get directory to clone: %s, %v", r.GetUrl(), err)
		}
		r.Directory = dir
		dirToInstall := path.Join(p.GetDirectory(), dir)
		log.Printf("Cloning %s to %s", r.GetUrl(), dirToInstall)
		hash, err := clone(dirToInstall, r.GetUrl())
		if err != nil {
			log.Fatalf("Failed to clone: %s, %v", r.GetUrl(), err)
		}
		if r.GetHash() == "" {
			r.Hash = hash
		} else {
			if err := checkout(dirToInstall, r.GetHash()); err != nil {
				log.Fatalf("Failed to checkout: %s, %v", r.GetUrl(), err)
			}
			log.Printf("Checked out %s to %s", r.GetUrl(), r.GetHash())
		}
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
	}); err != nil {
		return "", err
	}
	ref, err := r.Head()
	if err != nil {
		return "", err
	}
	return ref.Hash().String(), nil
}

func update() {
	log.Printf("Updating packages...")
	manifest, err := readJpkgFile(*jpkgManifestFilePath)
	if err != nil {
		log.Fatalf("Failed to read file %s, %v", *jpkgManifestFilePath, err)
	}
	lock, err := readJpkgFile(*jpkgLockFilePath)
	if err != nil {
		log.Fatalf("Lock file is required for update. Failed to read file %s, %v.", *jpkgManifestFilePath, err)
	}
	newLock := &pb.JpkgFile{
		Directory: manifest.GetDirectory(),
	}
	for _, mr := range manifest.GetRepositories() {
		if mr.GetHash() != "" {
			log.Printf("Skipping repository=%s, this is locked at %s in manifest", mr.GetUrl(), mr.GetHash())
			continue
		}
		target := mr
		for _, lr := range lock.GetRepositories() {
			if lr.GetUrl() == mr.GetUrl() {
				target = lr
			}
		}
		installedDir := path.Join(manifest.GetDirectory(), target.GetDirectory())
		log.Printf("Pulling %s %s", target.GetUrl(), installedDir)
		hash, err := pull(installedDir)
		if err != nil {
			if errors.Is(err, git.NoErrAlreadyUpToDate) {
				log.Printf("Repository=%s was already updated as hash=%s", target.GetUrl(), target.GetHash())
			} else {
				log.Fatalf("Failed to pull repository %s, %v", target.GetUrl(), err)
			}
		} else {
			target.Hash = hash
			log.Printf("Updated repository=%s to hash=%s", target.GetUrl(), target.GetHash())
		}
		newLock.Repositories = append(newLock.Repositories, target)
	}
	if err := writeJpkgLockFile(*jpkgLockFilePath, newLock); err != nil {
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
