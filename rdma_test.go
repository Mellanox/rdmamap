package rdmamap

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestGetRdmaDevices(t *testing.T) {
	rdmaDevices := GetRdmaDeviceList()
	t.Log("Devices: ", rdmaDevices)
}

func TestRdmaCharDevices(t *testing.T) {
	rdmaDevices := GetRdmaDeviceList()
	t.Log("Devices: ", rdmaDevices)

	for _, dev := range rdmaDevices {
		charDevices := GetRdmaCharDevices(dev)
		fmt.Printf("Rdma device: = %s", dev)
		t.Log(" Char devices: = ", charDevices)
	}
}

// createFakeSysfsDevice creates a subdirectory under classDir named entryName
// with an "ibdev" file containing rdmaDeviceName.
func createFakeSysfsDevice(t *testing.T, classDir, entryName, rdmaDeviceName string) {
	t.Helper()
	dir := filepath.Join(classDir, entryName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ibdev"), []byte(rdmaDeviceName+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestGetCharDevices_MultipleDevices(t *testing.T) {
	classDir := t.TempDir()
	createFakeSysfsDevice(t, classDir, "umad0", "mlx5_0")
	createFakeSysfsDevice(t, classDir, "umad1", "mlx5_0")

	devices := getCharDevices("mlx5_0", classDir, "umad")
	sort.Strings(devices)

	expected := []string{
		"/dev/infiniband/umad0",
		"/dev/infiniband/umad1",
	}
	if len(devices) != len(expected) {
		t.Fatalf("expected %d devices, got %d: %v", len(expected), len(devices), devices)
	}
	for i := range expected {
		if devices[i] != expected[i] {
			t.Errorf("device[%d] = %q, want %q", i, devices[i], expected[i])
		}
	}
}

func TestGetCharDevices_FiltersByPrefix(t *testing.T) {
	classDir := t.TempDir()
	createFakeSysfsDevice(t, classDir, "umad0", "mlx5_0")
	createFakeSysfsDevice(t, classDir, "issm0", "mlx5_0")

	devices := getCharDevices("mlx5_0", classDir, "umad")
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d: %v", len(devices), devices)
	}
	if devices[0] != "/dev/infiniband/umad0" {
		t.Errorf("got %q, want /dev/infiniband/umad0", devices[0])
	}
}

func TestGetCharDevices_FiltersByRdmaDevice(t *testing.T) {
	classDir := t.TempDir()
	createFakeSysfsDevice(t, classDir, "umad0", "mlx5_0")
	createFakeSysfsDevice(t, classDir, "umad1", "mlx5_1")

	devices := getCharDevices("mlx5_0", classDir, "umad")
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d: %v", len(devices), devices)
	}
	if devices[0] != "/dev/infiniband/umad0" {
		t.Errorf("got %q, want /dev/infiniband/umad0", devices[0])
	}
}

func TestGetCharDevices_NonexistentDir(t *testing.T) {
	devices := getCharDevices("mlx5_0", "/nonexistent/path", "umad")
	if devices != nil {
		t.Errorf("expected nil, got %v", devices)
	}
}

func TestGetCharDevices_EmptyDir(t *testing.T) {
	classDir := t.TempDir()
	devices := getCharDevices("mlx5_0", classDir, "umad")
	if len(devices) != 0 {
		t.Errorf("expected empty slice, got %v", devices)
	}
}

func TestGetCharDevices_ReturnsNilWhenNoMatch(t *testing.T) {
	classDir := t.TempDir()
	createFakeSysfsDevice(t, classDir, "issm0", "mlx5_0")

	devices := getCharDevices("mlx5_0", classDir, "umad")
	if len(devices) != 0 {
		t.Errorf("expected no devices, got %v", devices)
	}
}

func TestGetRdmaCharDevices_MultiPort(t *testing.T) {
	ucmDir := t.TempDir()
	umadDir := t.TempDir()
	uverbsDir := t.TempDir()

	origUcmDir := RdmaIbUcmDir
	origUmadDir := RdmaUmadDir
	origUverbsDir := RdmaUverbsDir
	origUcmDevice := RdmaUcmDevice
	defer func() {
		RdmaIbUcmDir = origUcmDir
		RdmaUmadDir = origUmadDir
		RdmaUverbsDir = origUverbsDir
		RdmaUcmDevice = origUcmDevice
	}()
	RdmaIbUcmDir = ucmDir
	RdmaUmadDir = umadDir
	RdmaUverbsDir = uverbsDir
	RdmaUcmDevice = "/nonexistent/rdma_cm"

	createFakeSysfsDevice(t, ucmDir, "ucm0", "mlx5_0")
	createFakeSysfsDevice(t, umadDir, "issm0", "mlx5_0")
	createFakeSysfsDevice(t, umadDir, "issm1", "mlx5_0")
	createFakeSysfsDevice(t, umadDir, "umad0", "mlx5_0")
	createFakeSysfsDevice(t, umadDir, "umad1", "mlx5_0")
	createFakeSysfsDevice(t, uverbsDir, "uverbs0", "mlx5_0")

	devices := GetRdmaCharDevices("mlx5_0")
	sort.Strings(devices)

	expected := []string{
		"/dev/infiniband/issm0",
		"/dev/infiniband/issm1",
		"/dev/infiniband/ucm0",
		"/dev/infiniband/umad0",
		"/dev/infiniband/umad1",
		"/dev/infiniband/uverbs0",
	}
	if len(devices) != len(expected) {
		t.Fatalf("expected %d devices, got %d: %v", len(expected), len(devices), devices)
	}
	for i := range expected {
		if devices[i] != expected[i] {
			t.Errorf("device[%d] = %q, want %q", i, devices[i], expected[i])
		}
	}
}

