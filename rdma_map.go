package rdmamap

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	netlink "github.com/vishvananda/netlink"
)

const (
	RdmaClassName = "infiniband"

	RdmaUcmFilePrefix = "ucm"

	RdmaIssmFilePrefix = "issm"
	RdmaUmadFilxPrefix = "umad"

	RdmaUverbsFilxPrefix = "uverbs"

	RdmaGidAttrDir     = "gid_attrs" //nolint:revive
	RdmaGidAttrNdevDir = "ndevs"     //nolint:revive
	RdmaPortsdir       = "ports"
	RdmaGidsDir        = "gids"

	RdmaNodeGuidFile = "node_guid" //nolint:revive

	RdmaCountersDir   = "counters"
	RdmaHwCountersDir = "hw_counters"

	// For local usage
	prevDir        = ".."
	nibbleBitSize  = 4
	loopBackIfName = "lo"

	// GID format constants
	gidParts            = 8 // Number of colon-separated parts in a GID
	portGUIDStartIndex  = 4 // Port GUID starts at index 4 in the GID parts
	gidPartLength       = 4 // Each GID part is 4 hex characters
	hexBytePairLength   = 2 // 2 hex characters represent 1 byte

	ReadOnlyPermissions = 0444
)

var (
	RdmaClassDir = "/sys/class/infiniband"
	RdmaIbUcmDir = "/sys/class/infiniband_cm"

	RdmaUmadDir = "/sys/class/infiniband_mad"

	RdmaUverbsDir = "/sys/class/infiniband_verbs"

	RdmaUcmDevice = "/dev/infiniband/rdma_cm"
	RdmaDeviceDir = "/dev/infiniband"

	PciDevDir = "/sys/bus/pci/devices"
	AuxDevDir = "/sys/bus/auxiliary/devices"
)

// GetRdmaDeviceList Returns a list of rdma device names
//
//nolint:prealloc
func GetRdmaDeviceList() []string {
	var rdmaDevices []string
	fd, err := os.Open(RdmaClassDir)
	if err != nil {
		return nil
	}
	defer fd.Close()

	fileInfos, err := fd.Readdir(-1)
	if err != nil {
		return nil
	}

	for i := range fileInfos {
		if fileInfos[i].IsDir() {
			continue
		}
		rdmaDevices = append(rdmaDevices, fileInfos[i].Name())
	}
	return rdmaDevices
}

func isDirForRdmaDevice(rdmaDeviceName, dirName string) bool {
	fileName := filepath.Join(dirName, "ibdev")

	fd, err := os.OpenFile(fileName, os.O_RDONLY, ReadOnlyPermissions)
	if err != nil {
		return false
	}
	defer fd.Close()

	if _, err = fd.Seek(0, io.SeekStart); err != nil {
		return false
	}

	data, err := io.ReadAll(fd)
	if err != nil {
		return false
	}
	return strings.Trim(string(data), "\n") == rdmaDeviceName
}

func getCharDevices(rdmaDeviceName, classDir, charDevPrefix string) []string {
	fd, err := os.Open(classDir)
	if err != nil {
		return nil
	}
	defer fd.Close()
	fileInfos, err := fd.Readdir(-1)
	if err != nil {
		return nil
	}

	devices := make([]string, 0, len(fileInfos))
	for i := range fileInfos {
		if fileInfos[i].Name() == "." || fileInfos[i].Name() == prevDir {
			continue
		}
		if !strings.Contains(fileInfos[i].Name(), charDevPrefix) {
			continue
		}
		dirName := filepath.Join(classDir, fileInfos[i].Name())
		if !isDirForRdmaDevice(rdmaDeviceName, dirName) {
			continue
		}
		devices = append(devices, filepath.Join(RdmaDeviceDir, fileInfos[i].Name()))
	}
	return devices
}

func getRdmaUcmDevices() []string {
	info, err := os.Stat(RdmaUcmDevice)
	if err != nil {
		return nil
	}
	if info.Name() == "rdma_cm" {
		return []string{RdmaUcmDevice}
	}
	return nil
}

