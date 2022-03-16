package memefs

import (
	"context"
	"memefsGo/model"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type MemeFSRoot struct {
	fs.Inode

	memeFS *MemeFS
}

type MemeFile struct {
	fs.Inode
	fuse.Attr

	memeFS *MemeFS
}

type MemeFS struct {
	fuseServer *fuse.Server

	config model.MemeFSConfig
	mu     sync.RWMutex
	memes  []model.Post
}

var baseEntries = []fuse.DirEntry{{
	Name: ".",
	Mode: fuse.S_IFDIR,
	Ino:  1,
}, {
	Name: "..",
	Mode: fuse.S_IFDIR,
	Ino:  2,
}}

func (m *MemeFSRoot) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	m.memeFS.mu.RLock()
	defer m.memeFS.mu.RUnlock()

	for ind, post := range m.memeFS.memes {
		if name != post.Title {
			continue
		}

		ttl := time.Second
		ino := uint64(ind + 2)

		attr := fuse.Attr{
			Ino:  ino,
			Size: post.Size,
			Mode: fuse.S_IFREG,
		}

		inode := m.NewInode(ctx, &MemeFile{Attr: attr, memeFS: m.memeFS}, fs.StableAttr{Mode: fuse.S_IFREG, Ino: ino})
		out.SetEntryTimeout(ttl)
		out.SetAttrTimeout(ttl)
		out.NodeId = ino
		out.Attr = attr

		return inode, fs.OK
	}

	return nil, syscall.ENOENT
}

func (m *MemeFSRoot) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {

	out.AttrValid = 1
	out.Attr = fuse.Attr{
		Ino:  1,
		Mode: fuse.S_IFDIR,
	}

	return fs.OK
}

func (m *MemeFSRoot) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	entries := []fuse.DirEntry{}
	copy(entries, baseEntries)

	for ind, post := range m.memeFS.getMemes() {
		entries = append(entries, fuse.DirEntry{
			Name: post.Title,
			Mode: fuse.S_IFREG,
			Ino:  uint64(ind + 2),
		})
	}

	return fs.NewListDirStream(entries), fs.OK
}

func (m *MemeFile) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {

	out.AttrValid = 1
	out.Attr = m.Attr

	return fs.OK
}

func (m *MemeFile) Read(ctx context.Context, f fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	data, ok := m.memeFS.getMeme(m.Attr.Ino)
	if !ok {
		return nil, syscall.ENOENT
	}

	return fuse.ReadResultData(data), fs.OK
}

var _ = (fs.NodeLookuper)((*MemeFSRoot)(nil))
var _ = (fs.NodeGetattrer)((*MemeFSRoot)(nil))
var _ = (fs.NodeReaddirer)((*MemeFSRoot)(nil))

var _ = (fs.NodeGetattrer)((*MemeFile)(nil))
var _ = (fs.NodeReader)((*MemeFile)(nil))

func (m *MemeFS) updateMemes(posts []model.Post) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memes = append(m.memes, posts...)
}

func (m *MemeFS) getMemes() []model.Post {
	m.mu.RLock()
	defer m.mu.RUnlock()

	posts := make([]model.Post, len(m.memes))
	copy(posts, m.memes)

	return posts
}

func (m *MemeFS) getMeme(ino uint64) ([]byte, bool) {
	url := ""
	{
		m.mu.RLock()
		defer m.mu.RUnlock()

		url = m.memes[ino-3].Url
	}

	return fetchMeme(url)
}

func (m *MemeFS) startFetching(ctx context.Context) {
	m.updateMemes(fetchPosts(&m.config))

	ticker := time.NewTicker(time.Duration(m.config.RefreshSecs) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.updateMemes(fetchPosts(&m.config))
		}
	}
}

func (m *MemeFS) Mount() error {
	srv, err := fs.Mount(m.config.Mountpoint, &MemeFSRoot{
		memeFS: m,
	}, &fs.Options{
		MountOptions: fuse.MountOptions{
			AllowOther: true,
			FsName:     "MemeFS",
			Name:       "meme",
			// Options:    []string{"big_writes"},
		}},
	)

	if err != nil {
		return err
	}

	m.fuseServer = srv

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go m.startFetching(ctx)
	m.fuseServer.Wait()

	return nil
}

func New(config model.MemeFSConfig) *MemeFS {
	return &MemeFS{
		fuseServer: nil,
		config:     config,
		memes:      []model.Post{},
	}
}
