// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strings"

	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
	"aahframework.org/vfs.v0"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// type AntiCSRFField and methods
//______________________________________________________________________________

// AntiCSRFField is used to insert Anti-CSRF HTML field dynamically
// while parsing templates on view engine.
type AntiCSRFField struct {
	engineName string
	field      string
	leftDelim  string
	rightDelim string
	inserter   *strings.Replacer
	vfs        *vfs.VFS
}

// NewAntiCSRFField method creates new instance of Anti-CSRF HTML field
// parser.
func NewAntiCSRFField(engineName, leftDelim, rightDelim string) *AntiCSRFField {
	return NewAntiCSRFFieldWithVFS(nil, engineName, leftDelim, rightDelim)
}

// NewAntiCSRFFieldWithVFS method creates new instance of Anti-CSRF HTML field
// parser with given VFS instance.
func NewAntiCSRFFieldWithVFS(fs *vfs.VFS, engineName, leftDelim, rightDelim string) *AntiCSRFField {
	csft := &AntiCSRFField{vfs: fs, engineName: engineName, leftDelim: leftDelim, rightDelim: rightDelim}

	csft.field = fmt.Sprintf(`	<input type="hidden" name="anti_csrf_token" value="%s anticsrftoken . %s">
     	</form>`, csft.leftDelim, csft.rightDelim)
	csft.inserter = strings.NewReplacer("</form>", csft.field)

	return csft
}

// InsertOnFile method inserts the Anti-CSRF HTML field for given HTML file and
// writes a processed file into temp directory then return the new file path.
func (ft *AntiCSRFField) InsertOnFiles(files ...string) []string {
	var ofiles []string

	for _, f := range files {
		fpath, err := ft.InsertOnFile(f)
		if err != nil {
			log.Errorf("anitcsrffield: unable to insert Anti-CSRF field for file: %s", f)
			ofiles = append(ofiles, f)
			continue
		}
		ofiles = append(ofiles, fpath)
	}

	return ofiles
}

// InsertOnFile method inserts the Anti-CSRF HTML filed for given HTML file and
// writes a processed file into temp directory then return the new file path.
func (ft *AntiCSRFField) InsertOnFile(file string) (string, error) {
	tmpDir, _ := ioutil.TempDir("", ft.engineName+"_anti_csrf")

	fileBytes, err := vfs.ReadFile(ft.vfs, file)
	if err != nil {
		return "", err
	}

	fileStr := string(fileBytes)
	f := StripPathPrefixAt(file, "views")
	fpath := filepath.Join(tmpDir, f)
	log.Tracef("Inserting Anti-CSRF field for file: %s", path.Join("views", f))
	fileStr = ft.InsertOnString(fileStr)
	if err = ess.MkDirAll(filepath.Dir(fpath), 0755); err != nil {
		return "", err
	}

	if err = ioutil.WriteFile(fpath, []byte(fileStr), 0755); err != nil {
		return "", err
	}

	return fpath, nil
}

// InsertOnString method inserts the Anti-CSRF HTML field on
// given HTML string and returns the processed HTML string.
func (ft *AntiCSRFField) InsertOnString(str string) string {
	return ft.inserter.Replace(str)
}
