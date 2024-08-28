package pastiche

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path"
	"path/filepath"

	"github.com/erixzone/crux/pkg/clog"
)

// TarStreamer - walk a directory hierarchy, writing it to a grpc stream in tar format.
type TarStreamer struct {
	writer io.Writer
}

// NewTarStreamer - wr arg is for .SendTar() to output on.
func NewTarStreamer(wr io.Writer) (*TarStreamer, error) {
	return &TarStreamer{writer: wr}, nil
}

// SendDir - Send all file & directory data as a compressed, tar
// formatted stream of bytes.  rootPath is an absolute path the
// directory to be tarred.  The resultant tar stream will be relative
// to the (final) directory.
func (ts *TarStreamer) SendDir(rootPath string) (err error) {
	ll := clog.Log.With("focus", "TAR")
	ll.Log(nil, "SendDir root path %s", rootPath)
	zw := gzip.NewWriter(ts.writer)
	defer zw.Close()

	tw := tar.NewWriter(zw)
	defer tw.Close()

	dirPrefix := filepath.Dir(rootPath)

	return filepath.Walk(rootPath, func(curPath string, fi os.FileInfo, err error) error {

		if err != nil {
			ll.Log(nil, "WALK:  ERROR %s  ", err)
			return err
		}

		// Partially populated header for the file/dir
		// Preserves fi.ModTime()
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		//header.Name = curPath
		header.Name, err = filepath.Rel(dirPrefix, curPath)
		if err != nil {
			return err
		}
		ll.Log(nil, "WALK:  curPath [%s]  header [%s]", curPath, header.Name)

		if err := tw.WriteHeader(header); err != nil {

			return err
		}

		// Non-files send only header.  Will not follow symlinks.
		if !fi.Mode().IsRegular() {
			ll.Log(nil, "   WALK:  Non-regular [%s], no data copy", fi.Name())
			return nil
		}

		// It's a file. Copy data.
		f, err := os.Open(curPath)
		defer f.Close()
		if err != nil {
			return err
		}

		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		ll.Log(nil, "   WALK:  Regular [%s], data copied", fi.Name())

		return nil
	})

}

// Untar - During an operation, any error in creating a file or
// directory will halt the untar, and leave all files up to the error
// in place.  Formats supported: .tar, .tgz It is assumed that the tar
// has a single root dir, such as created with prefix option in git
// archive:
//    git archive --format tar.gz -o ../branch2.tar.gz --prefix branch2 master
// Returns the total bytes written, or an error.
func Untar(rdr io.Reader, destPath string, isZipped bool) (bytes uint64, rootPath string, err error) {
	log := clog.Log.With("focus", "TAR")
	var totalBytes uint64
	var tr *tar.Reader

	// We should be getting a zipped tar data from the provided reader
	if isZipped {
		zr, err := gzip.NewReader(rdr)
		if err != nil {
			return 0, "", err
		}
		tr = tar.NewReader(zr)
	} else {
		tr = tar.NewReader(rdr)
	}

	var tarRootDir string
	for {
		header, err := tr.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Error(err)
			return 0, "", err
		}

		pathName := path.Join(destPath, header.Name)
		//TODO: Do we want to preserve tar file's mod times?
		//      Set dirs and files to header.ModTime

		switch header.Typeflag {
		case tar.TypeDir:
			// TODO: This only handles tars with a single root
			// directory (tar prefix).  Works for git
			// tarballs, but will fail for tars with
			// multiple entries at their top level
			if tarRootDir == "" {
				tarRootDir = pathName
				log.Log(nil, "UNTAR:  pathName  %s", pathName)
			}

			err = os.MkdirAll(pathName, 0777)
			if err != nil {
				log.Error(err)
				return 0, "", err
			}
		case tar.TypeReg, tar.TypeRegA:
			w, err := os.Create(pathName)
			if err != nil {
				log.Error(err)
				return 0, "", err
			}

			n, err := io.Copy(w, tr)
			if err != nil {
				log.Error(err)
				return 0, "", err
			}
			w.Close()
			totalBytes += uint64(n)

		case tar.TypeXGlobalHeader:
			// Get the Git commit id

			// The archive/tar library does not parse the
			// PAX header well enough to provide the git
			// commit id put in by git archive at the
			// start of the record.  "52 comment=" is the
			// constant value git archive inserts there,
			// followed by the commit id, completing the
			// first 52 bytes in the record.  See
			// git/builtin/get-tar-commit-id.c

			// If we need to extract this bad enough we
			// can either fix tar/reader.go, or cheese out
			// by shelling out to get get-tar-commit-id

			// For now, we'll use whatever key the user
			// passes in, without verifying.

			// fmt.Printf(">>>>>PAX global header %#v \n",
			// *header)
		}
	}

	return totalBytes, tarRootDir, nil
}
