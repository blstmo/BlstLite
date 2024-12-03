// main.go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
)

const (
    UBUNTU_IMAGE_URL = "https://cloud-images.ubuntu.com/releases/22.04/release/ubuntu-22.04-server-cloudimg-amd64.img"
    BASE_IMAGE_PATH  = "/var/lib/vps-service/base/ubuntu-22.04.qcow2"
    VPS_LIFETIME     = 15 * time.Minute
    RAM_SIZE        = 4096  // 4GB
    DISK_SIZE       = 50    // 50GB
    DOWNLOAD_SPEED  = 50    // 50Mbps
    UPLOAD_SPEED    = 15    // 15Mbps
    SSH_PORT_START  = 2200  // Starting port for SSH forwarding
)

type VPS struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Status      string    `json:"status"`
    QEMUPid     int       `json:"qemu_pid,omitempty"`
    VNCPort     int       `json:"vnc_port"`
    SSHPort     int       `json:"ssh_port"` 
    CreatedAt   time.Time `json:"created_at"`
    ExpiresAt   time.Time `json:"expires_at"`
    ImagePath   string    `json:"image_path"`
    Password    string    `json:"password"`
}

type VPSManager struct {
    instances    map[string]*VPS
    mutex        sync.RWMutex
    nextVNCPort  int
    nextSSHPort  int       // Added to track SSH ports
    baseDir      string
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

    if _, err := os.Stat(BASE_IMAGE_PATH); os.IsNotExist(err) {
        if err := downloadAndPrepareBaseImage(); err != nil {
            return nil, err
        }
    }

    return &VPSManager{
        instances:    make(map[string]*VPS),
        nextVNCPort:  5900,
        nextSSHPort:  SSH_PORT_START,
        baseDir:      baseDir,
    }, nil
}

func downloadAndPrepareBaseImage() error {
    log.Printf("Starting base image preparation")
    
    tmpDir := "/tmp/vps-download"
    if err := os.MkdirAll(tmpDir, 0755); err != nil {
        return fmt.Errorf("failed to create temp directory: %v", err)
    }
    defer os.RemoveAll(tmpDir)

    tmpImagePath := filepath.Join(tmpDir, "ubuntu-22.04.img")
    
    log.Printf("Downloading Ubuntu cloud image to %s", tmpImagePath)
    downloadCmd := exec.Command("wget",
        "--progress=bar:force",
        "-O", tmpImagePath,
        UBUNTU_IMAGE_URL)
    downloadCmd.Stdout = os.Stdout
    downloadCmd.Stderr = os.Stderr
    
    if err := downloadCmd.Run(); err != nil {
        return fmt.Errorf("failed to download Ubuntu image: %v", err)
    }

    baseDir := filepath.Dir(BASE_IMAGE_PATH)
    if err := os.MkdirAll(baseDir, 0755); err != nil {
        return fmt.Errorf("failed to create base directory: %v", err)
    }

    log.Printf("Converting and resizing image to %dG", DISK_SIZE)
    convertCmd := exec.Command("qemu-img", "convert",
        "-f", "raw",
        "-O", "qcow2",
        tmpImagePath,
        BASE_IMAGE_PATH)
    
    if output, err := convertCmd.CombinedOutput(); err != nil {
        return fmt.Errorf("failed to convert image: %v, output: %s", err, string(output))
    }

    // Resize the image
    resizeCmd := exec.Command("qemu-img", "resize", BASE_IMAGE_PATH, fmt.Sprintf("%dG", DISK_SIZE))
    if output, err := resizeCmd.CombinedOutput(); err != nil {
        return fmt.Errorf("failed to resize image: %v, output: %s", err, string(output))
    }

    if err := os.Chmod(BASE_IMAGE_PATH, 0644); err != nil {
        return fmt.Errorf("failed to set image permissions: %v", err)
    }

    log.Printf("Base image preparation completed successfully")
    return nil
}