// GetRdmaCharDevices returns a list of character device absolute paths for a
// requested rdmaDeviceName. For multi-port RDMA devices, all matching
// character devices of each type are returned (e.g., all umad and issm
// devices across all ports).
// Returns nil if no character devices are found.
func GetRdmaCharDevices(rdmaDeviceName string) []string {
	var rdmaCharDevices []string

	rdmaCharDevices = append(rdmaCharDevices,
		getCharDevices(rdmaDeviceName, RdmaIbUcmDir, RdmaUcmFilePrefix)...)
	rdmaCharDevices = append(rdmaCharDevices,
		getCharDevices(rdmaDeviceName, RdmaUmadDir, RdmaIssmFilePrefix)...)
	rdmaCharDevices = append(rdmaCharDevices,
		getCharDevices(rdmaDeviceName, RdmaUmadDir, RdmaUmadFilxPrefix)...)
	rdmaCharDevices = append(rdmaCharDevices,
		getCharDevices(rdmaDeviceName, RdmaUverbsDir, RdmaUverbsFilxPrefix)...)

	rdmaCharDevices = append(rdmaCharDevices,
		getRdmaUcmDevices()...)

	return rdmaCharDevices
}

// Gets a list of ports for a specified device
//
//nolint:prealloc
func GetPorts(rdmaDeviceName string) []string {
	var ports []string

	portsDir := filepath.Join(RdmaClassDir, rdmaDeviceName, RdmaPortsdir)
	fd, err := os.Open(portsDir)
	if err != nil {
		return nil
	}
	defer fd.Close()

	fileInfos, err := fd.Readdir(-1)
	if err != nil {
		return nil
	}

	for i := range fileInfos {
		if fileInfos[i].Name() == "." || fileInfos[i].Name() == prevDir {
			continue
		}
		ports = append(ports, fileInfos[i].Name())
	}
	return ports
}

//nolint:prealloc
func getNetdeviceIDs(rdmaDeviceName, port string) []string {
	var indices []string

	dir := filepath.Join(RdmaClassDir, rdmaDeviceName, RdmaPortsdir, port,
		RdmaGidAttrDir, RdmaGidAttrNdevDir)

	fd, err := os.Open(dir)
	if err != nil {
		return nil
	}
	defer fd.Close()

	fileInfos, err := fd.Readdir(-1)
	if err != nil {
		return nil
	}

	for i := range fileInfos {
		if fileInfos[i].Name() == "." || fileInfos[i].Name() == prevDir {
			continue
		}
		indices = append(indices, fileInfos[i].Name())
	}
	return indices
}

func isNetdevForRdma(rdmaDeviceName, port, index, netdevName string) bool {
	fileName := filepath.Join(RdmaClassDir, rdmaDeviceName, RdmaPortsdir, port,
		RdmaGidAttrDir, RdmaGidAttrNdevDir, index)

	fd, err := os.OpenFile(fileName, os.O_RDONLY, ReadOnlyPermissions)
	if err != nil {
		return false
	}
	defer fd.Close()

	if _, err = fd.Seek(0, io.SeekStart); err != nil {
		return false
	}

	data, err := io.ReadAll(fd)
	if err != nil {
		return false
	}
	return (strings.TrimSuffix(string(data), "\n") == netdevName)
}

func getRdmaDeviceForEth(netdevName string) (string, error) {
	// Iterate over the list of rdma devices,
	// read the gid table attribute netdev
	// if the netdev matches, found the matching rdma device

	devices := GetRdmaDeviceList()
	for _, dev := range devices {
		ports := GetPorts(dev)
		for _, port := range ports {
			indices := getNetdeviceIDs(dev, port)
			for _, index := range indices {
				found := isNetdevForRdma(dev, port, index, netdevName)
				if found {
					return dev, nil
				}
			}
		}
	}
	return "", fmt.Errorf("rdma device not found for netdev %v", netdevName)
}

func getNodeGUID(rdmaDeviceName string) ([]byte, error) {
	var nodeGUID []byte

	fileName := filepath.Join(RdmaClassDir, rdmaDeviceName, RdmaNodeGuidFile)

	fd, err := os.OpenFile(fileName, os.O_RDONLY, ReadOnlyPermissions)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	if _, err = fd.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(fd)
	if err != nil {
		return nil, err
	}
	data = data[:len(data)-1]
	var j int
	for _, b := range data {
		if b == ':' {
			continue
		}
		c, err := strconv.ParseUint(string(b), 16, 8)
		if err != nil {
			return nil, err
		}
		if (j % 2) == 0 {
			nodeGUID = append(nodeGUID, byte(c)<<nibbleBitSize)
		} else {
			nodeGUID[j/2] |= byte(c)
		}
		j++
	}
	return nodeGUID, nil
}

