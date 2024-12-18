// main.go
package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

const (
    // Progress Stages
    StageInitializing     = "initializing"
    StageCreatingDisk     = "creating_disk"
    StagePreparingCloudInit = "preparing_cloud_init"
    StageStartingQEMU     = "starting_qemu"
    StageConfigVNC        = "configuring_vnc"
    StageInstallingTemplate = "installing_template" // New stage
    StageCompleted        = "completed"
    StageFailed          = "failed"

    // Ubuntu Images
    UBUNTU_22_04_IMAGE_URL = "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img"
    UBUNTU_20_04_IMAGE_URL = "https://cloud-images.ubuntu.com/focal/current/focal-server-cloudimg-amd64.img"
    UBUNTU_24_04_IMAGE_URL = "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
    
    // Debian Images
    DEBIAN_11_IMAGE_URL = "https://cloud.debian.org/images/cloud/bullseye/latest/debian-11-generic-amd64.qcow2"
    DEBIAN_12_IMAGE_URL = "https://os-cdn.virtfusion.net/debian/debian-12-x86_64.qcow2"
    
    // Fedora Images
    FEDORA_38_IMAGE_URL = "https://download.fedoraproject.org/pub/fedora/linux/releases/38/Cloud/x86_64/images/Fedora-Cloud-Base-38-1.6.x86_64.qcow2"
    FEDORA_40_IMAGE_URL = "https://os-cdn.virtfusion.net/fedora/fedora-40-x86_64-virtfusion.qcow2"
    
    // RHEL-based Images
    ALMA_8_IMAGE_URL = "https://repo.almalinux.org/almalinux/8/cloud/x86_64/images/AlmaLinux-8-GenericCloud-latest.x86_64.qcow2"
    ALMA_9_IMAGE_URL = "https://os-cdn.virtfusion.net/alma/almalinux-9-x86_64.qcow2"
    ROCKY_8_IMAGE_URL = "https://os.virtfusion.net/images/rocky-linux-8-minimal-x86_64.qcow2"
    ROCKY_9_IMAGE_URL = "https://os-cdn.virtfusion.net/rocky/rocky-linux-9-x86_64.qcow2"
    
    // CentOS Images
    CENTOS_7_IMAGE_URL = "https://os.virtfusion.net/images/centos-7-minimal-x86_64.qcow2"
    CENTOS_9_IMAGE_URL = "https://os-cdn.virtfusion.net/centos/centos-stream-9-x86_64.qcow2"
    
    // Other constants
    BASE_DIR        = "/var/lib/vps-service/base"
    VPS_LIFETIME    = 15 * time.Minute
    RAM_SIZE        = 4096  // 4GB
    DISK_SIZE       = 50    // 50GB
    DOWNLOAD_SPEED  = 50    // 50Mbps
    UPLOAD_SPEED    = 15    // 15Mbps
    SSH_PORT_START  = 2200  // Starting port for SSH forwarding
    StatusRunning    = "running"
    StatusStopped    = "stopped"
    StatusStarting   = "starting"
    StatusStopping   = "stopping"
    StatusRestarting = "restarting"
    
)

var SUPPORTED_IMAGES = map[string]string{
    // Ubuntu
    "ubuntu-22.04": UBUNTU_22_04_IMAGE_URL,
    "ubuntu-20.04": UBUNTU_20_04_IMAGE_URL,
    "ubuntu-24.04": UBUNTU_24_04_IMAGE_URL,
    
    // Debian
    "debian-11": DEBIAN_11_IMAGE_URL,
    "debian-12": DEBIAN_12_IMAGE_URL,
    
    // Fedora
    "fedora-38": FEDORA_38_IMAGE_URL,
    "fedora-40": FEDORA_40_IMAGE_URL,
    
    // RHEL-based
    "almalinux-8": ALMA_8_IMAGE_URL,
    "almalinux-9": ALMA_9_IMAGE_URL,
    "rocky-8": ROCKY_8_IMAGE_URL,
    "rocky-9": ROCKY_9_IMAGE_URL,
    
    // CentOS
    "centos-7": CENTOS_7_IMAGE_URL,
    "centos-9": CENTOS_9_IMAGE_URL,
}

type VPS struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Hostname    string    `json:"hostname"`
    Status      string    `json:"status"`
    ImageType   string    `json:"image_type"`
    Template    string    `json:"template"`        // Add template to VPS struct
    QEMUPid     int       `json:"qemu_pid,omitempty"`
    VNCPort     int       `json:"vnc_port"`
    SSHPort     int       `json:"ssh_port"`
    CreatedAt   time.Time `json:"created_at"`
    ExpiresAt   time.Time `json:"expires_at"`
    ImagePath   string    `json:"image_path"`
    Password    string    `json:"password"`
    Stage       string    `json:"stage"`           // Current stage of creation
    Progress    int       `json:"progress"`        // Progress percentage (0-100)
    ErrorMsg    string    `json:"error,omitempty"` // Error message if something fails
}


