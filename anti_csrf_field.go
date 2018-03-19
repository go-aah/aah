// Copyright (c) Jeevanandam M. (https://github.com/jeevatkm)
// go-aah/view source code and usage is governed by a MIT style
// license that can be found in the LICENSE file.

package view

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"aahframework.org/essentials.v0"
	"aahframework.org/log.v0"
)

//‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾‾
// type AntiCSRFField and methods
//________________________________________

// AntiCSRFField is used to insert Anti-CSRF HTML field dynamically
// while parsing templates on view engine.
type AntiCSRFField struct {
	engineName string
	field      string
	inserter   *strings.Replacer
	leftDelim  string
	rightDelim string
}

// NewAntiCSRFField method creates new instance of Anti-CSRF HTML field
// parser.
func NewAntiCSRFField(engineName, leftDelim, rightDelim string) *AntiCSRFField {
	csft := &AntiCSRFField{engineName: engineName, leftDelim: leftDelim, rightDelim: rightDelim}

	csft.field = fmt.Sprintf(`	<input type="hidden" name="anti_csrf_token" value="%s anitcsrftoken . %s">
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

	fileBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return "", err
	}

	fileStr := string(fileBytes)
	fpath := filepath.Join(tmpDir, StripPathPrefixAt(file, "views"))
	if strings.Contains(fileStr, "</form>") {
		log.Tracef("Inserting Anti-CSRF field for file: %s", file)
		fileStr = ft.InsertOnString(fileStr)
		if err = ess.MkDirAll(filepath.Dir(fpath), 0755); err != nil {
			return "", err
		}

		if err = ioutil.WriteFile(fpath, []byte(fileStr), 0755); err != nil {
			return "", err
		}

		return fpath, nil
	}

	return file, nil
}

// InsertOnString method inserts the Anti-CSRF HTML field on
// given HTML string and returns the processed HTML string.
func (ft *AntiCSRFField) InsertOnString(str string) string {
	return ft.inserter.Replace(str)
}
