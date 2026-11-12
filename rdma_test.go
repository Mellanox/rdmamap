package rdmamap

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

const fakeStatsRdmaDevice = "rdma0"

// createFakePortCounters creates <classDir>/rdma0/ports/<port>/<dirName>
// with one file per counter.
func createFakePortCounters(t *testing.T, classDir string, port int, dirName string, counters map[string]uint64) {
	t.Helper()
	dir := filepath.Join(classDir, fakeStatsRdmaDevice, RdmaPortsdir, fmt.Sprint(port), dirName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, value := range counters {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(fmt.Sprintf("%d\n", value)), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func setFakeRdmaClassDir(t *testing.T) string {
	t.Helper()
	classDir := t.TempDir()
	origClassDir := RdmaClassDir
	t.Cleanup(func() { RdmaClassDir = origClassDir })
	RdmaClassDir = classDir
	return classDir
}

func statsToMap(entries []RdmaStatEntry) map[string]uint64 {
	m := make(map[string]uint64, len(entries))
	for _, e := range entries {
		m[e.Name] = e.Value
	}
	return m
}

func checkPortStats(t *testing.T, got RdmaPortStats, port int, hwStats, stats map[string]uint64) {
	t.Helper()
	if got.Port != port {
		t.Errorf("Port = %d, want %d", got.Port, port)
	}
	if g := statsToMap(got.HwStats); !reflect.DeepEqual(g, hwStats) {
		t.Errorf("HwStats = %v, want %v", g, hwStats)
	}
	if g := statsToMap(got.Stats); !reflect.DeepEqual(g, stats) {
		t.Errorf("Stats = %v, want %v", g, stats)
	}
}

func TestGetRdmaSysfsAllStats(t *testing.T) {
	hwCounters := map[string]uint64{"rx_write_requests": 7, "out_of_sequence": 1}
	counters := map[string]uint64{"port_rcv_data": 100, "port_xmit_data": 200}

	tests := []struct {
		name       string
		hwCounters map[string]uint64
		counters   map[string]uint64
	}{
		{name: "hw_counters and counters", hwCounters: hwCounters, counters: counters},
		// e.g. drivers without process_mad (efa, irdma): no counters directory
		{name: "only hw_counters", hwCounters: hwCounters},
		// e.g. drivers without per port hw stats (mlx4 VF): no hw_counters directory
		{name: "only counters", counters: counters},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			classDir := setFakeRdmaClassDir(t)
			if tc.hwCounters != nil {
				createFakePortCounters(t, classDir, 2, RdmaHwCountersDir, tc.hwCounters)
			}
			if tc.counters != nil {
				createFakePortCounters(t, classDir, 2, RdmaCountersDir, tc.counters)
			}

			stats, err := GetRdmaSysfsAllStats(fakeStatsRdmaDevice, 2)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := func(m map[string]uint64) map[string]uint64 {
				if m == nil {
					return map[string]uint64{}
				}
				return m
			}
			checkPortStats(t, stats, 2, want(tc.hwCounters), want(tc.counters))
		})
	}
}

func TestGetRdmaSysfsAllPortsStats_MultiPortHwCountersOnly(t *testing.T) {
	classDir := setFakeRdmaClassDir(t)
	createFakePortCounters(t, classDir, 1, RdmaHwCountersDir, map[string]uint64{"rx_write_requests": 1})
	createFakePortCounters(t, classDir, 2, RdmaHwCountersDir, map[string]uint64{"rx_write_requests": 2})

	stats, err := GetRdmaSysfsAllPortsStats(fakeStatsRdmaDevice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stats.PortStats) != 2 {
		t.Fatalf("expected stats for 2 ports, got %d: %v", len(stats.PortStats), stats.PortStats)
	}
	sort.Slice(stats.PortStats, func(i, j int) bool { return stats.PortStats[i].Port < stats.PortStats[j].Port })
	for i, ps := range stats.PortStats {
		port := i + 1
		checkPortStats(t, ps, port, map[string]uint64{"rx_write_requests": uint64(port)}, map[string]uint64{})
	}
}