type VPSTemplate struct {
    ID          string            `json:"id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    OSVariants  []string         `json:"os_variants"`     // Supported OS images
    Packages    map[string][]string `json:"packages"`     // OS-specific packages
    Commands    map[string][]string `json:"commands"`     // OS-specific commands
}

type VPSManager struct {
    instances    map[string]*VPS
    ipInstances  map[string]string  // maps IP -> VPS ID
    mutex        sync.RWMutex
    nextVNCPort  int
    nextSSHPort  int
    baseDir      string
    metricsCache map[string]*MetricsCache
    metricsMutex sync.RWMutex
}


type MetricsCache struct {
    LastUpdate     time.Time
    LastDiskStats  DiskMetrics
    LastNetStats   NetworkMetrics
    MetricsHistory []ResourceMetrics
}


var SUPPORTED_TEMPLATES = map[string]VPSTemplate{
    "blank": {
        ID:          "blank",
        Name:        "Blank Server",
        Description: "Basic server with no additional software",
        OSVariants:  []string{"ubuntu-24.04", "ubuntu-22.04", "ubuntu-20.04", "debian-12", "debian-11", "fedora-40", "fedora-38", "rocky-9", "rocky-8", "almalinux-9", "almalinux-8",},
    },
    "docker": {
        ID:          "docker",
        Name:        "Docker Development Environment",
        Description: "Server with Docker and Docker Compose pre-installed",
        OSVariants:  []string{"ubuntu-22.04", "ubuntu-20.04", "debian-12", "debian-11", "fedora-40", "fedora-38", "rocky-9", "rocky-8", "almalinux-9", "almalinux-8"},
        Packages: map[string][]string{
            "ubuntu": {"apt-transport-https", "ca-certificates", "curl", "software-properties-common"},
            "debian": {"apt-transport-https", "ca-certificates", "curl", "software-properties-common"},
            "fedora": {"dnf-plugins-core", "curl"},
            "rocky":  {"yum-utils", "epel-release"},
            "almalinux": {"yum-utils", "epel-release"},
            "centos": {"yum-utils", "epel-release"},
        },
        Commands: map[string][]string{
            "ubuntu": {
                "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -",
                "add-apt-repository \"deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable\"",
                "apt-get update",
                "apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin",
                "systemctl enable docker",
                "systemctl start docker",
            },
            "debian": {
                "curl -fsSL https://download.docker.com/linux/debian/gpg | apt-key add -",
                "add-apt-repository \"deb [arch=amd64] https://download.docker.com/linux/debian $(lsb_release -cs) stable\"",
                "apt-get update",
                "apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin",
                "systemctl enable docker",
                "systemctl start docker",
            },
            "fedora": {
                "dnf -y remove docker docker-* podman* buildah*",
                "dnf config-manager --add-repo https://download.docker.com/linux/fedora/docker-ce.repo",
                "dnf -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin",
                "systemctl enable docker",
                "systemctl start docker",
            },
            "rocky": {
                "dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo",
                "dnf -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin",
                "systemctl enable docker",
                "systemctl start docker",
            },
            "almalinux": {
                "dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo",
                "dnf -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin",
                "systemctl enable docker",
                "systemctl start docker",
            },
            "centos": {
                "if [ -f /etc/centos-release ] && grep -q 'CentOS Linux release 7' /etc/centos-release; then " +
                    "yum-config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo && " +
                    "yum -y install docker-ce docker-ce-cli containerd.io; " +
                "else " +
                    "dnf config-manager --add-repo https://download.docker.com/linux/centos/docker-ce.repo && " +
                    "dnf -y install docker-ce docker-ce-cli containerd.io docker-compose-plugin; " +
                "fi",
                "systemctl enable docker",
                "systemctl start docker",
            },
        },
    },
    "nodejs": {
        ID:          "nodejs",
        Name:        "Node.js Development Environment",
        Description: "Server with Node.js, NPM, and common development tools",
        OSVariants:  []string{"ubuntu-22.04", "ubuntu-20.04", "debian-12", "debian-11", "fedora-40", "fedora-38", "rocky-9", "rocky-8", "almalinux-9", "almalinux-8"},
        Packages: map[string][]string{
            "ubuntu": {"curl", "build-essential"},
            "debian": {"curl", "build-essential"},
            "fedora": {"curl", "gcc", "gcc-c++", "make", "python3"},
            "rocky": {"curl", "gcc", "gcc-c++", "make", "epel-release", "python3"},
            "almalinux": {"curl", "gcc", "gcc-c++", "make", "epel-release", "python3"},
            "centos": {"curl", "gcc", "gcc-c++", "make", "epel-release", "python3"},
        },
        Commands: map[string][]string{
            "ubuntu": {
                "curl -fsSL https://deb.nodesource.com/setup_18.x | bash -",
                "apt-get install -y nodejs",
                "npm install -g yarn pm2 typescript ts-node",
            },
            "debian": {
                "curl -fsSL https://deb.nodesource.com/setup_18.x | bash -",
                "apt-get install -y nodejs",
                "npm install -g yarn pm2 typescript ts-node",
            },
            "fedora": {
                "dnf -y module reset nodejs",
                "dnf -y module enable nodejs:18",
                "dnf -y install nodejs",
                "npm install -g yarn pm2 typescript ts-node",
            },
            "rocky": {
                "curl -fsSL https://rpm.nodesource.com/setup_18.x | bash -",
                "dnf -y install nodejs",
                "npm install -g yarn pm2 typescript ts-node",
            },
            "almalinux": {
                "curl -fsSL https://rpm.nodesource.com/setup_18.x | bash -",
                "dnf -y install nodejs",
                "npm install -g yarn pm2 typescript ts-node",
            },
            "centos": {
                "if [ -f /etc/centos-release ] && grep -q 'CentOS Linux release 7' /etc/centos-release; then " +
                    "curl -fsSL https://rpm.nodesource.com/setup_18.x | bash - && " +
                    "yum -y install nodejs; " +
                "else " +
                    "curl -fsSL https://rpm.nodesource.com/setup_18.x | bash - && " +
                    "dnf -y install nodejs; " +
                "fi",
                "npm install -g yarn pm2 typescript ts-node",
            },
        },
    },
    "golang": {
        ID:          "golang",
        Name:        "Go Development Environment",
        Description: "Server with Go and common development tools",
        OSVariants:  []string{"ubuntu-22.04", "ubuntu-20.04", "debian-12", "debian-11", "fedora-40", "fedora-38", "rocky-9", "rocky-8", "almalinux-9", "almalinux-8"},
        Packages: map[string][]string{
            "ubuntu": {"curl", "git", "build-essential"},
            "debian": {"curl", "git", "build-essential"},
            "fedora": {"curl", "git", "gcc", "gcc-c++", "make"},
            "rocky": {"curl", "git", "gcc", "gcc-c++", "make"},
            "almalinux": {"curl", "git", "gcc", "gcc-c++", "make"},
            "centos": {"curl", "git", "gcc", "gcc-c++", "make"},
        },
        Commands: map[string][]string{
            "ubuntu": {
                "curl -OL https://go.dev/dl/go1.21.5.linux-amd64.tar.gz",
                "rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc",
                "rm go1.21.5.linux-amd64.tar.gz",
                "/usr/local/go/bin/go install golang.org/x/tools/gopls@latest",
                "/usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest",
            },
            "debian": {
                "curl -OL https://go.dev/dl/go1.21.5.linux-amd64.tar.gz",
                "rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc",
                "rm go1.21.5.linux-amd64.tar.gz",
                "/usr/local/go/bin/go install golang.org/x/tools/gopls@latest",
                "/usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest",
            },
            "fedora": {
                "curl -OL https://go.dev/dl/go1.21.5.linux-amd64.tar.gz",
                "rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc",
                "rm go1.21.5.linux-amd64.tar.gz",
                "/usr/local/go/bin/go install golang.org/x/tools/gopls@latest",
                "/usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest",
            },
            "rocky": {
                "curl -OL https://go.dev/dl/go1.21.5.linux-amd64.tar.gz",
                "rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc",
                "rm go1.21.5.linux-amd64.tar.gz",
                "/usr/local/go/bin/go install golang.org/x/tools/gopls@latest",
                "/usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest",
            },
            "almalinux": {
                "curl -OL https://go.dev/dl/go1.21.5.linux-amd64.tar.gz",
                "rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc",
                "rm go1.21.5.linux-amd64.tar.gz",
                "/usr/local/go/bin/go install golang.org/x/tools/gopls@latest",
                "/usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest",
            },
            "centos": {
                "curl -OL https://go.dev/dl/go1.21.5.linux-amd64.tar.gz",
                "rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile",
                "echo 'export PATH=$PATH:/usr/local/go/bin' >> /root/.bashrc",
                "rm go1.21.5.linux-amd64.tar.gz",
                "/usr/local/go/bin/go install golang.org/x/tools/gopls@latest",
                "/usr/local/go/bin/go install github.com/go-delve/delve/cmd/dlv@latest",
            },
        },
    },
    "python": {
        ID:          "python",
        Name:        "Python Development Environment",
        Description: "Server with Python, pip, and common development tools",
        OSVariants:  []string{"ubuntu-22.04", "ubuntu-20.04", "debian-12", "debian-11", "fedora-40", "fedora-38", "rocky-9", "rocky-8", "almalinux-9", "almalinux-8"},
        Packages: map[string][]string{
            "ubuntu": {"python3", "python3-pip", "python3-venv", "build-essential", "python3-dev", "git"},
            "debian": {"python3", "python3-pip", "python3-venv", "build-essential", "python3-dev", "git"},
            "fedora": {"python3", "python3-pip", "python3-devel", "gcc", "gcc-c++", "make", "git", "python3-wheel"},
            "rocky": {"epel-release", "python3", "python3-pip", "python3-devel", "gcc", "gcc-c++", "make", "git"},
            "almalinux": {"epel-release", "python3", "python3-pip", "python3-devel", "gcc", "gcc-c++", "make", "git"},
            "centos": {"epel-release", "python3", "python3-pip", "python3-devel", "gcc", "gcc-c++", "make", "git"},
        },
        Commands: map[string][]string{
            "ubuntu": {
                "pip3 install --upgrade pip",
                "pip3 install poetry virtualenv pylint black mypy pytest jupyter",
                "echo 'alias python=python3' >> /root/.bashrc",
                "echo 'alias pip=pip3' >> /root/.bashrc",
            },
            "debian": {
                "pip3 install --upgrade pip",
                "pip3 install poetry virtualenv pylint black mypy pytest jupyter",
                "echo 'alias python=python3' >> /root/.bashrc",
                "echo 'alias pip=pip3' >> /root/.bashrc",
            },
            "fedora": {
                "dnf -y update",
                "python3 -m ensurepip --upgrade",
                "python3 -m pip install --upgrade pip setuptools wheel",
                "python3 -m pip install poetry virtualenv pylint black mypy pytest jupyter",
                "echo 'alias python=python3' >> /root/.bashrc",
                "echo 'alias pip=pip3' >> /root/.bashrc",
            },
            "rocky": {
                "dnf -y update",
                "python3 -m pip install --upgrade pip",
                "python3 -m pip install poetry virtualenv pylint black mypy pytest jupyter",
                "echo 'alias python=python3' >> /root/.bashrc",
                "echo 'alias pip=pip3' >> /root/.bashrc",
            },
            "almalinux": {
                "dnf -y update",
                "python3 -m pip install --upgrade pip",
                "python3 -m pip install poetry virtualenv pylint black mypy pytest jupyter",
                "echo 'alias python=python3' >> /root/.bashrc",
                "echo 'alias pip=pip3' >> /root/.bashrc",
            },
            "centos": {
                "if [ -f /etc/centos-release ] && grep -q 'CentOS Linux release 7' /etc/centos-release; then " +
                    "yum -y update && " +
                    "python3 -m pip install --upgrade pip && " +
                    "python3 -m pip install poetry virtualenv pylint black mypy pytest jupyter; " +
                "else " +
                    "dnf -y update && " +
                    "python3 -m pip install --upgrade pip && " +
                    "python3 -m pip install poetry virtualenv pylint black mypy pytest jupyter; " +
                "fi",
                "echo 'alias python=python3' >> /root/.bashrc",
                "echo 'alias pip=pip3' >> /root/.bashrc",
            },
        },
    },
}


func getBaseImagePath(imageType string) string {
    return filepath.Join(BASE_DIR, imageType + ".qcow2")
}

func checkProcess(pid int) error {
    proc, err := os.FindProcess(pid)
    if err != nil {
        return fmt.Errorf("process not found: %v", err)
    }

    if err := proc.Signal(syscall.Signal(0)); err != nil {
        return fmt.Errorf("process check failed: %v", err)
    }

    cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
    if err != nil {
        return fmt.Errorf("failed to read process cmdline: %v", err)
    }

    cmdline := string(cmdlineBytes)
    if !strings.Contains(cmdline, "qemu-system") {
        return fmt.Errorf("process is not a QEMU process")
    }

    return nil
}

func generatePassword() (string, error) {
    bytes := make([]byte, 4)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(bytes)[:6], nil
}

func NewVPSManager(baseDir string) (*VPSManager, error) {
    dirs := []string{"images", "disks", "logs", "base"}
    for _, dir := range dirs {
        path := filepath.Join(baseDir, dir)
        if err := os.MkdirAll(path, 0755); err != nil {
            return nil, fmt.Errorf("failed to create directory %s: %v", path, err)
        }
    }

    for imageType := range SUPPORTED_IMAGES {
        baseImagePath := getBaseImagePath(imageType)
        if _, err := os.Stat(baseImagePath); os.IsNotExist(err) {
            if err := downloadAndPrepareBaseImage(imageType); err != nil {
                log.Printf("Warning: Failed to prepare %s base image: %v", imageType, err)
            }
        }
    }


    manager := &VPSManager{
        instances:     make(map[string]*VPS),
        ipInstances:   make(map[string]string),
        nextVNCPort:   5900,
        nextSSHPort:   SSH_PORT_START,
        baseDir:       baseDir,
        metricsCache:  make(map[string]*MetricsCache),
    }


    // Start metrics collection routine
    go manager.metricsCollector()
    
    return manager, nil
}


func (m *VPSManager) hasVPSForIP(ip string) (bool, string) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()
    
    if vpsID, exists := m.ipInstances[ip]; exists {
        if vps, ok := m.instances[vpsID]; ok {
            // Check if VPS has expired
            if time.Now().After(vps.ExpiresAt) {
                return false, ""
            }
            return true, vpsID
        }
    }
    return false, ""
}


func downloadAndPrepareBaseImage(imageType string) error {
    imageURL, exists := SUPPORTED_IMAGES[imageType]
    if !exists {
        return fmt.Errorf("unsupported image type: %s", imageType)
    }

    log.Printf("Starting base image preparation for %s", imageType)
    
    tmpDir := "/tmp/vps-download"
    if err := os.MkdirAll(tmpDir, 0755); err != nil {
        return fmt.Errorf("failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tmpDir)

    tmpImagePath := filepath.Join(tmpDir, filepath.Base(imageURL))
    baseImagePath := getBaseImagePath(imageType)
    
    log.Printf("Downloading %s image to %s", imageType, tmpImagePath)
    downloadCmd := exec.Command("wget",
        "--progress=bar:force",
        "-O", tmpImagePath,
        imageURL)
    downloadCmd.Stdout = os.Stdout
    downloadCmd.Stderr = os.Stderr
    
    if err := downloadCmd.Run(); err != nil {
        return fmt.Errorf("failed to download image: %v", err)
    }

    baseDir := filepath.Dir(baseImagePath)
    if err := os.MkdirAll(baseDir, 0755); err != nil {
        return fmt.Errorf("failed to create base directory: %v", err)
    }

    log.Printf("Converting and resizing image to %dG", DISK_SIZE)
    convertCmd := exec.Command("qemu-img", "convert",
        "-f", "qcow2",
        "-O", "qcow2",
        tmpImagePath,
        baseImagePath)
    
    if output, err := convertCmd.CombinedOutput(); err != nil {
        return fmt.Errorf("failed to convert image: %v, output: %s", err, string(output))
    }

    resizeCmd := exec.Command("qemu-img", "resize", baseImagePath, fmt.Sprintf("%dG", DISK_SIZE))
    if output, err := resizeCmd.CombinedOutput(); err != nil {
        return fmt.Errorf("failed to resize image: %v, output: %s", err, string(output))
    }

    if err := os.Chmod(baseImagePath, 0644); err != nil {
        return fmt.Errorf("failed to set image permissions: %v", err)
    }

    log.Printf("Base image preparation completed successfully for %s", imageType)
    return nil
}

func prependIndent(commands []string, indent string) []string {
    indented := make([]string, len(commands))
    for i, cmd := range commands {
        indented[i] = indent + cmd
    }
    return indented
}

func createCloudInitISO(path string, rootPassword string, imageType string, hostname string, template string) error {
    tmpDir, err := os.MkdirTemp("", "cloud-init")
    if err != nil {
        return err
    }
    defer os.RemoveAll(tmpDir)

    // Get template configuration
    templateConfig, exists := SUPPORTED_TEMPLATES[template]
    if !exists {
        templateConfig = SUPPORTED_TEMPLATES["blank"]
    }

    // Determine OS family for package management
    osFamily := getOSFamily(imageType)
    if osFamily == "" {
        return fmt.Errorf("unsupported OS type: %s", imageType)
    }

    // Get OS-specific packages and commandsa
    packages := templateConfig.Packages[osFamily]
    commands := templateConfig.Commands[osFamily]

    // Combine all commands including package installation
    var allCommands []string

    // Add package installation commands based on OS family
    if len(packages) > 0 {
        switch osFamily {
        case "ubuntu", "debian":
            allCommands = append(allCommands,
                "apt-get update",
                "DEBIAN_FRONTEND=noninteractive apt-get install -y "+strings.Join(packages, " "))
        case "fedora", "rocky", "almalinux", "centos":
            allCommands = append(allCommands,
                "dnf update -y",
                "dnf install -y "+strings.Join(packages, " "))
        }
    }

    // Add template-specific commands
    allCommands = append(allCommands, commands...)

    // Create cloud-init user-data content
    var userData bytes.Buffer
    userData.WriteString(fmt.Sprintf(`#cloud-config
users:
  - name: root
    lock_passwd: false
    ssh_pwauth: true

chpasswd:
  list: |
    root:%s
  expire: false

ssh_pwauth: true
disable_root: false

hostname: %s

package_update: true
package_upgrade: true

# Install required packages
packages:
%s

# Run commands
runcmd:
  - sed -i 's/#PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config
  - systemctl restart ssh || systemctl restart sshd
%s
`, rootPassword, hostname, formatPackageList(packages), formatCommandList(allCommands)))

    if err := os.WriteFile(filepath.Join(tmpDir, "user-data"), userData.Bytes(), 0644); err != nil {
        return err
    }

    metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", uuid.New().String(), hostname)
    if err := os.WriteFile(filepath.Join(tmpDir, "meta-data"), []byte(metaData), 0644); err != nil {
        return err
    }

    cmd := exec.Command("genisoimage", "-output", path, "-volid", "cidata", "-joliet", "-rock",
        filepath.Join(tmpDir, "user-data"), filepath.Join(tmpDir, "meta-data"))
    
    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("failed to create ISO: %v, output: %s", err, string(output))
    }

    return nil
}

// Helper function to format command list for cloud-init
func formatCommandList(commands []string) string {
    var formatted strings.Builder
    for _, cmd := range commands {
        formatted.WriteString(fmt.Sprintf("  - %s\n", cmd))
    }
    return formatted.String()
}

// Helper function to format package list for cloud-init
func formatPackageList(packages []string) string {
    var formatted strings.Builder
    for _, pkg := range packages {
        formatted.WriteString(fmt.Sprintf("  - %s\n", pkg))
    }
    return formatted.String()
}

// Helper function to determine OS family
func getOSFamily(imageType string) string {
    switch {
    case strings.HasPrefix(imageType, "ubuntu"):
        return "ubuntu"
    case strings.HasPrefix(imageType, "debian"):
        return "debian"
    case strings.HasPrefix(imageType, "fedora"):
        return "fedora"
    case strings.HasPrefix(imageType, "rocky"):
        return "rocky"
    case strings.HasPrefix(imageType, "almalinux"):
        return "almalinux"
    case strings.HasPrefix(imageType, "centos"):
        return "centos"
    default:
        return ""
    }
}

// Add validation for template and OS compatibility
func validateTemplateAndOS(template string, imageType string) error {
    templateConfig, exists := SUPPORTED_TEMPLATES[template]
    if !exists {
        return fmt.Errorf("unsupported template: %s", template)
    }

    if len(templateConfig.OSVariants) > 0 {
        supported := false
        for _, variant := range templateConfig.OSVariants {
            if variant == imageType {
                supported = true
                break
            }
        }
        if !supported {
            return fmt.Errorf("template %s does not support OS %s", template, imageType)
        }
    }

    return nil
}

// Modify the HTTP handler for listing templates to include OS compatibility
func (m *VPSManager) handleListTemplates(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Get OS filter from query parameter
    osType := r.URL.Query().Get("os")

    templates := make([]struct {
        VPSTemplate
        Compatible bool `json:"compatible"`
    }, 0, len(SUPPORTED_TEMPLATES))

    for _, template := range SUPPORTED_TEMPLATES {
        compatible := true
        if osType != "" {
            compatible = false
            for _, variant := range template.OSVariants {
                if variant == osType {
                    compatible = true
                    break
                }
            }
        }

        templates = append(templates, struct {
            VPSTemplate
            Compatible bool `json:"compatible"`
        }{
            VPSTemplate: template,
            Compatible: compatible,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(templates)
}

func startWebsockifyProxy(vncPort int) error {
    wsPort := vncPort + 1000

    killCmd := exec.Command("pkill", "-f", fmt.Sprintf("websockify.*:%d", wsPort))
    killCmd.Run()

    time.Sleep(time.Second)

    logFile, err := os.Create(fmt.Sprintf("/tmp/websockify_%d.log", wsPort))
    if err != nil {
        return fmt.Errorf("failed to create websockify log file: %v", err)
    }
    defer logFile.Close()

    cmd := exec.Command("websockify",
        "--verbose",
        fmt.Sprintf("%d", wsPort),
        fmt.Sprintf("localhost:%d", vncPort),
        "--web", "/usr/share/novnc",
    )
    
    cmd.Stdout = logFile
    cmd.Stderr = logFile

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start websockify: %v", err)
    }

    go func() {
        if err := cmd.Wait(); err != nil {
            log.Printf("Websockify process ended: %v", err)
        }
    }()

    time.Sleep(2 * time.Second)

    checkCmd := exec.Command("pgrep", "-f", fmt.Sprintf("websockify.*:%d", wsPort))
    if err := checkCmd.Run(); err != nil {
        logContent, _ := os.ReadFile(fmt.Sprintf("/tmp/websockify_%d.log", wsPort))
        return fmt.Errorf("websockify failed to start: %v, logs: %s", err, string(logContent))
    }

    return nil
}

func stopWebsockifyProxy(vncPort int) error {
    wsPort := vncPort + 1000
    cmd := exec.Command("pkill", "-f", fmt.Sprintf("websockify.*:%d", wsPort))
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to stop websockify: %v", err)
    }
    return nil
}

func (m *VPSManager) CreateVPS(name string, hostname string, imageType string, template string) (*VPS, error) {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    log.Printf("Starting VPS creation process for: %s with image: %s, template: %s and hostname: %s", 
        name, imageType, template, hostname)

    // Initialize VPS with template
    vps := &VPS{
        ID:          uuid.New().String(),
        Name:        name,
        Hostname:    hostname,
        Status:      "creating",
        ImageType:   imageType,
        Template:    template,  // Add template to VPS struct
        VNCPort:     m.nextVNCPort,
        SSHPort:     m.nextSSHPort,
        CreatedAt:   time.Now(),
        ExpiresAt:   time.Now().Add(VPS_LIFETIME),
        Stage:       StageInitializing,
        Progress:    0,
    }
    m.nextVNCPort++
    m.nextSSHPort++
    
    // Store the instance immediately so progress can be tracked
    m.instances[vps.ID] = vps

    // Run creation in a goroutine to allow progress tracking
    go func() {
        if err := m.createVPSWithProgress(vps); err != nil {
            m.mutex.Lock()
            vps.Status = "failed"
            vps.Stage = StageFailed
            vps.ErrorMsg = err.Error()
            m.mutex.Unlock()
            log.Printf("Failed to create VPS %s: %v", vps.ID, err)
            return
        }
    }()

    return vps, nil
}

func (m *VPSManager) createVPSWithProgress(vps *VPS) error {
    updateProgress := func(stage string, progress int) {
        m.mutex.Lock()
        vps.Stage = stage
        vps.Progress = progress
        m.mutex.Unlock()
    }

    // Validate image type
    updateProgress(StageInitializing, 10)
    if _, exists := SUPPORTED_IMAGES[vps.ImageType]; !exists {
        return fmt.Errorf("unsupported image type: %s", vps.ImageType)
    }

    // Validate hostname
    if !isValidHostname(vps.Hostname) {
        return fmt.Errorf("invalid hostname format: %s", vps.Hostname)
    }

    // Check/prepare base image
    updateProgress(StageInitializing, 20)
    baseImagePath := getBaseImagePath(vps.ImageType)
    if _, err := os.Stat(baseImagePath); os.IsNotExist(err) {
        if err := downloadAndPrepareBaseImage(vps.ImageType); err != nil {
            return fmt.Errorf("failed to prepare base image: %v", err)
        }
    }

    // Generate password
    password, err := generatePassword()
    if err != nil {
        return fmt.Errorf("failed to generate password: %v", err)
    }
    vps.Password = password

    // Create instance directory
    instanceDir := filepath.Join(m.baseDir, "disks", vps.ID)
    if err := os.MkdirAll(instanceDir, 0755); err != nil {
        return fmt.Errorf("failed to create instance directory: %v", err)
    }

    // Create disk image
    updateProgress(StageCreatingDisk, 40)
    vps.ImagePath = filepath.Join(instanceDir, "disk.qcow2")
    createDisk := exec.Command("qemu-img", "create",
        "-f", "qcow2",
        "-F", "qcow2",
        "-b", baseImagePath,
        vps.ImagePath)
    
    if output, err := createDisk.CombinedOutput(); err != nil {
        return fmt.Errorf("failed to create disk: %v, output: %s", err, string(output))
    }

    // Create cloud-init ISO
    updateProgress(StagePreparingCloudInit, 60)
    cloudInitPath := filepath.Join(instanceDir, "cloud-init.iso")
    if err := createCloudInitISO(cloudInitPath, vps.Password, vps.ImageType, vps.Hostname, vps.Template); err != nil {
        return fmt.Errorf("failed to create cloud-init ISO: %v", err)
    }

    // Start QEMU
    updateProgress(StageStartingQEMU, 80)
    pidFile := filepath.Join(instanceDir, "qemu.pid")
    logFile := filepath.Join(m.baseDir, "logs", fmt.Sprintf("%s.log", vps.ID))
    monitorSocket := filepath.Join(instanceDir, "qemu-monitor.sock")

    args := []string{
        "-name", fmt.Sprintf("guest=%s,debug-threads=on", vps.Name),
        "-machine", "pc,accel=kvm,usb=off,vmport=off",
        "-cpu", "host",
        "-m", fmt.Sprintf("%d", RAM_SIZE),
        "-smp", "2,sockets=2,cores=1,threads=1",
        "-drive", fmt.Sprintf("file=%s,format=qcow2", vps.ImagePath),
        "-drive", fmt.Sprintf("file=%s,format=raw", cloudInitPath),
        "-vnc", fmt.Sprintf("0.0.0.0:%d", vps.VNCPort-5900),
        "-device", fmt.Sprintf("virtio-net-pci,netdev=net0,mac=%s", generateMacAddress(vps.ID)),
        "-netdev", fmt.Sprintf(
            "user,id=net0,hostfwd=tcp:0.0.0.0:%d-:22",
            vps.SSHPort,
        ),
        "-qmp", fmt.Sprintf("unix:%s,server,nowait", monitorSocket),
        "-pidfile", pidFile,
        "-daemonize",
        "-enable-kvm",
    }


    cmd := exec.Command("qemu-system-x86_64", args...)
    
    stdout, err := os.Create(logFile)
    if err != nil {
        return fmt.Errorf("failed to create log file: %v", err)
    }
    defer stdout.Close()
    cmd.Stdout = stdout
    cmd.Stderr = stdout

    if err := cmd.Start(); err != nil {
        return fmt.Errorf("failed to start QEMU: %v", err)
    }

    // Wait for PID file
    var pid int
    timeout := time.After(30 * time.Second)
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-timeout:
            logs, _ := os.ReadFile(logFile)
            return fmt.Errorf("timeout waiting for QEMU to start. Logs: %s", string(logs))
            
        case <-ticker.C:
            if pidBytes, err := os.ReadFile(pidFile); err == nil {
                if _, err := fmt.Sscanf(string(pidBytes), "%d", &pid); err == nil {
                    goto pidFound
                }
            }
        }
    }

pidFound:
    // Verify QEMU process
    retries := 3
    for i := 0; i < retries; i++ {
        if err := checkProcess(pid); err == nil {
            break
        }
        if i == retries-1 {
            logs, _ := os.ReadFile(logFile)
            return fmt.Errorf("QEMU process verification failed after %d retries. Logs: %s", retries, string(logs))
        }
        time.Sleep(time.Second)
    }

    vps.QEMUPid = pid

    // Configure VNC
    updateProgress(StageConfigVNC, 90)
    if err := startWebsockifyProxy(vps.VNCPort); err != nil {
        log.Printf("Warning: Failed to start websockify proxy: %v", err)
    }

    // Complete
    updateProgress(StageCompleted, 100)
    m.mutex.Lock()
    vps.Status = "running"
    m.mutex.Unlock()

    // Schedule cleanup
    go m.scheduleCleanup(vps)

    return nil
}

func isValidHostname(hostname string) bool {
    if len(hostname) > 253 {
        return false
    }
    
    parts := strings.Split(hostname, ".")
    for _, part := range parts {
        if len(part) > 63 {
            return false
        }
        if !regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`).MatchString(part) {
            return false
        }
    }
    
    return true
}


