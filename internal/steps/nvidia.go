package steps

import (
	"os/exec"

	"github.com/paulo-amaral/dockup/internal/sysinfo"
)

// NVIDIA Container Toolkit replaced the deprecated nvidia-docker2. The repo
// setup below follows the official install guide for deb and rpm families:
// https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html
const nvidiaDeb = `set -e
curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg
curl -sL https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
  sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' \
  > /etc/apt/sources.list.d/nvidia-container-toolkit.list
apt-get update
apt-get install -y nvidia-container-toolkit
nvidia-ctk runtime configure --runtime=docker
systemctl restart docker
echo "NVIDIA Container Toolkit ready. Test: docker run --rm --gpus all ubuntu nvidia-smi"`

const nvidiaRpm = `set -e
curl -sL https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo \
  > /etc/yum.repos.d/nvidia-container-toolkit.repo
yum install -y nvidia-container-toolkit
nvidia-ctk runtime configure --runtime=docker
systemctl restart docker
echo "NVIDIA Container Toolkit ready. Test: docker run --rm --gpus all ubuntu nvidia-smi"`

func nvidiaStep(info sysinfo.Info) Step {
	desc := "GPU support for containers (replaces deprecated nvidia-docker2)"
	if !info.HasGPU {
		desc = "No NVIDIA GPU detected on this host; install only if you know better"
	}
	return Step{
		ID:        "nvidia",
		Title:     "Install NVIDIA Container Toolkit",
		Desc:      desc,
		NeedsRoot: true,
		Command: func(i sysinfo.Info) *exec.Cmd {
			if i.Family == "rpm" {
				return shell(nvidiaRpm)
			}
			return shell(nvidiaDeb)
		},
	}
}
