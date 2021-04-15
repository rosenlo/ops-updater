package g

import (
	"ops-updater/file"
)

var SelfDir string

func InitGlobalVariables() {
	SelfDir = file.SelfDir()
}