func TestGetRdmaUcmDevices_Found(t *testing.T) {
	tmpDir := t.TempDir()
	ucmFile := filepath.Join(tmpDir, "rdma_cm")
	if err := os.WriteFile(ucmFile, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	origUcmDevice := RdmaUcmDevice
	defer func() { RdmaUcmDevice = origUcmDevice }()
	RdmaUcmDevice = ucmFile

	devices := getRdmaUcmDevices()
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d: %v", len(devices), devices)
	}
	if devices[0] != ucmFile {
		t.Errorf("got %q, want %q", devices[0], ucmFile)
	}
}

func TestGetRdmaUcmDevices_NotFound(t *testing.T) {
	origUcmDevice := RdmaUcmDevice
	defer func() { RdmaUcmDevice = origUcmDevice }()
	RdmaUcmDevice = "/nonexistent/rdma_cm"

	devices := getRdmaUcmDevices()
	if devices != nil {
		t.Errorf("expected nil, got %v", devices)
	}
}

func TestGetRdmaUcmDevices_WrongName(t *testing.T) {
	tmpDir := t.TempDir()
	wrongFile := filepath.Join(tmpDir, "not_rdma_cm")
	if err := os.WriteFile(wrongFile, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	origUcmDevice := RdmaUcmDevice
	defer func() { RdmaUcmDevice = origUcmDevice }()
	RdmaUcmDevice = wrongFile

	devices := getRdmaUcmDevices()
	if devices != nil {
		t.Errorf("expected nil, got %v", devices)
	}
}

func TestGetRdmaCharDevices_WithRdmaCm(t *testing.T) {
	ucmDir := t.TempDir()
	umadDir := t.TempDir()
	uverbsDir := t.TempDir()
	rdmaCmDir := t.TempDir()
	rdmaCmFile := filepath.Join(rdmaCmDir, "rdma_cm")
	if err := os.WriteFile(rdmaCmFile, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	origUcmDir := RdmaIbUcmDir
	origUmadDir := RdmaUmadDir
	origUverbsDir := RdmaUverbsDir
	origUcmDevice := RdmaUcmDevice
	defer func() {
		RdmaIbUcmDir = origUcmDir
		RdmaUmadDir = origUmadDir
		RdmaUverbsDir = origUverbsDir
		RdmaUcmDevice = origUcmDevice
	}()
	RdmaIbUcmDir = ucmDir
	RdmaUmadDir = umadDir
	RdmaUverbsDir = uverbsDir
	RdmaUcmDevice = rdmaCmFile

	createFakeSysfsDevice(t, umadDir, "umad0", "mlx5_0")
	createFakeSysfsDevice(t, uverbsDir, "uverbs0", "mlx5_0")

	devices := GetRdmaCharDevices("mlx5_0")
	sort.Strings(devices)

	expected := []string{
		rdmaCmFile,
		"/dev/infiniband/umad0",
		"/dev/infiniband/uverbs0",
	}
	sort.Strings(expected)

	if len(devices) != len(expected) {
		t.Fatalf("expected %d devices, got %d: %v", len(expected), len(devices), devices)
	}
	for i := range expected {
		if devices[i] != expected[i] {
			t.Errorf("device[%d] = %q, want %q", i, devices[i], expected[i])
		}
	}
}

func TestGetRdmaCharDevices_NoDevices(t *testing.T) {
	emptyDir := t.TempDir()

	origUcmDir := RdmaIbUcmDir
	origUmadDir := RdmaUmadDir
	origUverbsDir := RdmaUverbsDir
	origUcmDevice := RdmaUcmDevice
	defer func() {
		RdmaIbUcmDir = origUcmDir
		RdmaUmadDir = origUmadDir
		RdmaUverbsDir = origUverbsDir
		RdmaUcmDevice = origUcmDevice
	}()
	RdmaIbUcmDir = emptyDir
	RdmaUmadDir = emptyDir
	RdmaUverbsDir = emptyDir
	RdmaUcmDevice = "/nonexistent/rdma_cm"

	devices := GetRdmaCharDevices("mlx5_0")
	if len(devices) != 0 {
		t.Errorf("expected no devices, got %v", devices)
	}
}

func TestRdmaDeviceForNetdevice(_ *testing.T) {
	netdev := "ib0"
	rdmaDev, err := GetRdmaDeviceForNetdevice(netdev)
	if err == nil {
		fmt.Printf("netdev = %s, rdmadev = %s\n", netdev, rdmaDev)
	} else {
		fmt.Printf("rdma device not found for netdev = %s\n", netdev)
	}

	found := IsRDmaDeviceForNetdevice(netdev)
	fmt.Printf("rdma device %t for netdev = %s\n", found, netdev)

	netdev = "ens1f0"
	found = IsRDmaDeviceForNetdevice(netdev)
	fmt.Printf("rdma device %t for netdev = %s\n", found, netdev)

	netdev = loopBackIfName
	found = IsRDmaDeviceForNetdevice(netdev)
	fmt.Printf("rdma device %t for netdev = %s\n", found, netdev)
}

func TestRdmaDeviceStats(t *testing.T) {
	stats, err := GetRdmaSysfsAllPortsStats("mlx5_1")
	if err == nil {
		t.Log(stats)
	} else {
		t.Log("error is: ", err)
	}
}

func TestRdmaDeviceForPcidev(t *testing.T) {
	devs := GetRdmaDevicesForPcidev("0000:05:00.0")
	t.Log("rdma devs :", devs)
}

func TestRdmaDeviceForAuxdev(t *testing.T) {
	devs := GetRdmaDevicesForAuxdev("mlx5_core.sf.4")
	t.Log("rdma devs :", devs)
}
