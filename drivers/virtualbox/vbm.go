package virtualbox

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/docker/machine/libmachine/log"
)

var (
	reColonLine       = regexp.MustCompile(`(.+):\s+(.*)`)
	reEqualLine       = regexp.MustCompile(`(.+)=(.*)`)
	reEqualQuoteLine  = regexp.MustCompile(`"(.+)"="(.*)"`)
	reMachineNotFound = regexp.MustCompile(`Could not find a registered machine named '(.+)'`)

	ErrMachineNotExist = errors.New("machine does not exist")
	ErrVBMNotFound     = errors.New("VBoxManage not found. Make sure VirtualBox is installed and VBoxManage is in the path")

	vboxManageCmd = detectVBoxManageCmd()
)

// VBoxManager defines the interface to communicate to VirtualBox.
type VBoxManager interface {
	vbm(args ...string) error

	vbmOut(args ...string) (string, error)

	vbmOutErr(args ...string) (string, string, error)
}

// VBoxCmdManager communicates with VirtualBox through the commandline using `VBoxManage`.
type VBoxCmdManager struct{}

func (v *VBoxCmdManager) vbm(args ...string) error {
	_, _, err := v.vbmOutErr(args...)
	return err
}

func (v *VBoxCmdManager) vbmOut(args ...string) (string, error) {
	stdout, _, err := v.vbmOutErr(args...)
	return stdout, err
}

func (v *VBoxCmdManager) vbmOutErr(args ...string) (string, string, error) {
	cmd := exec.Command(vboxManageCmd, args...)
	log.Debugf("COMMAND: %v %v", vboxManageCmd, strings.Join(args, " "))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	stderrStr := stderr.String()
	if len(args) > 0 {
		log.Debugf("STDOUT:\n{\n%v}", stdout.String())
		log.Debugf("STDERR:\n{\n%v}", stderrStr)
	}

	if err != nil {
		if ee, ok := err.(*exec.Error); ok && ee.Err == exec.ErrNotFound {
			err = ErrVBMNotFound
		}
	}

	if err == nil || strings.HasPrefix(err.Error(), "exit status ") {
		// VBoxManage will sometimes not set the return code, but has a fatal error
		// such as VBoxManage.exe: error: VT-x is not available. (VERR_VMX_NO_VMX)
		if strings.Contains(stderrStr, "error:") {
			err = fmt.Errorf("%v %v failed:\n%v", vboxManageCmd, strings.Join(args, " "), stderrStr)
		}
	}

	return stdout.String(), stderrStr, err
}

func checkVBoxManageVersion(version string) error {
	if !strings.HasPrefix(version, "5.") && !strings.HasPrefix(version, "4.") {
		return fmt.Errorf("We support Virtualbox starting with version 4. Your VirtualBox install is %q. Please upgrade at https://www.virtualbox.org", version)
	}

	return nil
}
