package rdmamap

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type RdmaStatEntry struct {
	Name  string
	Value uint64
}

type RdmaPortStats struct {
	HwStats []RdmaStatEntry /* /sys/class/infiniband/<dev>/<port>/hw_counters */
	Stats   []RdmaStatEntry /* /sys/class/infiniband/<dev>/<port>/counters */
	Port    int
}

type RdmaStats struct {
	PortStats []RdmaPortStats
}

func readCounter(name string) uint64 {

	fd, err := os.OpenFile(name, os.O_RDONLY, 0444)
	if err != nil {
		return 0
	}
	defer fd.Close()

	fd.Seek(0, os.SEEK_SET)
	data, err := ioutil.ReadAll(fd)
	if err != nil {
		return 0
	}
	dataStr := string(data)
	dataStr = strings.Trim(dataStr, "\n")
	value, _ := strconv.ParseUint(dataStr, 10, 64)
	return value
}

func getCountersFromDir(path string) ([]RdmaStatEntry, error) {

	var stats []RdmaStatEntry

	fd, err := os.Open(path)
	if err != nil {
		return stats, err
	}
	fileInfos, err := fd.Readdir(-1)
	defer fd.Close()

	for _, file := range fileInfos {
		if file.IsDir() {
			continue
		}
		value := readCounter(filepath.Join(path, file.Name()))
		entry := RdmaStatEntry{file.Name(), value}
		stats = append(stats, entry)
	}
	return stats, nil
}

// Get RDMA Sysfs stats from counters directory of a port of a rdma device
// Port number starts from 1.
func GetRdmaSysfsStats(rdmaDevice string, port int) ([]RdmaStatEntry, error) {

	path := filepath.Join(RdmaClassDir, rdmaDevice,
		RdmaPortsdir, strconv.Itoa(port), RdmaCountersDir)

	rdmastats, err := getCountersFromDir(path)
	return rdmastats, err
}

// Get RDMA Sysfs stats from hw_counters directory of a port of a rdma device
// Port number starts from 1.
func GetRdmaSysfsHwStats(rdmaDevice string, port int) ([]RdmaStatEntry, error) {

	path := filepath.Join(RdmaClassDir, rdmaDevice,
		RdmaPortsdir, strconv.Itoa(port), RdmaHwCountersDir)

	rdmastats, err := getCountersFromDir(path)
	return rdmastats, err
}

// Get RDMA sysfs starts from counter and hw_counters directory for a requested
// port of a device.
func GetRdmaSysfsAllStats(rdmaDevice string, port int) (RdmaPortStats, error) {
	var portstats RdmaPortStats

	hwstats, err := GetRdmaSysfsHwStats(rdmaDevice, port)
	if err != nil {
		return portstats, nil
	}
	portstats.HwStats = hwstats

	stats, err := GetRdmaSysfsStats(rdmaDevice, port)
	if err != nil {
		return portstats, nil
	}
	portstats.Stats = stats
	portstats.Port = port
	return portstats, nil
}

// Get RDMA sysfs starts from counter and hw_counters directory for a
// rdma device.
func GetRdmaSysfsAllPortsStats(rdmaDevice string) (RdmaStats, error) {
	var allstats RdmaStats

	path := filepath.Join(RdmaClassDir, rdmaDevice, RdmaPortsdir)
	fd, err := os.Open(path)
	if err != nil {
		return allstats, err
	}
	fileInfos, err := fd.Readdir(-1)
	defer fd.Close()

	for i, file := range fileInfos {
		if fileInfos[i].Name() == "." || fileInfos[i].Name() == ".." {
			continue
		}
		if !file.IsDir() {
			continue
		}
		port, _ := strconv.Atoi(file.Name())
		portstats, err := GetRdmaSysfsAllStats(rdmaDevice, port)
		if err != nil {
			return allstats, err
		}
		allstats.PortStats = append(allstats.PortStats, portstats)
	}
	return allstats, nil
}
