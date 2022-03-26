package memefs

import (
	"context"
	"memefsGo/helper"
	"memefsGo/model"
	"sync"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

type MemeFSNode struct {
	fs.Inode
	fuse.Attr

	memeFS *MemeFS
	data   []byte
}

type MemeFS struct {
	fuseServer *fuse.Server

	config model.MemeFSConfig
	mu     sync.RWMutex
	memes  []model.Post
}

func (m *MemeFS) getCurOwner() fuse.Owner {
	uid, gid := helper.GetCurUIDAndGID()
	return fuse.Owner{
		Uid: uid,
		Gid: gid,
	}
}

func (m *MemeFSNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	m.memeFS.mu.RLock()
	defer m.memeFS.mu.RUnlock()

	ttl := time.Second
	owner := m.memeFS.getCurOwner()
	now := uint64(time.Now().Unix())
	for ind, post := range m.memeFS.memes {
		if name != post.Title {
			continue
		}

		ino := uint64(ind + 2)

		attr := fuse.Attr{
			Ino:   ino,
			Size:  post.Size,
			Mode:  fuse.S_IFREG,
			Owner: owner,
			Atime: now,
			Mtime: now,
			Ctime: now,
			// Crtime_: uint64(now.Unix()),
		}

		inode := m.NewInode(ctx, &MemeFSNode{Attr: attr, memeFS: m.memeFS}, fs.StableAttr{Mode: fuse.S_IFREG, Ino: ino})
		out.SetEntryTimeout(ttl)
		out.SetAttrTimeout(ttl)
		out.NodeId = ino
		out.Attr = attr

		return inode, fs.OK
	}

	return nil, syscall.ENOENT
}

func (m *MemeFSNode) Readdir(ctx context.Context) (fs.DirStream, syscall.Errno) {
	entries := []fuse.DirEntry{{
		Name: ".",
		Mode: fuse.S_IFDIR,
	}, {
		Name: "..",
		Mode: fuse.S_IFDIR,
	}}

	for ind, post := range m.memeFS.getMemes() {
		entries = append(entries, fuse.DirEntry{
			Name: post.Title,
			Mode: fuse.S_IFREG,
			Ino:  uint64(ind + 2),
		})
	}

	return fs.NewListDirStream(entries), fs.OK
}

func (m *MemeFSNode) Getattr(ctx context.Context, f fs.FileHandle, out *fuse.AttrOut) syscall.Errno {

	out.AttrValid = 1
	out.Attr = m.Attr

	return fs.OK
}

func (m *MemeFSNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	data, ok := m.memeFS.getMeme(m.Attr.Ino)
	if !ok {
		return nil, 0, syscall.ENOENT
	}

	m.data = data

	// We don't return a filehandle since we don't really need
	// one. The file content is immutable, so hint the kernel to
	// cache the data.
	return nil, fuse.FOPEN_KEEP_CACHE, fs.OK
}

func (m *MemeFSNode) Read(ctx context.Context, f fs.FileHandle, dest []byte, off int64) (fuse.ReadResult, syscall.Errno) {
	if m.data == nil || len(m.data) == 0 {
		return nil, syscall.ENOENT
	}

	return fuse.ReadResultData(m.data[off:helper.Min(int(off)+len(dest), len(m.data))]), fs.OK
}

func (m *MemeFSNode) Release(ctx context.Context, f fs.FileHandle) syscall.Errno {
	m.data = nil
	return fs.OK
}

var _ = (fs.NodeLookuper)((*MemeFSNode)(nil))
var _ = (fs.NodeGetattrer)((*MemeFSNode)(nil))
var _ = (fs.NodeReaddirer)((*MemeFSNode)(nil))
var _ = (fs.NodeOpener)((*MemeFSNode)(nil))
var _ = (fs.NodeReader)((*MemeFSNode)(nil))
var _ = (fs.NodeReleaser)((*MemeFSNode)(nil))

func (m *MemeFS) updateMemes(posts []model.Post) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.memes = posts
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

		url = m.memes[ino-2].Url
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
	owner := m.getCurOwner()
	now := uint64(time.Now().Unix())
	srv, err := fs.Mount(m.config.Mountpoint, &MemeFSNode{
		memeFS: m,
		Attr: fuse.Attr{
			Ino:   1,
			Mode:  fuse.S_IFDIR,
			Owner: owner,
			Atime: now,
			Mtime: now,
			Ctime: now,

			// Crtime_: uint64(now.Unix()),
		},
	}, &fs.Options{
		MountOptions: fuse.MountOptions{
			AllowOther: true,
			FsName:     "MemeFS",
			Name:       "meme",
			Debug:      m.config.Debug,
			// Options:    []string{"ro"},
		}},
	)

	defer m.Unmount()

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

func (m *MemeFS) Unmount() {
	if m.fuseServer != nil {
		m.fuseServer.Unmount()
	}
}

func New(config model.MemeFSConfig) *MemeFS {
	return &MemeFS{
		fuseServer: nil,
		config:     config,
		memes:      []model.Post{},
	}
}