func (m *VPSManager) StopVPS(id string) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    vps, exists := m.instances[id]
    if !exists {
        return fmt.Errorf("VPS not found")
    }

    if vps.Status == StatusStopped {
        return fmt.Errorf("VPS is already stopped")
    }

    if vps.QEMUPid <= 0 {
        return fmt.Errorf("VPS does not have a valid PID")
    }

    // Get the QEMU monitor socket path
    instanceDir := filepath.Join(m.baseDir, "disks", vps.ID)
    monitorSocket := filepath.Join(instanceDir, "qemu-monitor.sock")

    // Create a temporary file for command output
    tmpFile, err := os.CreateTemp("", "qemu-command-*")
    if err != nil {
        return fmt.Errorf("failed to create temp file: %v", err)
    }
    defer os.Remove(tmpFile.Name())

    // Send system_powerdown command to QEMU monitor
    cmd := exec.Command("echo", "system_powerdown")
    socat := exec.Command("socat", "-", fmt.Sprintf("UNIX-CONNECT:%s", monitorSocket))
    
    // Connect the commands
    socatIn, err := socat.StdinPipe()
    if err != nil {
        return fmt.Errorf("failed to create pipe: %v", err)
    }
    
    cmd.Stdout = socatIn
    socat.Stdout = tmpFile
    socat.Stderr = tmpFile

    // Start socat first
    if err := socat.Start(); err != nil {
        return fmt.Errorf("failed to start socat: %v", err)
    }

    // Run the echo command
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to send command: %v", err)
    }

    // Close stdin pipe
    socatIn.Close()

    // Wait for socat to finish
    if err := socat.Wait(); err != nil {
        output, _ := os.ReadFile(tmpFile.Name())
        return fmt.Errorf("failed to execute command: %v, output: %s", err, string(output))
    }

    vps.Status = StatusStopping

    // Wait for shutdown to complete
    go func() {
        timeout := time.After(2 * time.Minute)
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()

        for {
            select {
            case <-timeout:
                // Force stop if graceful shutdown fails
                if proc, err := os.FindProcess(vps.QEMUPid); err == nil {
                    proc.Kill()
                }
                m.mutex.Lock()
                vps.Status = StatusStopped
                m.mutex.Unlock()
                return
                
            case <-ticker.C:
                if err := checkProcess(vps.QEMUPid); err != nil {
                    m.mutex.Lock()
                    vps.Status = StatusStopped
                    m.mutex.Unlock()
                    return
                }
            }
        }
    }()

    return nil
}