func createCloudInitISO(path string, rootPassword string) error {
    tmpDir, err := os.MkdirTemp("", "cloud-init")
    if err != nil {
        return err
    }
    defer os.RemoveAll(tmpDir)

    // Updated cloud-init configuration with more explicit password settings
    userData := fmt.Sprintf(`#cloud-config
users:
  - name: root
    lock_passwd: false
    passwd: "%s"
    hashed_passwd: null
    ssh_pwauth: true

# Enable password authentication in SSH
ssh_pwauth: true

# Disable SSH root lockout
disable_root: false

# More direct password configuration
chpasswd:
  list: |
     root:%s
  expire: false

# Make sure SSH password auth is enabled in sshd_config
write_files:
  - path: /etc/ssh/sshd_config.d/99-cloud-init.conf
    content: |
        PasswordAuthentication yes
        PermitRootLogin yes

runcmd:
  - systemctl restart ssh
  - echo "root:%s" | chpasswd
`, rootPassword, rootPassword, rootPassword)

    if err := os.WriteFile(filepath.Join(tmpDir, "user-data"), []byte(userData), 0644); err != nil {
        return err
    }

    metaData := "instance-id: 1\nlocal-hostname: ubuntu-vps\n"
    if err := os.WriteFile(filepath.Join(tmpDir, "meta-data"), []byte(metaData), 0644); err != nil {
        return err
    }

    // Create a new ISO with the cloud-init config
    cmd := exec.Command("genisoimage", "-output", path, "-volid", "cidata", "-joliet", "-rock",
        filepath.Join(tmpDir, "user-data"), filepath.Join(tmpDir, "meta-data"))

    if output, err := cmd.CombinedOutput(); err != nil {
        return fmt.Errorf("failed to create ISO: %v, output: %s", err, string(output))
    }

    log.Printf("Created cloud-init ISO at %s with password configuration", path)
    return nil
}

