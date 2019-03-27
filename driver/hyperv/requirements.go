package hyperv

import (
	"fmt"
	"strings"
	"errors"

	safeerr "code.cloudfoundry.org/cfdev/errors"
)

const (
	admin_role            = "[Security.Principal.WindowsBuiltInRole]::Administrator"
	current_user          = "New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())"
	hyperv_disabled_error = `You must first enable Hyper-V on your machine before you run CF Dev. Please use the following tutorial to enable this functionality on your machine

https://docs.microsoft.com/en-us/virtualization/hyper-v-on-windows/quick-start/enable-hyper-v`
)

func (d *HyperV) hasAdminPrivileged() error {
	command := fmt.Sprintf("(%s).IsInRole(%s)", current_user, admin_role)

	output, err := d.Powershell.Output(command)
	if err != nil {
		return fmt.Errorf("checking for admin privileges: %s", err)
	}

	if strings.Contains(strings.ToLower(output), "true") {
		return nil
	}

	return safeerr.SafeWrap(errors.New("You must run cf dev with an admin privileged powershell"), "Running without admin privileges")
}

func (d *HyperV) hypervEnabled() error {
	// The Microsoft-Hyper-V and the Microsoft-Hyper-V-Management-PowerShell are required.
	status, err := d.hypervStatus("Microsoft-Hyper-V")
	if err != nil {
		return err
	}

	if !strings.Contains(strings.ToLower(status), "enabled") {
		return safeerr.SafeWrap(errors.New(hyperv_disabled_error), "Microsoft-Hyper-V disabled")
	}

	status, err = d.hypervStatus("Microsoft-Hyper-V-Management-PowerShell")
	if err != nil {
		return err
	}

	if !strings.Contains(strings.ToLower(status), "enabled") {
		return safeerr.SafeWrap(errors.New(hyperv_disabled_error), "Microsoft-Hyper-V-Management-PowerShell disabled")
	}

	return nil
}

func (d *HyperV) hypervStatus(featureName string) (string, error) {
	command := fmt.Sprintf("(Get-WindowsOptionalFeature -FeatureName %s -Online).State", featureName)

	output, err := d.Powershell.Output(command)
	if err != nil {
		return "", fmt.Errorf("checking whether hyperv is enabled: %s", err)
	}

	return strings.TrimSpace(output), nil
}