func (m *VPSManager) StartVPS(id string) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    vps, exists := m.instances[id]
    if !exists {
        return fmt.Errorf("VPS not found")
    }

    if vps.Status == StatusRunning {
        return fmt.Errorf("VPS is already running")
    }

    instanceDir := filepath.Join(m.baseDir, "disks", vps.ID)
    pidFile := filepath.Join(instanceDir, "qemu.pid")
    logFile := filepath.Join(m.baseDir, "logs", fmt.Sprintf("%s.log", vps.ID))
    cloudInitPath := filepath.Join(instanceDir, "cloud-init.iso")
    monitorSocket := filepath.Join(instanceDir, "qemu-monitor.sock")

    // Remove existing monitor socket if it exists
    os.Remove(monitorSocket)

    args := []string{
        "-name", fmt.Sprintf("guest=%s,debug-threads=on", vps.Name),
        "-machine", "pc,accel=kvm,usb=off,vmport=off",
        "-cpu", "host",
        "-m", fmt.Sprintf("%d", RAM_SIZE),
        "-smp", "2,sockets=2,cores=1,threads=1",
        "-drive", fmt.Sprintf("file=%s,format=qcow2", vps.ImagePath),
        "-drive", fmt.Sprintf("file=%s,format=raw", cloudInitPath),
        "-vnc", fmt.Sprintf("0.0.0.0:%d", vps.VNCPort-5900),
        "-device", "virtio-net-pci,netdev=user0",
        "-netdev", fmt.Sprintf(
            "user,id=user0,hostfwd=tcp:0.0.0.0:%d-:22",
            vps.SSHPort,
        ),
        "-qmp", fmt.Sprintf("unix:%s,server,nowait", monitorSocket),
        "-pidfile", pidFile,
        "-daemonize",
        "-enable-kvm",
    }

    cmd := exec.Command("qemu-system-x86_64", args...)
    
    stdout, err := os.Create(logFile)
    if err != nil {
        return fmt.Errorf("failed to create log file: %v", err)
    }
    defer stdout.Close()
    cmd.Stdout = stdout
    cmd.Stderr = stdout

    vps.Status = StatusStarting

    if err := cmd.Start(); err != nil {
        vps.Status = StatusStopped
        return fmt.Errorf("failed to start QEMU: %v", err)
    }

    // Wait for PID file
    var pid int
    timeout := time.After(30 * time.Second)
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-timeout:
            vps.Status = StatusStopped
            logs, _ := os.ReadFile(logFile)
            return fmt.Errorf("timeout waiting for QEMU to start. Logs: %s", string(logs))
            
        case <-ticker.C:
            if pidBytes, err := os.ReadFile(pidFile); err == nil {
                if _, err := fmt.Sscanf(string(pidBytes), "%d", &pid); err == nil {
                    goto pidFound
                }
            }
        }
    }

