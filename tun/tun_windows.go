// +build windows

package tun

import (
	// "encoding/binary"
	"fmt"
	"net"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

const (
	TAPWIN32_MAX_REG_SIZE = 256
	TUNTAP_COMPONENT_ID   = "tap0901"
	ADAPTER_KEY           = `SYSTEM\CurrentControlSet\Control\Class\{4D36E972-E325-11CE-BFC1-08002BE10318}`
)

var (
	TAP_IOCTL_GET_MTU          = tapControlCode(3, 0)
	TAP_IOCTL_SET_MEDIA_STATUS = tapControlCode(6, 0)
	TAP_IOCTL_CONFIG_TUN       = tapControlCode(10, 0)
)

type tun struct {
	mtu        int
	devicePath string
	fd         syscall.Handle
}

func openTunTap(addr net.IP, network net.IP, mask net.IP) (*tun, error) {
	t := new(tun)
	id, err := getTuntapComponentID()
	if err != nil {
		return nil, err
	}
	t.devicePath = fmt.Sprintf(`\\.\Global\%s.tap`, id)
	name := syscall.StringToUTF16(t.devicePath)
	tuntap, err := syscall.CreateFile(
		&name[0],
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_ATTRIBUTE_SYSTEM|syscall.FILE_FLAG_OVERLAPPED,
		0)
	if err != nil {
		fmt.Println("here")
		return nil, err
	}
	var returnLen uint32
	configTunParam := append(addr.To4(), network.To4()...)
	configTunParam = append(configTunParam, mask.To4()...)
	fmt.Println(configTunParam)
	//configTunParam = []byte{10, 0, 0, 1, 10, 0, 0, 0, 255, 255, 255, 0}
	if err = syscall.DeviceIoControl(
		tuntap,
		TAP_IOCTL_CONFIG_TUN,
		&configTunParam[0],
		uint32(len(configTunParam)),
		&configTunParam[0],
		uint32(len(configTunParam)),
		&returnLen,
		nil); err != nil {
		fmt.Println("here2")
		return nil, err
	}

	// get MTU
	// var umtu = make([]byte, 4)
	// if err = syscall.DeviceIoControl(
	// 	tuntap,
	// 	TAP_IOCTL_GET_MTU,
	// 	nil,
	// 	0,
	// 	&umtu[0],
	// 	uint32(len(umtu)),
	// 	&returnLen,
	// 	nil); err != nil {
	// 	fmt.Println("here3")
	// 	return nil, err
	// }
	// mtu := binary.LittleEndian.Uint32(umtu)
	mtu := 1500

	// set connect.
	inBuffer := []byte("\x01\x00\x00\x00")
	if err = syscall.DeviceIoControl(
		tuntap,
		TAP_IOCTL_SET_MEDIA_STATUS,
		&inBuffer[0],
		uint32(len(inBuffer)),
		&inBuffer[0],
		uint32(len(inBuffer)),
		&returnLen,
		nil); err != nil {
		return nil, err
	}
	t.fd = tuntap
	t.mtu = int(mtu)
	return t, nil
}

func getTuntapComponentID() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE,
		ADAPTER_KEY,
		registry.ENUMERATE_SUB_KEYS|registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()

	names, err := k.ReadSubKeyNames(-1)
	if err != nil {
		return "", err
	}

	for _, name := range names {
		n, _ := matchKey(k, name, TUNTAP_COMPONENT_ID)
		if n != "" {
			return n, nil
		}
	}
	return "", fmt.Errorf("Not Found")

}
func matchKey(zones registry.Key, kname string, componentID string) (string, error) {
	k, err := registry.OpenKey(zones, kname, registry.READ)
	if err != nil {
		return "", err
	}
	defer k.Close()

	cID, _, err := k.GetStringValue("ComponentId")
	if cID == componentID {
		netCfgInstanceID, _, err := k.GetStringValue("NetCfgInstanceId")
		if err != nil {
			return "", err
		}
		return netCfgInstanceID, nil

	}
	return "", fmt.Errorf("ComponentId != componentId")
}

func (t tun) Read(ch []byte) (n int, err error) {
	overlappedRx := syscall.Overlapped{}
	var hevent windows.Handle
	hevent, err = windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return
	}
	overlappedRx.HEvent = syscall.Handle(hevent)
	buf := make([]byte, t.mtu)
	var l uint32

	if err := syscall.ReadFile(t.fd, buf, &l, &overlappedRx); err != nil {
	}
	if _, err := syscall.WaitForSingleObject(overlappedRx.HEvent, syscall.INFINITE); err != nil {
		fmt.Println(err)
	}
	//overlappedRx.Offset += l
	totalLen := overlappedRx.InternalHigh
	/*switch buf[0] & 0xf0 {
	  case 0x40:
	      totalLen = 256 * int(buf[2]) + int(buf[3])
	  case 0x60:
	      continue
	      totalLen = 256 * int(buf[4]) + int(buf[5]) + IPv6_HEADER_LENGTH
	  }*/
	//fmt.Println("read data", buf[:totalLen])
	send := make([]byte, totalLen)
	copy(send, buf)
	copy(ch, buf)
	return int(totalLen), nil
}

func (t tun) Write(ch []byte) (n int, err error) {
	overlappedRx := syscall.Overlapped{}
	var hevent windows.Handle
	hevent, err = windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return
	}
	overlappedRx.HEvent = syscall.Handle(hevent)
	var l uint32
	syscall.WriteFile(t.fd, ch, &l, &overlappedRx)
	syscall.WaitForSingleObject(overlappedRx.HEvent, syscall.INFINITE)
	overlappedRx.Offset += uint32(len(ch))
	return len(ch), nil
}

func (t tun) Close() error {
	return syscall.Close(t.fd)
}

func unicodeTostring(src []byte) string {
	var dst []byte
	for _, ch := range src {
		if ch != byte(0) {
			dst = append(dst, ch)
		}
	}
	return string(dst)
}

func ctl_code(deviceType, function, method, access uint32) uint32 {
	return (deviceType << 16) | (access << 14) | (function << 2) | method
}

func tapControlCode(request, method uint32) uint32 {
	return ctl_code(34, request, method, 0)
}

func (tun *Tun) Open() {
	_, e := openTunTap(net.ParseIP("10.1.0.2"), net.ParseIP("10.1.0.1"), net.ParseIP("255.255.255.0"))
	if e == nil {
		// tun.Fd = (*t).(io.ReadWriteCloser)
	}
}
