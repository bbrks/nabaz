package local

import (
	"errors"
	"time"

	git "github.com/nabaz-io/go-git.v4"
	"github.com/nabaz-io/go-git.v4/plumbing"
	"github.com/nabaz-io/go-git.v4/plumbing/object"
	"github.com/nabaz-io/go-git.v4/storage/memory"

	"github.com/nabaz-io/go-git.v4/plumbing/format/diff"
	"github.com/nabaz-io/nabaz/pkg/hypertest/scm/code"
)

// LocalGitHistory is history supplied by .git
type LocalGitHistory struct {
	// Path is the path to the local git repository.
	*git.Repository
	headCommitID string
	RootPath     string
}

// NewLocalGitHistory creates a new LocalGitRepo.
func NewLocalGitHistory(path string) (*LocalGitHistory, error) {
	git.GitDirName = ".git"
	originalDotGit, err := git.PlainOpenWithOptions(path, &git.PlainOpenOptions{DetectDotGit: true})
	if err != nil {
		return nil, err
	}

	wt, err := originalDotGit.Worktree()
	if err != nil {
		return nil, err
	}

	rootPath := wt.Filesystem.Root()
	if err != nil {
		return nil, err
	}

	git.GitDirName = ".nabazgit"

	gitRepo, err := git.Init(memory.NewStorage(), wt.Filesystem)
	switch err {
	case nil:
		// do nothing
	case git.ErrRepositoryAlreadyExists:
		gitRepo, err = git.Open(memory.NewStorage(), wt.Filesystem)
		if err != nil {
			return nil, err
		}
	default:
		return nil, err
	}
	localRepo := &LocalGitHistory{
		Repository: gitRepo,
		RootPath:   rootPath,
	}

	return localRepo, nil
}

func (l *LocalGitHistory) SaveAllFiles() error {
	wt, err := l.Worktree()
	if err != nil {
		return err
	}

	_, err = wt.Add(".")
	if err != nil {
		return err
	}

	_, err = wt.Commit("A regular Nabaz commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Auto",
			Email: "hello@nabaz.io",
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}
	l.updateHeadCommitID()

	return nil
}

// HeadCommitID returns the commit ID of the HEAD of the repository.
func (l *LocalGitHistory) HeadCommitID() (string, error) {
	if l.headCommitID == "" {
		head, err := l.Repository.Head()
		if err != nil {
			return "", errors.New("could not get HEAD commit")
		}
		l.headCommitID = head.Hash().String()
	}
	return l.headCommitID, nil
}

func (l *LocalGitHistory) updateHeadCommitID() {
	head, err := l.Repository.Head()
	if err != nil {
		panic("failed to reload head commit id")
	}
	l.headCommitID = head.Hash().String()
}

func (r *LocalGitHistory) CommitParents(commitID string) ([]string, error) {
	commit, err := r.Repository.CommitObject(plumbing.NewHash(commitID))
	if err != nil {
		return nil, err
	}

	parents := make([]string, 0)
	for _, parent := range commit.ParentHashes {
		parents = append(parents, parent.String())

	}
	return parents, nil
}

func (r *LocalGitHistory) GetFileContent(path string, commitID string) ([]byte, error) {
	hash := plumbing.NewHash(commitID)
	commit, err := r.Repository.CommitObject(hash)
	if err != nil {
		return nil, err
	}

	file, err := commit.File(path)
	if err != nil {
		return nil, err
	}

	content, err := file.Contents()
	if err != nil {
		return nil, err
	}

	return []byte(content), nil
}

func (l *LocalGitHistory) Diff(currentCommitID string, olderCommitID string) ([]code.FileDiff, error) {
	currentCommit, err := l.Repository.CommitObject(plumbing.NewHash(currentCommitID))
	if err != nil {
		return nil, err
	}

	olderCommit, err := l.Repository.CommitObject(plumbing.NewHash(olderCommitID))
	if err != nil {
		return nil, err
	}

	patch, err := currentCommit.Patch(olderCommit)
	if err != nil {
		return nil, err
	}

	patches := patch.FilePatches()

	fileDiffs := make([]code.FileDiff, len(patches))
	for i, patch := range patches {
		isBinary := patch.IsBinary()
		from, to := patch.Files()
		status := fileChangeNature(from, to)

		path, prevPath := "", ""
		if status != code.REMOVED {
			path = to.Path()
		}
		if status != code.ADDED {
			prevPath = from.Path()
		}

		fileDiffs[i] = code.FileDiff{
			Path:         path,
			Patch:        patch.Chunks(),
			IsBinary:     isBinary,
			Status:       status,
			PreviousPath: prevPath,
		}
	}
	return fileDiffs, nil

}

// fileChangeNature figures out whats the nature of the change, i.e. if the file was added, deleted or modified.
func fileChangeNature(from diff.File, to diff.File) code.FileStatus {
	if from == nil {
		return code.ADDED
	}
	if to == nil {
		return code.REMOVED
	}

	if from.Path() != to.Path() {
		return code.RENAMED
	}

	return code.MODIFIED
}

func (l *LocalGitHistory) HEAD() string {
	commitID, _ := l.HeadCommitID()
	return commitID
}