func startWebsockifyProxy(vncPort int) error {
    // Calculate websocket port (6900 + offset)
    wsPort := vncPort + 1000  // So 5900 -> 6900, 5901 -> 6901, etc.

    // Kill any existing websockify processes for this port
    killCmd := exec.Command("pkill", "-f", fmt.Sprintf("websockify.*:%d", wsPort))
    killCmd.Run() // Ignore errors as process might not exist

    time.Sleep(time.Second)

    logFile, err := os.Create(fmt.Sprintf("/tmp/websockify_%d.log", wsPort))
    if err != nil {
        return fmt.Errorf("failed to create websockify log file: %v", err)
    }
    defer logFile.Close()

    // Start websockify with more options
    cmd := exec.Command("websockify", 
        "--verbose",
        fmt.Sprintf("%d", wsPort),                    // Listen on 6900+ port
        fmt.Sprintf("localhost:%d", vncPort),         // Connect to VNC on 5900+ port
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

    // Check if process is running
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

func (m *VPSManager) CreateVPS(name string) (*VPS, error) {
    m.mutex.Lock()
    defer m.mutex.Unlock()

    log.Printf("Starting VPS creation process for: %s", name)

    password, err := generatePassword()
    if err != nil {
        return nil, fmt.Errorf("failed to generate password: %v", err)
    }

    vps := &VPS{
        ID:          uuid.New().String(),
        Name:        name,
        Status:      "creating",
        VNCPort:     m.nextVNCPort,
        SSHPort:     m.nextSSHPort,  // Assign SSH port
        CreatedAt:   time.Now(),
        ExpiresAt:   time.Now().Add(VPS_LIFETIME),
        Password:    password,
    }
    m.nextVNCPort++
    m.nextSSHPort++

    instanceDir := filepath.Join(m.baseDir, "disks", vps.ID)
    if err := os.MkdirAll(instanceDir, 0755); err != nil {
        return nil, fmt.Errorf("failed to create instance directory: %v", err)
    }

    vps.ImagePath = filepath.Join(instanceDir, "disk.qcow2")
    log.Printf("Creating disk image at: %s", vps.ImagePath)

    createDisk := exec.Command("qemu-img", "create",
        "-f", "qcow2",
        "-F", "qcow2",
        "-b", BASE_IMAGE_PATH,
        vps.ImagePath)
    
    if output, err := createDisk.CombinedOutput(); err != nil {
        os.RemoveAll(instanceDir)
        return nil, fmt.Errorf("failed to create disk: %v, output: %s", err, string(output))
    }

    cloudInitPath := filepath.Join(instanceDir, "cloud-init.iso")
    if err := createCloudInitISO(cloudInitPath, vps.Password); err != nil {
        os.RemoveAll(instanceDir)
        return nil, fmt.Errorf("failed to create cloud-init ISO: %v", err)
    }

    pidFile := filepath.Join(instanceDir, "qemu.pid")

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
        "-pidfile", pidFile,
        "-daemonize",
        "-enable-kvm",
    }

    cmd := exec.Command("qemu-system-x86_64", args...)
    
    logFile, err := os.Create(filepath.Join(m.baseDir, "logs", fmt.Sprintf("%s.log", vps.ID)))
    if err != nil {
        os.RemoveAll(instanceDir)
        return nil, fmt.Errorf("failed to create log file: %v", err)
    }
    defer logFile.Close()

    cmd.Stdout = logFile
    cmd.Stderr = logFile

    log.Printf("Starting QEMU with command: %v", cmd.Args)
    if err := cmd.Run(); err != nil {
        os.RemoveAll(instanceDir)
        return nil, fmt.Errorf("failed to start QEMU: %v", err)
    }

    // Wait for PID file
    var pid int
    for i := 0; i < 10; i++ {
        time.Sleep(500 * time.Millisecond)
        pidBytes, err := os.ReadFile(pidFile)
        if err == nil {
            if _, err := fmt.Sscanf(string(pidBytes), "%d", &pid); err == nil {
                break
            }
        }
        if i == 9 {
            os.RemoveAll(instanceDir)
            return nil, fmt.Errorf("failed to get QEMU PID")
        }
    }

    // Verify the QEMU process is running and valid
    if err := checkProcess(pid); err != nil {
        os.RemoveAll(instanceDir)
        return nil, fmt.Errorf("QEMU process verification failed: %v", err)
    }

    vps.QEMUPid = pid
    vps.Status = "running"
    m.instances[vps.ID] = vps

    // Start websockify proxy for VNC access
    if err := startWebsockifyProxy(vps.VNCPort); err != nil {
        log.Printf("Warning: Failed to start websockify proxy: %v", err)
        // Don't fail the VPS creation if websockify fails, just log the error
    }

    // Schedule cleanup after lifetime expires
    go m.scheduleCleanup(vps)

    log.Printf("VPS %s (ID: %s) successfully created with PID %d, SSH port %d", 
        vps.Name, vps.ID, vps.QEMUPid, vps.SSHPort)
    return vps, nil
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

    // Stop websockify first
    if err := stopWebsockifyProxy(vps.VNCPort); err != nil {
        log.Printf("Warning: Failed to stop websockify: %v", err)
    }

    // Then stop QEMU
    if vps.QEMUPid > 0 {
        if proc, err := os.FindProcess(vps.QEMUPid); err == nil {
            proc.Kill()
        }
    }

    // Cleanup files
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
        Name string `json:"name"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    vps, err := m.CreateVPS(req.Name)
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
            
            // Stop websockify first
            if err := stopWebsockifyProxy(vps.VNCPort); err != nil {
                log.Printf("Warning: Failed to stop websockify for VPS %s: %v", id, err)
            }

            // Kill QEMU process
            if vps.QEMUPid > 0 {
                if proc, err := os.FindProcess(vps.QEMUPid); err == nil {
                    log.Printf("Killing QEMU process %d for VPS %s", vps.QEMUPid, id)
                    proc.Kill()
                    proc.Wait() // Wait for the process to actually terminate
                }
            }

            // Cleanup files
            instanceDir := filepath.Join(m.baseDir, "disks", id)
            if err := os.RemoveAll(instanceDir); err != nil {
                log.Printf("Warning: Failed to remove instance directory for VPS %s: %v", id, err)
            }

            log.Printf("Successfully cleaned up VPS %s", id)
        }(id, vps)
    }

    // Wait for all cleanup goroutines to complete
    wg.Wait()
    log.Println("All VPS instances have been cleaned up")
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

    // Set up signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    // Run cleanup when program exits
    go func() {
        sig := <-sigChan
        log.Printf("Received signal %v, starting cleanup...", sig)
        manager.cleanup()
        log.Println("Cleanup completed, exiting...")
        os.Exit(0)
    }()

    // Ensure cleanup runs even on panic
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Panic occurred: %v", r)
            manager.cleanup()
            panic(r) // Re-panic after cleanup
        }
    }()

    apiMux := http.NewServeMux()
    apiMux.HandleFunc("/api/vps/create", manager.handleCreateVPS)
    apiMux.HandleFunc("/api/vps/list", manager.handleListVPS)
    apiMux.HandleFunc("/api/vps/get", manager.handleGetVPS)
    apiMux.HandleFunc("/api/vps/delete", manager.handleDeleteVPS)

    http.Handle("/api/", NewAuthMiddleware(apiKey, apiMux))
    http.Handle("/novnc/", http.StripPrefix("/novnc/", http.FileServer(http.Dir("/usr/share/novnc"))))

    log.Printf("Server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}