package secftd

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const maxArchiveMemberBytes int64 = 1 << 30

type namedReader struct {
	name string
	open func() (io.ReadCloser, error)
}

type archiveMemberReader struct {
	reader  io.ReadCloser
	archive *zip.ReadCloser
}

func (r *archiveMemberReader) Read(p []byte) (int, error) { return r.reader.Read(p) }
func (r *archiveMemberReader) Close() error {
	memberErr := r.reader.Close()
	archiveErr := r.archive.Close()
	if memberErr != nil {
		return memberErr
	}
	return archiveErr
}

func discoverInputs(root string) ([]namedReader, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	paths := []string{root}
	if info.IsDir() {
		paths = nil
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".zip" || ext == ".txt" || ext == ".csv" {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(paths)

	var readers []namedReader
	for _, path := range paths {
		if strings.EqualFold(filepath.Ext(path), ".zip") {
			archive, err := zip.OpenReader(path)
			if err != nil {
				return nil, fmt.Errorf("open %s: %w", path, err)
			}
			for _, member := range archive.File {
				if member.FileInfo().IsDir() {
					continue
				}
				clean := filepath.Clean(member.Name)
				if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
					archive.Close()
					return nil, fmt.Errorf("unsafe archive member %q", member.Name)
				}
				if member.UncompressedSize64 > uint64(maxArchiveMemberBytes) {
					archive.Close()
					return nil, fmt.Errorf("archive member %q exceeds size limit", member.Name)
				}
				ext := strings.ToLower(filepath.Ext(member.Name))
				if ext != ".txt" && ext != ".csv" {
					continue
				}
				archivePath := path
				memberName := member.Name
				readers = append(readers, namedReader{
					name: filepath.Base(path) + ":" + member.Name,
					open: func() (io.ReadCloser, error) {
						z, err := zip.OpenReader(archivePath)
						if err != nil {
							return nil, err
						}
						for _, candidate := range z.File {
							if candidate.Name != memberName {
								continue
							}
							r, err := candidate.Open()
							if err != nil {
								_ = z.Close()
								return nil, err
							}
							return &archiveMemberReader{reader: r, archive: z}, nil
						}
						_ = z.Close()
						return nil, fmt.Errorf("archive member %q disappeared", memberName)
					},
				})
			}
			if err := archive.Close(); err != nil {
				return nil, err
			}
			continue
		}
		p := path
		readers = append(readers, namedReader{
			name: filepath.Base(p),
			open: func() (io.ReadCloser, error) { return os.Open(p) },
		})
	}
	return readers, nil
}
