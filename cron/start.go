package cron

import (
	"fmt"
	"log"
	"os/exec"
	"path"
	"strings"
	"time"

	"ops-common/model"
	"ops-common/utils"
	"ops-updater/file"
	"ops-updater/g"
)

func StartDesiredAgent(da *model.DesiredAgent) {
	if err := InsureDesiredAgentDirExists(da); err != nil {
		return
	}

	restart, err := InsureNewVersionFiles(da)
	if err != nil {
		return
	}

	if err := Untar(da, restart); err != nil {
		return
	}

	if err := StopAgentOf(da.Name, da.Version, restart); err != nil {
		return
	}

	if err := ControlStartIn(da.AgentVersionDir); err != nil {
		return
	}

	file.WriteString(path.Join(da.AgentDir, ".version"), da.Version)
}

func Untar(da *model.DesiredAgent, restart bool) error {
	if !restart {
		return nil
	}
	cmd := exec.Command("tar", "zxf", da.TarballFilename)
	cmd.Dir = da.AgentVersionDir
	err := cmd.Run()
	if g.Config().Debug {
		log.Println("untar...")
	}
	if err != nil {
		log.Println("tar zxf", da.TarballFilename, "fail", err)
		return err
	}

	return nil
}

func ControlStartIn(workdir string) error {
	out, err := ControlStatus(workdir)
	if err == nil && strings.Contains(out, "started") {
		return nil
	}

	_, err = ControlStart(workdir)
	if g.Config().Debug {
		log.Println("start agent...")
	}
	if err != nil {
		return err
	}

	time.Sleep(time.Second * 3)

	out, err = ControlStatus(workdir)
	if err == nil && strings.Contains(out, "started") {
		return nil
	}

	return err
}

func InsureNewVersionFiles(da *model.DesiredAgent) (bool, error) {
	downloadMd5Cmd := exec.Command("wget", "-q", "-T 5", "-t 1", da.Md5Url, "-O", da.Md5Filename)
	downloadMd5Cmd.Dir = da.AgentVersionDir
	err := downloadMd5Cmd.Run()
	if g.Config().Debug {
		log.Println("download md5 file")
	}
	if err != nil {
		log.Println("wget -q -T 5 -t 1", da.Md5Url, "-O", da.Md5Filename, "fail", err)
		return false, err
	}

	if FilesReady(da) {
		return false, nil
	}

	downloadTarballCmd := exec.Command("wget", "-q", "-T 5", "-t 1", da.TarballUrl, "-O", da.TarballFilename)
	downloadTarballCmd.Dir = da.AgentVersionDir
	err = downloadTarballCmd.Run()
	if g.Config().Debug {
		log.Println("download tarball")
	}
	if err != nil {
		log.Println("wget -q -T 5 -t 1", da.TarballUrl, "-O", da.TarballFilename, "fail", err)
		return false, err
	}

	if utils.Md5sumCheck(da.AgentVersionDir, da.Md5Filename) {
		return true, nil
	} else {
		return false, fmt.Errorf("md5sum -c fail")
	}
}

func FilesReady(da *model.DesiredAgent) bool {
	if !file.IsExist(da.Md5Filepath) {
		return false
	}

	if !file.IsExist(da.TarballFilepath) {
		return false
	}

	if !file.IsExist(da.ControlFilepath) {
		return false
	}

	return utils.Md5sumCheck(da.AgentVersionDir, da.Md5Filename)
}

func InsureDesiredAgentDirExists(da *model.DesiredAgent) error {
	err := file.InsureDir(da.AgentDir)
	if err != nil {
		log.Println("insure dir", da.AgentDir, "fail", err)
		return err
	}

	err = file.InsureDir(da.AgentVersionDir)
	if err != nil {
		log.Println("insure dir", da.AgentVersionDir, "fail", err)
	}
	return err
}