pidFound:
    // Verify QEMU process
    retries := 3
    for i := 0; i < retries; i++ {
        if err := checkProcess(pid); err == nil {
            break
        }
        if i == retries-1 {
            vps.Status = StatusStopped
            logs, _ := os.ReadFile(logFile)
            return fmt.Errorf("QEMU process verification failed after %d retries. Logs: %s", retries, string(logs))
        }
        time.Sleep(time.Second)
    }

    vps.QEMUPid = pid
    vps.Status = StatusRunning

    return nil
}

func (m *VPSManager) RestartVPS(id string) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    vps, exists := m.instances[id]
    if !exists {
        return fmt.Errorf("VPS not found")
    }

    if vps.Status != StatusRunning {
        return fmt.Errorf("VPS must be running to restart")
    }

    if vps.QEMUPid <= 0 {
        return fmt.Errorf("VPS does not have a valid PID")
    }

    // Get the QEMU monitor socket path
    instanceDir := filepath.Join(m.baseDir, "disks", vps.ID)
    monitorSocket := filepath.Join(instanceDir, "qemu-monitor.sock")

    // Create a temporary file for command output
    tmpFile, err := os.CreateTemp("", "qemu-command-*")
    if err != nil {
        return fmt.Errorf("failed to create temp file: %v", err)
    }
    defer os.Remove(tmpFile.Name())

    // Send system_reset command to QEMU monitor
    cmd := exec.Command("echo", "system_reset")
    socat := exec.Command("socat", "-", fmt.Sprintf("UNIX-CONNECT:%s", monitorSocket))
    
    // Connect the commands
    socatIn, err := socat.StdinPipe()
    if err != nil {
        return fmt.Errorf("failed to create pipe: %v", err)
    }
    
    cmd.Stdout = socatIn
    socat.Stdout = tmpFile
    socat.Stderr = tmpFile

    // Start socat first
    if err := socat.Start(); err != nil {
        return fmt.Errorf("failed to start socat: %v", err)
    }

    // Run the echo command
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("failed to send command: %v", err)
    }

    // Close stdin pipe
    socatIn.Close()

    // Wait for socat to finish
    if err := socat.Wait(); err != nil {
        output, _ := os.ReadFile(tmpFile.Name())
        return fmt.Errorf("failed to execute command: %v, output: %s", err, string(output))
    }

    vps.Status = StatusRestarting

    // Update status after a delay
    go func() {
        time.Sleep(30 * time.Second)
        m.mutex.Lock()
        vps.Status = StatusRunning
        m.mutex.Unlock()
    }()

    return nil
}

