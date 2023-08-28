//go:build ierr

package main

import (
	"fmt"
	"log/slog"
	"os"
	"syscall/js"

	promise "github.com/nlepage/go-js-promise"
	"golang.zx2c4.com/wireguard/device"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"remoon.net/xhe/pkg/vtun"
	"remoon.net/xhe/pkg/xhe"
)

func main() {
	js.Global().Set("XheLink", js.FuncOf(connect))
	<-make(chan any)
}

func connect(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() (ierr error) {
		defer then(&ierr, nil, func() {
			reject(ierr.Error())
		})

		if len(args) == 0 {
			return fmt.Errorf("config is required")
		}

		config, ierr := getConfig[xhe.Config](args[0])
		h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: config.LogLevel})
		slog.SetDefault(slog.New(h))

		tun, ierr := vtun.CreateTUN("xhe", 2400-80)
		config.GoTun = tun
		dev, ierr := xhe.Run(config)
		stack := tun.GetStack()

		xwg := NewXheWireguard(dev, stack, tun.NIC())
		resolve(xwg.ToJS())
		return
	}()
	return
}

type XheWireguard struct {
	dev   *device.Device
	stack *stack.Stack
	nic   tcpip.NICID
}

func NewXheWireguard(dev *device.Device, net *stack.Stack, nic tcpip.NICID) *XheWireguard {
	return &XheWireguard{
		dev:   dev,
		stack: net,
		nic:   nic,
	}
}

func (xwg *XheWireguard) ToJS() (root js.Value) {
	root = js.Global().Get("Object").New()
	root.Set("ListenTCP", js.FuncOf(xwg.ListenTCP))
	root.Set("IpcGet", js.FuncOf(xwg.IpcGet))
	root.Set("IpcSet", js.FuncOf(xwg.IpcSet))
	return root
}

func (n *XheWireguard) IpcGet(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() (ierr error) {
		defer then(&ierr, nil, func() {
			reject(ierr.Error())
		})
		config, ierr := n.dev.IpcGet()
		resolve(config)
		return
	}()
	return
}

func (n *XheWireguard) IpcSet(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() (ierr error) {
		defer then(&ierr, nil, func() {
			reject(ierr.Error())
		})
		if len(args) == 0 {
			return fmt.Errorf("wireguard config required")
		}
		conf := args[0].String()
		ierr = n.dev.IpcSet(conf)
		resolve("")
		return
	}()
	return
}
