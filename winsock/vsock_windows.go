package winsock

// TODO: replace go-ntdll. No need for it.

import (
	"fmt"
	"syscall"
	"unicode/utf16"
	"unsafe"

	ntdll "github.com/hillu/go-ntdll"
	"golang.org/x/sys/windows"
)

const (
	// https://docs.microsoft.com/en-us/windows/win32/fileio/file-access-rights-constants
	FILE_READ_DATA        = 0x0001
	FILE_WRITE_DATA       = 0x0002
	FILE_READ_EA          = 0x0008
	FILE_WRITE_EA         = 0x0010
	FILE_EXECUTE          = 0x0020
	FILE_READ_ATTRIBUTES  = 0x0080
	FILE_WRITE_ATTRIBUTES = 0x0100
	FILE_ALL_ACCESS       = windows.STANDARD_RIGHTS_REQUIRED | windows.SYNCHRONIZE | 0x01FF

	// https://docs.microsoft.com/en-us/windows/win32/fileio/file-security-and-access-rights
	FILE_GENERIC_READ    = windows.STANDARD_RIGHTS_READ | FILE_READ_DATA | FILE_READ_ATTRIBUTES | FILE_READ_EA | windows.SYNCHRONIZE
	FILE_GENERIC_WRITE   = windows.STANDARD_RIGHTS_WRITE | FILE_WRITE_DATA | windows.FILE_WRITE_ATTRIBUTES | FILE_WRITE_EA | windows.FILE_APPEND_DATA | windows.SYNCHRONIZE
	FILE_GENERIC_EXECUTE = windows.STANDARD_RIGHTS_EXECUTE | FILE_READ_ATTRIBUTES | FILE_EXECUTE | windows.SYNCHRONIZE
)

var (
	// FILE_DEVICE_UNKNOWN defines a device type that does not have an explicitly defined type.
	// Information available at: https://docs.microsoft.com/en-us/windows-hardware/drivers/kernel/specifying-device-types
	FILE_DEVICE_UNKNOWN uint32 = 34

	GET_CONFIG_FUNCTION        = 0x800
	METHOD_BUFFERED     uint32 = 0

	FILE_ANY_ACCESS      uint32 = 0
	FILE_READ_ACCESS     uint32 = 1
	FILE_WRITE_ACCESS    uint32 = 2
	OBJ_CASE_INSENSITIVE uint32 = 64

	INVALID_SOCKET = ^ntdll.Handle(0)

	vSockDeviceName = "\\??\\Viosock"
)

func ctl_code(device_type, function, method, access uint32) uint32 {
	return (device_type << 16) | (access << 14) | (function << 2) | method
}

func ioctlGetConfig() uint32 {
	return ctl_code(FILE_DEVICE_UNKNOWN, 0x800, METHOD_BUFFERED, FILE_READ_ACCESS)
}

// NtStatus holds the status returned by NtCreateFile
type NtStatus uint32

type SocketConfig struct {
	GuestCID uint64
}

func GetVioSocketConfig(sock syscall.Handle) (SocketConfig, error) {
	var sockCfg SocketConfig
	var bytesReturned uint32

	err := syscall.DeviceIoControl(sock,
		ioctlGetConfig(),
		nil,
		0,
		(*byte)(unsafe.Pointer(&sockCfg)),
		uint32(unsafe.Sizeof(sockCfg)),
		&bytesReturned, nil)

	if err != nil {
		return SocketConfig{}, err
	}

	return sockCfg, nil
}

type ObjectAttributes struct {
	Length                   uint32
	RootDirectory            windows.Handle
	ObjectName               *UnicodeString
	Attributes               uint32
	SecurityDescriptor       *byte
	SecurityQualityOfService *byte
}

type IOStatusBlock struct {
	StatusPointer uintptr
	Information   uintptr
}

type UnicodeString struct {
	Length    uint16
	MaxLength uint16
	Buffer    *uint16
}

func NewUnicodeString(s string) *ntdll.UnicodeString {
	buf := utf16.Encode([]rune(s))
	return &ntdll.UnicodeString{
		Length:        uint16(2 * len(buf)),
		MaximumLength: uint16(2 * len(buf)),
		Buffer:        &buf[0],
	}
}

func InitializeObjectAttributes(name string, attributes uint32, rootDir ntdll.Handle, pSecurityDescriptor *byte) (oa ntdll.ObjectAttributes, e error) {
	oa = ntdll.ObjectAttributes{
		RootDirectory:      rootDir,
		Attributes:         attributes,
		SecurityDescriptor: pSecurityDescriptor,
	}
	oa.Length = uint32(unsafe.Sizeof(oa))

	if len(name) > 0 {
		us := NewUnicodeString(name)
		oa.ObjectName = us
	}

	return
}

func VIOSockCreateSocket(params uint64) (syscall.Handle, error) {
	var hSock ntdll.Handle = INVALID_SOCKET
	var status ntdll.NtStatus
	var rootDor ntdll.Handle
	oa, err := InitializeObjectAttributes(vSockDeviceName, OBJ_CASE_INSENSITIVE, rootDor, nil)
	if err != nil {
		return 0, err
	}
	var iosb ntdll.IoStatusBlock
	var allocSize *int64

	status = ntdll.NtCreateFile(&hSock, FILE_GENERIC_READ|FILE_GENERIC_WRITE, &oa, &iosb, allocSize, 0,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, ntdll.FILE_OPEN, ntdll.FILE_NON_DIRECTORY_FILE, nil, 0)

	fmt.Println(status)
	fmt.Println(hSock)

	return (syscall.Handle)(hSock), nil
}
