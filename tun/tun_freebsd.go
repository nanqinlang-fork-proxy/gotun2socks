// +build freebsd

package tun

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
)

const (
	TUNSIFHEAD = 2147775584
	TUNGIFHEAD = 1074033761
)

func (tun *Tun) Open() {
	dynamicOpened := false
	dynamicName := ""
	for i := 0; i < 16; i++ {
		tunName := fmt.Sprintf("/dev/tun%d", i)
		dynamicName = fmt.Sprintf("tun%d", i)
		fd, err := os.OpenFile(tunName, os.O_RDWR, 0)
		if err == nil {
			ifr := make([]byte, 8)
			_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
				uintptr(fd.Fd()), uintptr(TUNSIFHEAD),
				uintptr(unsafe.Pointer(&ifr[0])))
			if errno != 0 {
				log.Fatalf("[CRIT] Cannot ioctl TUNSIFHEAD: %v", errno)
			}

			syscall.SetNonblock(int(fd.Fd()), false)
			tun.Fd = fd
			dynamicOpened = true
			break
		}
		log.Printf("[WARN] Failed to open TUN/TAP device '%s': %v", dynamicName, err)
	}
	if !dynamicOpened {
		log.Fatalf("[CRIT] Cannot allocate TUN/TAP device dynamically.")
	}

	tun.actualName = dynamicName
	log.Printf("[INFO] TUN/TAP device %s opened.", tun.actualName)
}

// this is currently broken D:

/*

#include <string.h>
#include <unistd.h>
#include <fcntl.h>
#include <netinet/in.h>
#include <netinet/ip.h>
#include <arpa/inet.h>
#include <sys/ioctl.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <net/if.h>
#include <net/if_tun.h>
#include <stdio.h>
#include <stdlib.h>

char * tundev_open(int * tunfd) {

  char * name = (char *) malloc(IFNAMSIZ);
  int tun = 0;
  *tunfd = -1;
  do {
    memset(name, 0, IFNAMSIZ);
    sprintf(name, "/dev/tun%d", tun);
    int fd = open(name, O_RDWR);
    if (fd > 0) {
      int i = 0;
      if ( ioctl(fd, TUNSIFHEAD, &i) < 0 ) {
        close(fd);
        perror("TUNSIFHEAD");
        break;
      }
      *tunfd = fd;
      break;
    }
    tun ++;
  } while(tun < 10);
  return name;
}

int tundev_up(char * ifname, char * addr, char * netmask, int mtu) {

  struct ifreq ifr;
  memset(&ifr, 0, sizeof(struct ifreq));
  strncpy(ifr.ifr_name, ifname, IFNAMSIZ);
  int fd = socket(AF_INET, SOCK_DGRAM, IPPROTO_IP);
  if ( fd > 0 ) {
    ifr.ifr_mtu = mtu;
    if ( ioctl(fd, SIOCSIFMTU, (void*) &ifr) < 0) {
      close(fd);
      perror("SIOCSIFMTU");
      return -1;
    }

    struct sockaddr_in src;
    memset(&src, 0, sizeof(struct sockaddr_in));
    src.sin_family = AF_INET;
    if ( ! inet_aton(addr, &src.sin_addr) ) {
      printf("invalid srcaddr %s\n", addr);
      close(fd);
      return -1;
    }

    memset(&ifr, 0, sizeof(struct ifreq));
    strncpy(ifr.ifr_name, ifname, IFNAMSIZ);
    memcpy(&ifr.ifr_addr, &src, sizeof(struct sockaddr_in));
    if ( ioctl(fd, SIOCSIFADDR, (void*)&ifr) < 0 ) {
      close(fd);
      perror("SIOCSIFADDR");
     return -1;
    }

    memset(&ifr, 0, sizeof(struct ifreq));
    strncpy(ifr.ifr_name, ifname, IFNAMSIZ);
    if ( ioctl(fd, SIOCGIFFLAGS, (void*)&ifr) < 0 ) {
      close(fd);
      perror("SIOCGIFFLAGS");
      return -1;
    }
    ifr.ifr_flags |= IFF_UP ;
    if ( ioctl(fd, SIOCSIFFLAGS, (void*)&ifr) < 0 ) {
      perror("SIOCSIFFLAGS");
      close(fd);
      return -1;
    }

    close(fd);
    return 0;
  }
  return -1;
}

void tundev_close(int fd) {
  close(fd);
}

void tundev_free(const char * name) {
  if (name) {
    free((void*)name);
  }
}

*/
