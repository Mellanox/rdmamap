package rdmamap

import (
	"fmt"
	"testing"
)

func TestGetRdmaDevices(t *testing.T) {
	rdmaDevices := GetRdmaDeviceList()
	fmt.Println("Devices: ", rdmaDevices)
}

func TestRdmaCharDevices(t *testing.T) {
	rdmaDevices := GetRdmaDeviceList()
	fmt.Println("Devices: ", rdmaDevices)

	for _, dev := range rdmaDevices {
		charDevices := GetRdmaCharDevices(dev)
		fmt.Printf("Rdma device: = %s", dev)
		fmt.Println(" Char devices: = ", charDevices)
	}
}

func TestRdmaDeviceForNetdevice(t *testing.T) {

	netdev := "ib0"
	rdmaDev, err := GetRdmaDeviceForNetdevice(netdev)
	if err == nil {
		fmt.Printf("netdev = %s, rdmadev = %s\n", netdev, rdmaDev)
	} else {
		fmt.Println("rdma device not found for netdev = %s\n", netdev)
	}
	t.Fatal(nil)
}