// Add new HTTP handlers for the start/stop operations
func (m *VPSManager) handleStartVPS(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    if err := m.StartVPS(id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func (m *VPSManager) handleStopVPS(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    if err := m.StopVPS(id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}
// Add new HTTP handler for restart endpoint
func (m *VPSManager) handleRestartVPS(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    if err := m.RestartVPS(id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func (m *VPSManager) scheduleCleanup(vps *VPS) {
    time.Sleep(VPS_LIFETIME)
    m.DeleteVPS(vps.ID)
}

func (m *VPSManager) DeleteVPS(id string) error {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    vps, exists := m.instances[id]
    if !exists {
        return fmt.Errorf("VPS not found")
    }

    // Remove IP association
    for ip, vpsID := range m.ipInstances {
        if vpsID == id {
            delete(m.ipInstances, ip)
            break
        }
    }

    if err := stopWebsockifyProxy(vps.VNCPort); err != nil {
        log.Printf("Warning: Failed to stop websockify: %v", err)
    }

    if vps.QEMUPid > 0 {
        if proc, err := os.FindProcess(vps.QEMUPid); err == nil {
            proc.Kill()
        }
    }

    instanceDir := filepath.Join(m.baseDir, "disks", vps.ID)
    os.RemoveAll(instanceDir)

    delete(m.instances, id)
    return nil
}

func (m *VPSManager) GetVPS(id string) (*VPS, error) {
    m.mutex.RLock()
    defer m.mutex.RUnlock()

    vps, exists := m.instances[id]
    if !exists {
        return nil, fmt.Errorf("VPS not found")
    }
    return vps, nil
}

func (m *VPSManager) ListVPS() []*VPS {
    m.mutex.RLock()
    defer m.mutex.RUnlock()

    vpsList := make([]*VPS, 0, len(m.instances))
    for _, vps := range m.instances {
        vpsList = append(vpsList, vps)
    }
    return vpsList
}

func (m *VPSManager) validateInstances() {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    for id, vps := range m.instances {
        if err := checkProcess(vps.QEMUPid); err != nil {
            log.Printf("VPS %s (ID: %s) is no longer running: %v", vps.Name, id, err)
            vps.Status = "stopped"
        }
    }
}

// HTTP Handlers
func (m *VPSManager) handleCreateVPS(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    var req struct {
        Name      string `json:"name"`
        Hostname  string `json:"hostname"`
        ImageType string `json:"image_type"`
        Template  string `json:"template"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Set defaults if not provided
    if req.Template == "" {
        req.Template = "blank"
    }
    if req.ImageType == "" {
        req.ImageType = "ubuntu-22.04"
    }
    if req.Hostname == "" {
        req.Hostname = req.Name + ".vps.local"
    }

    vps, err := m.CreateVPS(req.Name, req.Hostname, req.ImageType, req.Template)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(vps)
}

func (m *VPSManager) handleListVPS(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    m.validateInstances()
    vpsList := m.ListVPS()
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(vpsList)
}

func (m *VPSManager) handleGetVPS(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    vps, err := m.GetVPS(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(vps)
}

func (m *VPSManager) handleDeleteVPS(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodDelete {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    if err := m.DeleteVPS(id); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
}

func (m *VPSManager) handleListImages(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    images := make([]string, 0, len(SUPPORTED_IMAGES))
    for imageType := range SUPPORTED_IMAGES {
        images = append(images, imageType)
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(images)
}

func (m *VPSManager) handleGetProgress(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, "Missing VPS ID", http.StatusBadRequest)
        return
    }

    m.mutex.RLock()
    vps, exists := m.instances[id]
    m.mutex.RUnlock()

    if !exists {
        http.Error(w, "VPS not found", http.StatusNotFound)
        return
    }

    response := struct {
        Stage    string `json:"stage"`
        Progress int    `json:"progress"`
        Status   string `json:"status"`
        Error    string `json:"error,omitempty"`
    }{
        Stage:    vps.Stage,
        Progress: vps.Progress,
        Status:   vps.Status,
        Error:    vps.ErrorMsg,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

type AuthMiddleware struct {
    apiKey string
    next   http.Handler
}

func NewAuthMiddleware(apiKey string, next http.Handler) *AuthMiddleware {
    return &AuthMiddleware{
        apiKey: apiKey,
        next:   next,
    }
}

func (m *AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

    if r.Method == "OPTIONS" {
        w.WriteHeader(http.StatusOK)
        return
    }

    apiKey := r.Header.Get("X-API-Key")
    if apiKey == "" || apiKey != m.apiKey {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    m.next.ServeHTTP(w, r)
}

func verifySystemRequirements() error {
    if _, err := exec.LookPath("qemu-system-x86_64"); err != nil {
        return fmt.Errorf("qemu-system-x86_64 not found: %v", err)
    }

    if _, err := os.Stat("/dev/kvm"); err != nil {
        return fmt.Errorf("KVM not available: %v", err)
    }

    if output, err := exec.Command("ls", "-l", "/dev/kvm").CombinedOutput(); err != nil {
        return fmt.Errorf("failed to check KVM permissions: %v", err)
    } else {
        log.Printf("KVM device permissions: %s", string(output))
    }

    return nil
}

func (m *VPSManager) cleanup() {
    log.Println("Starting cleanup of all VPS instances...")
    
    m.mutex.Lock()
    defer m.mutex.Unlock()

    var wg sync.WaitGroup
    for id, vps := range m.instances {
        wg.Add(1)
        go func(id string, vps *VPS) {
            defer wg.Done()
            
            log.Printf("Cleaning up VPS %s (ID: %s)", vps.Name, id)
            
            if err := stopWebsockifyProxy(vps.VNCPort); err != nil {
                log.Printf("Warning: Failed to stop websockify for VPS %s: %v", id, err)
            }

            if vps.QEMUPid > 0 {
                if proc, err := os.FindProcess(vps.QEMUPid); err == nil {
                    log.Printf("Killing QEMU process %d for VPS %s", vps.QEMUPid, id)
                    proc.Kill()
                    proc.Wait()
                }
            }

            instanceDir := filepath.Join(m.baseDir, "disks", id)
            if err := os.RemoveAll(instanceDir); err != nil {
                log.Printf("Warning: Failed to remove instance directory for VPS %s: %v", id, err)
            }

            log.Printf("Successfully cleaned up VPS %s", id)
        }(id, vps)
    }

    wg.Wait()
    log.Println("All VPS instances have been cleaned up")
}




type ResourceMetrics struct {
    CPU     CPUMetrics     `json:"cpu"`
    Memory  MemoryMetrics  `json:"memory"`
    Disk    DiskMetrics    `json:"disk"`
    Network NetworkMetrics `json:"network"`
    Time    time.Time      `json:"time"`
}

type CPUMetrics struct {
    Usage float64 `json:"usage"` // Percentage (0-100)
}

type MemoryMetrics struct {
    Used  int64 `json:"used"`  // Bytes
    Total int64 `json:"total"` // Bytes
    Cache int64 `json:"cache"` // Bytes
}

type DiskMetrics struct {
    ReadBytes  int64   `json:"read_bytes"`
    WriteBytes int64   `json:"write_bytes"`
    ReadOps    int64   `json:"read_ops"`
    WriteOps   int64   `json:"write_ops"`
    ReadSpeed  float64 `json:"read_speed"`  // Bytes per second
    WriteSpeed float64 `json:"write_speed"` // Bytes per second
}

type NetworkMetrics struct {
    RXBytes    int64   `json:"rx_bytes"`
    TXBytes    int64   `json:"tx_bytes"`
    RXPackets  int64   `json:"rx_packets"`
    TXPackets  int64   `json:"tx_packets"`
    RXSpeed    float64 `json:"rx_speed"` // Bytes per second
    TXSpeed    float64 `json:"tx_speed"` // Bytes per second
}



func (m *VPSManager) metricsCollector() {
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        m.mutex.RLock()
        instances := make(map[string]*VPS)
        for id, vps := range m.instances {
            instances[id] = vps
        }
        m.mutex.RUnlock()

        for id, vps := range instances {
            if vps.Status == StatusRunning {
                if metrics, err := m.collectMetrics(id); err == nil {
                    m.updateMetricsCache(id, metrics)
                }
            }
        }
    }
}

func generateMacAddress(id string) string {
    // Use first 6 bytes of UUID as MAC address
    cleanID := strings.ReplaceAll(id, "-", "")
    if len(cleanID) < 12 {
        cleanID = cleanID + strings.Repeat("0", 12-len(cleanID))
    }
    return fmt.Sprintf("52:54:00:%s:%s:%s",
        cleanID[0:2],
        cleanID[2:4],
        cleanID[4:6])
}

func (m *VPSManager) collectMetrics(id string) (*ResourceMetrics, error) {
    m.mutex.RLock()
    vps, exists := m.instances[id]
    m.mutex.RUnlock()

    if !exists || vps.QEMUPid <= 0 {
        return nil, fmt.Errorf("VPS not found or not running")
    }

    metrics := &ResourceMetrics{
        Time: time.Now(),
    }

    // Get CPU stats from /proc/[pid]/stat
    if cpuStats, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", vps.QEMUPid)); err == nil {
        fields := strings.Fields(string(cpuStats))
        if len(fields) >= 15 {
            utime, _ := strconv.ParseInt(fields[13], 10, 64)
            stime, _ := strconv.ParseInt(fields[14], 10, 64)
            
            total := float64(utime + stime)
            // Calculate percentage based on total system time
            if uptime, err := os.ReadFile("/proc/uptime"); err == nil {
                uptimeFields := strings.Fields(string(uptime))
                if systemUptime, err := strconv.ParseFloat(uptimeFields[0], 64); err == nil {
                    numCPUs := float64(runtime.NumCPU())
                    cpuUsage := (total / systemUptime) * (100 / numCPUs)
                    metrics.CPU = CPUMetrics{
                        Usage: cpuUsage,
                    }
                }
            }
        }
    }

    // Get memory stats from /proc/[pid]/status
    if memStats, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", vps.QEMUPid)); err == nil {
        var vmSize, rss int64
        scanner := bufio.NewScanner(strings.NewReader(string(memStats)))
        for scanner.Scan() {
            line := scanner.Text()
            if strings.HasPrefix(line, "VmSize:") {
                fields := strings.Fields(line)
                if len(fields) >= 2 {
                    vmSize, _ = strconv.ParseInt(fields[1], 10, 64)
                    vmSize *= 1024 // Convert from KB to bytes
                }
            } else if strings.HasPrefix(line, "VmRSS:") {
                fields := strings.Fields(line)
                if len(fields) >= 2 {
                    rss, _ = strconv.ParseInt(fields[1], 10, 64)
                    rss *= 1024 // Convert from KB to bytes
                }
            }
        }
        metrics.Memory = MemoryMetrics{
            Used:  rss,
            Total: int64(RAM_SIZE) * 1024 * 1024, // Convert MB to bytes
            Cache: vmSize - rss,
        }
    }

    // Get disk I/O stats from /proc/[pid]/io
    if ioStats, err := os.ReadFile(fmt.Sprintf("/proc/%d/io", vps.QEMUPid)); err == nil {
        var readBytes, writeBytes int64
        scanner := bufio.NewScanner(strings.NewReader(string(ioStats)))
        for scanner.Scan() {
            line := scanner.Text()
            if strings.HasPrefix(line, "read_bytes:") {
                fields := strings.Fields(line)
                if len(fields) >= 2 {
                    readBytes, _ = strconv.ParseInt(fields[1], 10, 64)
                }
            } else if strings.HasPrefix(line, "write_bytes:") {
                fields := strings.Fields(line)
                if len(fields) >= 2 {
                    writeBytes, _ = strconv.ParseInt(fields[1], 10, 64)
                }
            }
        }
        metrics.Disk = DiskMetrics{
            ReadBytes:  readBytes,
            WriteBytes: writeBytes,
            ReadOps:    0, // These will be calculated from differences
            WriteOps:   0,
            ReadSpeed:  0,
            WriteSpeed: 0,
        }
    }

    instanceDir := filepath.Join(m.baseDir, "disks", id)
    monitorSocket := filepath.Join(instanceDir, "qemu-monitor.sock")
    
    log.Printf("[NetworkMetrics] Starting network metrics collection for VPS %s", id)
    
    // Initialize network metrics
    metrics.Network = NetworkMetrics{
        RXBytes:   0,
        TXBytes:   0,
        RXPackets: 0,
        TXPackets: 0,
        RXSpeed:   0,
        TXSpeed:   0,
    }

    // First, get the list of PCI devices
    pciListCmd := `{ "execute": "qom-list", "arguments": {"path": "/machine/i440fx/pci.0"} }`
    if output, err := m.executeQMPCommand(monitorSocket, pciListCmd); err == nil {
        log.Printf("[NetworkMetrics] PCI devices list: %s", string(output))

        // Try to find our network device
        netDevCmd := `{ "execute": "qom-list", "arguments": {"path": "/machine/i440fx/pci.0/virtio-net-pci.0"} }`
        if netOutput, err := m.executeQMPCommand(monitorSocket, netDevCmd); err == nil {
            log.Printf("[NetworkMetrics] Network device properties: %s", string(netOutput))

            // Get the device properties
            statsCmd := `{ "execute": "qom-get", "arguments": {"path": "/machine/i440fx/pci.0/virtio-net-pci.0", "property": "host_features"} }`
            if statsOutput, err := m.executeQMPCommand(monitorSocket, statsCmd); err == nil {
                log.Printf("[NetworkMetrics] Network device stats: %s", string(statsOutput))
            }

            // Try alternative stats command
            altStatsCmd := `{ "execute": "query-rx-filter", "arguments": {"name": "net0"} }`
            if statsOutput, err := m.executeQMPCommand(monitorSocket, altStatsCmd); err == nil {
                log.Printf("[NetworkMetrics] RX filter stats: %s", string(statsOutput))
            }
        }

        // Try querying netdev directly
        netdevCmd := `{ "execute": "query-netdev" }`
        if netdevOutput, err := m.executeQMPCommand(monitorSocket, netdevCmd); err == nil {
            log.Printf("[NetworkMetrics] Netdev info: %s", string(netdevOutput))
        }
    }

    // If we still don't have stats, try reading from /proc
    if metrics.Network.RXBytes == 0 {
        m.mutex.RLock()
        vps, exists := m.instances[id]
        m.mutex.RUnlock()

        if exists && vps.QEMUPid > 0 {
            // Try to read network stats from /proc/[pid]/net/dev
            if data, err := os.ReadFile(fmt.Sprintf("/proc/%d/net/dev", vps.QEMUPid)); err == nil {
                scanner := bufio.NewScanner(bytes.NewReader(data))
                for scanner.Scan() {
                    line := scanner.Text()
                    if strings.Contains(line, "eth0:") || strings.Contains(line, "ens3:") {
                        fields := strings.Fields(line)
                        if len(fields) >= 17 {
                            metrics.Network.RXBytes, _ = strconv.ParseInt(fields[1], 10, 64)
                            metrics.Network.RXPackets, _ = strconv.ParseInt(fields[2], 10, 64)
                            metrics.Network.TXBytes, _ = strconv.ParseInt(fields[9], 10, 64)
                            metrics.Network.TXPackets, _ = strconv.ParseInt(fields[10], 10, 64)
                            log.Printf("[NetworkMetrics] Found network stats in /proc/net/dev")
                            break
                        }
                    }
                }
            }
        }
    }

    // Calculate speeds using the metrics cache
    m.metricsMutex.Lock()
    cache, exists := m.metricsCache[id]
    if exists && !cache.LastUpdate.IsZero() {
        duration := metrics.Time.Sub(cache.LastUpdate).Seconds()
        if duration > 0 {
            metrics.Network.RXSpeed = float64(metrics.Network.RXBytes-cache.LastNetStats.RXBytes) / duration
            metrics.Network.TXSpeed = float64(metrics.Network.TXBytes-cache.LastNetStats.TXBytes) / duration
            log.Printf("[NetworkMetrics] Calculated speeds - RX: %.2f bytes/sec, TX: %.2f bytes/sec",
                metrics.Network.RXSpeed, metrics.Network.TXSpeed)
        }
    }
    m.metricsMutex.Unlock()

    log.Printf("[NetworkMetrics] Final metrics for VPS %s:", id)
    log.Printf("[NetworkMetrics] RX Bytes: %d", metrics.Network.RXBytes)
    log.Printf("[NetworkMetrics] TX Bytes: %d", metrics.Network.TXBytes)
    log.Printf("[NetworkMetrics] RX Packets: %d", metrics.Network.RXPackets)
    log.Printf("[NetworkMetrics] TX Packets: %d", metrics.Network.TXPackets)

    return metrics, nil
}

func (m *VPSManager) executeQMPCommand(socket, command string) ([]byte, error) {
    log.Printf("[QMP] Connecting to socket: %s", socket)
    
    conn, err := net.Dial("unix", socket)
    if err != nil {
        log.Printf("[QMP] Failed to connect to socket: %v", err)
        return nil, fmt.Errorf("failed to connect to QMP socket: %v", err)
    }
    defer conn.Close()

    // Read the greeting
    greeting := make([]byte, 1024)
    n, err := conn.Read(greeting)
    if err != nil {
        log.Printf("[QMP] Failed to read greeting: %v", err)
        return nil, fmt.Errorf("failed to read QMP greeting: %v", err)
    }
    log.Printf("[QMP] Received greeting: %s", string(greeting[:n]))

    // First, switch to JSON mode
    jsonMode := `{ "execute": "qmp_capabilities" }` + "\n"
    if _, err := conn.Write([]byte(jsonMode)); err != nil {
        log.Printf("[QMP] Failed to send JSON mode command: %v", err)
        return nil, fmt.Errorf("failed to send JSON mode command: %v", err)
    }

    // Read and discard the response
    buf := make([]byte, 1024)
    if _, err := conn.Read(buf); err != nil {
        log.Printf("[QMP] Failed to read JSON mode response: %v", err)
        return nil, fmt.Errorf("failed to read JSON mode response: %v", err)
    }

    // Send the actual command with a newline
    fullCommand := command + "\n"
    log.Printf("[QMP] Sending command: %s", command)
    if _, err := conn.Write([]byte(fullCommand)); err != nil {
        log.Printf("[QMP] Failed to send command: %v", err)
        return nil, fmt.Errorf("failed to send command: %v", err)
    }

    // Read the response with a larger buffer
    response := make([]byte, 4096)
    n, err = conn.Read(response)
    if err != nil {
        log.Printf("[QMP] Failed to read command response: %v", err)
        return nil, fmt.Errorf("failed to read command response: %v", err)
    }

    // Try to find the complete JSON response
    respStr := string(response[:n])
    log.Printf("[QMP] Raw response: %s", respStr)

    // Look for complete JSON object
    start := strings.Index(respStr, "{")
    end := strings.LastIndex(respStr, "}")
    
    if start == -1 || end == -1 || start > end {
        log.Printf("[QMP] Invalid JSON response format")
        return nil, fmt.Errorf("invalid JSON response format")
    }

    jsonResponse := respStr[start:end+1]
    log.Printf("[QMP] Extracted JSON: %s", jsonResponse)
    
    return []byte(jsonResponse), nil
}


func (m *VPSManager) updateMetricsCache(id string, metrics *ResourceMetrics) {
    m.metricsMutex.Lock()
    defer m.metricsMutex.Unlock()

    cache, exists := m.metricsCache[id]
    if !exists {
        cache = &MetricsCache{
            MetricsHistory: make([]ResourceMetrics, 0, 300), // Store 10 minutes of data at 2s intervals
        }
        m.metricsCache[id] = cache
    }

    // Calculate speeds based on previous measurements
    if !cache.LastUpdate.IsZero() {
        duration := metrics.Time.Sub(cache.LastUpdate).Seconds()
        if duration > 0 {
            // Calculate disk speeds
            metrics.Disk.ReadSpeed = float64(metrics.Disk.ReadBytes-cache.LastDiskStats.ReadBytes) / duration
            metrics.Disk.WriteSpeed = float64(metrics.Disk.WriteBytes-cache.LastDiskStats.WriteBytes) / duration

            // Calculate network speeds
            metrics.Network.RXSpeed = float64(metrics.Network.RXBytes-cache.LastNetStats.RXBytes) / duration
            metrics.Network.TXSpeed = float64(metrics.Network.TXBytes-cache.LastNetStats.TXBytes) / duration
        }
    }

    // Update cache
    cache.LastUpdate = metrics.Time
    cache.LastDiskStats = metrics.Disk
    cache.LastNetStats = metrics.Network
    
    // Add to history and maintain window
    cache.MetricsHistory = append(cache.MetricsHistory, *metrics)
    if len(cache.MetricsHistory) > 300 {
        cache.MetricsHistory = cache.MetricsHistory[1:]
    }
}

// Add new HTTP handler
func (m *VPSManager) handleGetMetrics(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    id := r.URL.Query().Get("id")
    if id == "" {
        http.Error(w, "Missing VPS ID", http.StatusBadRequest)
        return
    }

    m.metricsMutex.RLock()
    cache, exists := m.metricsCache[id]
    m.metricsMutex.RUnlock()

    if !exists {
        http.Error(w, "No metrics available for this VPS", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(cache.MetricsHistory)
}



func (m *VPSManager) parseCPUMetrics(data []byte) CPUMetrics {
    var cpuMetrics CPUMetrics
    
    // Example JSON response from QEMU:
    // [{"CPU":0,"current":true,"halted":false,"qom_path":"/machine/unattached/device[0]","thread_id":123},...]
    type CPUInfo struct {
        CPU       int  `json:"CPU"`
        Current   bool `json:"current"`
        Halted    bool `json:"halted"`
        ThreadID  int  `json:"thread_id"`
    }
    
    var cpuInfos []CPUInfo
    if err := json.Unmarshal(data, &cpuInfos); err != nil {
        return cpuMetrics
    }

    // Get CPU usage by checking /proc/[thread_id]/stat for each CPU
    var totalUsage float64
    for _, cpu := range cpuInfos {
        if cpu.ThreadID > 0 {
            usage := getThreadCPUUsage(cpu.ThreadID)
            totalUsage += usage
        }
    }

    // Average the usage across all CPUs
    if len(cpuInfos) > 0 {
        cpuMetrics.Usage = totalUsage / float64(len(cpuInfos))
    }

    return cpuMetrics
}

func (m *VPSManager) parseMemoryMetrics(data []byte) MemoryMetrics {
    var memMetrics MemoryMetrics
    
    // Example JSON response from QEMU:
    // {"base-memory": 4294967296, "plugged-memory": 0}
    type MemInfo struct {
        BaseMemory    int64 `json:"base-memory"`
        PluggedMemory int64 `json:"plugged-memory"`
    }
    
    var memInfo MemInfo
    if err := json.Unmarshal(data, &memInfo); err != nil {
        return memMetrics
    }

    memMetrics.Total = memInfo.BaseMemory + memInfo.PluggedMemory
    
    // Try to get actual memory usage from balloon device
    if balloonData, err := m.executeQMPCommand(filepath.Join(m.baseDir, "monitor.sock"), "query-balloon"); err == nil {
        type BalloonInfo struct {
            Actual int64 `json:"actual"`
        }
        var balloonInfo BalloonInfo
        if err := json.Unmarshal(balloonData, &balloonInfo); err == nil {
            memMetrics.Used = balloonInfo.Actual
        }
    }

    return memMetrics
}

func (m *VPSManager) parseDiskMetrics(data []byte) DiskMetrics {
    var diskMetrics DiskMetrics
    
    // Example JSON response from QEMU:
    // [{"device":"drive-virtio-disk0","stats":{"rd_bytes":1234,"wr_bytes":5678,"rd_operations":10,"wr_operations":20}}]
    type BlockStats struct {
        Stats struct {
            ReadBytes    int64 `json:"rd_bytes"`
            WriteBytes   int64 `json:"wr_bytes"`
            ReadOps     int64 `json:"rd_operations"`
            WriteOps    int64 `json:"wr_operations"`
        } `json:"stats"`
    }
    
    var blockInfos []BlockStats
    if err := json.Unmarshal(data, &blockInfos); err != nil {
        return diskMetrics
    }

    // Sum up stats from all block devices
    for _, block := range blockInfos {
        diskMetrics.ReadBytes += block.Stats.ReadBytes
        diskMetrics.WriteBytes += block.Stats.WriteBytes
        diskMetrics.ReadOps += block.Stats.ReadOps
        diskMetrics.WriteOps += block.Stats.WriteOps
    }

    return diskMetrics
}

func (m *VPSManager) parseNetworkMetrics(data []byte) NetworkMetrics {
    var netMetrics NetworkMetrics
    
    // Example JSON response from QEMU:
    // [{"name":"net0","stats":{"rx_bytes":1234,"tx_bytes":5678,"rx_packets":10,"tx_packets":20}}]
    type NetStats struct {
        Stats struct {
            RXBytes     int64 `json:"rx_bytes"`
            TXBytes     int64 `json:"tx_bytes"`
            RXPackets   int64 `json:"rx_packets"`
            TXPackets   int64 `json:"tx_packets"`
        } `json:"stats"`
    }
    
    var netInfos []NetStats
    if err := json.Unmarshal(data, &netInfos); err != nil {
        return netMetrics
    }

    // Sum up stats from all network interfaces
    for _, net := range netInfos {
        netMetrics.RXBytes += net.Stats.RXBytes
        netMetrics.TXBytes += net.Stats.TXBytes
        netMetrics.RXPackets += net.Stats.RXPackets
        netMetrics.TXPackets += net.Stats.TXPackets
    }

    return netMetrics
}

// Helper function to get CPU usage for a specific thread
func getThreadCPUUsage(threadID int) float64 {
    data, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", threadID))
    if err != nil {
        return 0
    }

    fields := strings.Fields(string(data))
    if len(fields) < 15 {
        return 0
    }

    // Fields 14 and 15 are utime and stime (user and system CPU time)
    utime, _ := strconv.ParseFloat(fields[13], 64)
    stime, _ := strconv.ParseFloat(fields[14], 64)
    
    // Calculate CPU usage percentage based on total CPU time
    totalCPUTime := utime + stime
    
    // Get process uptime
    if uptimeData, err := os.ReadFile("/proc/uptime"); err == nil {
        uptime, _ := strconv.ParseFloat(strings.Fields(string(uptimeData))[0], 64)
        if uptime > 0 {
            // Calculate percentage based on total CPU time and uptime
            // Multiply by 100 for percentage and divide by number of CPUs
            numCPUs := float64(runtime.NumCPU())
            return (totalCPUTime / (uptime * numCPUs)) * 100
        }
    }

    return 0
}



func main() {
    log.Printf("Verifying system requirements...")
    if err := verifySystemRequirements(); err != nil {
        log.Fatal(err)
    }

    apiKey := os.Getenv("API_KEY")
    if apiKey == "" {
        log.Fatal("API_KEY environment variable is required")
    }

    baseDir := "/var/lib/vps-service"
    for _, dir := range []string{
        baseDir,
        filepath.Join(baseDir, "base"),
        filepath.Join(baseDir, "disks"),
        filepath.Join(baseDir, "logs"),
    } {
        if err := os.MkdirAll(dir, 0755); err != nil {
            log.Fatalf("Failed to create directory %s: %v", dir, err)
        }
    }

    manager, err := NewVPSManager(baseDir)
    if err != nil {
        log.Fatal(err)
    }

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        sig := <-sigChan
        log.Printf("Received signal %v, starting cleanup...", sig)
        manager.cleanup()
        log.Println("Cleanup completed, exiting...")
        os.Exit(0)
    }()

    defer func() {
        if r := recover(); r != nil {
            log.Printf("Panic occurred: %v", r)
            manager.cleanup()
            panic(r)
        }
    }()





    apiMux := http.NewServeMux()
    apiMux.HandleFunc("/api/vps/create", manager.handleCreateVPS)
    apiMux.HandleFunc("/api/vps/list", manager.handleListVPS)
    apiMux.HandleFunc("/api/vps/get", manager.handleGetVPS)
    apiMux.HandleFunc("/api/vps/progress", manager.handleGetProgress)
    apiMux.HandleFunc("/api/images/list", manager.handleListImages)
    apiMux.HandleFunc("/api/vps/delete", manager.handleDeleteVPS)
    apiMux.HandleFunc("/api/vps/restart", manager.handleRestartVPS)
    apiMux.HandleFunc("/api/vps/start", manager.handleStartVPS)
    apiMux.HandleFunc("/api/vps/metrics", manager.handleGetMetrics)
    apiMux.HandleFunc("/api/vps/stop", manager.handleStopVPS)
    apiMux.HandleFunc("/api/templates/list", manager.handleListTemplates)
    
    http.Handle("/api/", NewAuthMiddleware(apiKey, apiMux))
    http.Handle("/novnc/", http.StripPrefix("/novnc/", http.FileServer(http.Dir("/usr/share/novnc"))))

    log.Printf("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}