func getPortGUID(rdmaDeviceName, port string) ([]byte, error) {
	// Read the GID from port's gids/0
	// The GID format is: fe80:0000:0000:0000:PPPP:PPPP:PPPP:PPPP
	// where the last 8 bytes (64 bits) are the port GUID
	fileName := filepath.Join(RdmaClassDir, rdmaDeviceName, RdmaPortsdir, port, RdmaGidsDir, "0")

	fd, err := os.OpenFile(fileName, os.O_RDONLY, ReadOnlyPermissions)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	if _, err = fd.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	data, err := io.ReadAll(fd)
	if err != nil {
		return nil, err
	}

	// Parse GID format: "fe80:0000:0000:0000:0015:5dff:fd34:025b\n"
	gidStr := strings.TrimSpace(string(data))
	parts := strings.Split(gidStr, ":")
	if len(parts) != gidParts {
		return nil, fmt.Errorf("invalid GID format: %s", gidStr)
	}

	// Extract last 4 groups (8 bytes) which represent the port GUID
	var portGUID []byte
	for i := portGUIDStartIndex; i < gidParts; i++ {
		// Each part is 4 hex digits (2 bytes)
		if len(parts[i]) != gidPartLength {
			return nil, fmt.Errorf("invalid GID part: %s", parts[i])
		}
		// Parse first byte
		b1, err := strconv.ParseUint(parts[i][0:hexBytePairLength], 16, 8)
		if err != nil {
			return nil, err
		}
		// Parse second byte
		b2, err := strconv.ParseUint(parts[i][hexBytePairLength:gidPartLength], 16, 8)
		if err != nil {
			return nil, err
		}
		portGUID = append(portGUID, byte(b1), byte(b2))
	}

	return portGUID, nil
}

func getRdmaDeviceForIb(linkAttr *netlink.LinkAttrs) (string, error) {
	// Match the port GUID EUI bytes with the IpoIB netdevice hw address EUI
	// IPoIB hardware addresses encode the port GUID, not the node GUID
	lleui64 := linkAttr.HardwareAddr[12:]

	devices := GetRdmaDeviceList()
	for _, dev := range devices {
		ports := GetPorts(dev)
		for _, port := range ports {
			portGUID, err := getPortGUID(dev, port)
			if err != nil {
				// If we can't read the port GUID, skip this port
				continue
			}
			if bytes.Equal(lleui64, portGUID) {
				return dev, nil
			}
		}
	}
	return "", nil
}

// Get RDMA device for the netdevice
func GetRdmaDeviceForNetdevice(netdevName string) (string, error) {
	handle, err := netlink.LinkByName(netdevName)
	if err != nil {
		return "", err
	}
	netAttr := handle.Attrs()
	switch netAttr.EncapType {
	case "ether":
		return getRdmaDeviceForEth(netdevName)
	case "infiniband":
		return getRdmaDeviceForIb(netAttr)
	default:
		return "", fmt.Errorf("unknown device type")
	}
}

// Returns true if rdma device exist for netdevice, else false
func IsRDmaDeviceForNetdevice(netdevName string) bool {
	rdma, _ := GetRdmaDeviceForNetdevice(netdevName)

	return (rdma != "")
}

//nolint:prealloc
func getRdmaDevicesFromDir(dirName string) []string {
	var rdmadevs []string

	entries, err := os.ReadDir(dirName)
	if err != nil {
		return rdmadevs
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		rdmadevs = append(rdmadevs, entry.Name())
	}
	return rdmadevs
}

// Get list of RDMA devices for a pci device.
// When switchdev mode is used, there may be more than one rdma device.
// Example pcidevName: 0000:05:00.0,
// when found, returns list of devices one or more devices names such as
// mlx5_0, mlx5_10
func GetRdmaDevicesForPcidev(pcidevName string) []string {
	dirName := filepath.Join(PciDevDir, pcidevName, RdmaClassName)
	return getRdmaDevicesFromDir(dirName)
}

// Get list of RDMA devices for an auxiliary device.
// When switchdev mode is used, there may be more than one rdma device.
// Example deviceID: mlx5_core.sf.4,
// when found, returns list of devices one or more devices names such as
// mlx5_0, mlx5_10
func GetRdmaDevicesForAuxdev(deviceID string) []string {
	dirName := filepath.Join(AuxDevDir, deviceID, RdmaClassName)
	return getRdmaDevicesFromDir(dirName)
}
