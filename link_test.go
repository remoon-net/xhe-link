//go:build ierr

package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/lainio/err2/assert"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"remoon.net/xhe/pkg/vtun"
	"remoon.net/xhe/pkg/xhe"
)

type keyAttr struct {
	ip     string
	key    []byte
	pubkey []byte
}

var key1 keyAttr
var key2 keyAttr

var serverCfg xhe.Config
var clientCfg xhe.Config

var signalerAddr string = "127.0.0.1:61111"

func TestMain(m *testing.M) {
	var ierr error
	defer then(&ierr, nil, func() {
		panic(ierr)
	})

	key1.key, ierr = base64.StdEncoding.DecodeString("SA7wvbecJtRXtb9ATH9h7Vu+GLq4qoOVPg/SrxIGP0w=")
	key2.key, ierr = base64.StdEncoding.DecodeString("oKL7+pbuh/kJvD1pleelYM5r/F5i/G5iCZ7fNqPT8lU=")

	ierr = exec.Command("npm", "run", "build").Run()
	ierr = func() (ierr error) {
		pubkey := wgtypes.Key(key1.key).PublicKey()
		key1.pubkey = pubkey[:]
		ip, ierr := xhe.GetIP(key1.pubkey)
		key1.ip = ip.Addr().String()
		return
	}()
	ierr = func() (ierr error) {
		pubkey := wgtypes.Key(key2.key).PublicKey()
		key2.pubkey = pubkey[:]
		ip, ierr := xhe.GetIP(key2.pubkey)
		key2.ip = ip.Addr().String()
		return
	}()

	serverCfg = xhe.Config{
		LogLevel:   slog.LevelDebug,
		PrivateKey: hex.EncodeToString(key1.key),
		Links:      []string{"https://xhe.remoon.net"},
		Peers: []string{
			"peer://" + hex.EncodeToString(key2.pubkey),
		},
	}
	clientCfg = xhe.Config{
		LogLevel:   slog.LevelDebug,
		PrivateKey: hex.EncodeToString(key2.key),
		Peers: []string{
			fmt.Sprintf("https://xhe.remoon.net?peer=%s&keepalive=15", hex.EncodeToString(key1.pubkey)),
		},
	}

	caddy := exec.Command("caddy", "file-server", "--listen", signalerAddr)
	ierr = caddy.Start()
	defer caddy.Process.Kill()

	m.Run()
}

func TestReqAtServer(t *testing.T) {
	var ierr error
	defer then(&ierr, nil, func() {
		t.Error(ierr)
	})
	cfg := serverCfg
	tun, ierr := vtun.CreateTUN("xhe", 2400-80)
	cfg.GoTun = tun
	dev, ierr := xhe.Run(cfg)
	defer dev.Close()

	cmd := exec.Command("node", "link_test.js")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ierr = cmd.Start()
	defer cmd.Process.Kill()

	time.Sleep(5 * time.Second)

	client := newClient(tun)
	resp, ierr := client.Get(fmt.Sprintf("http://[%s]/", key2.ip))
	body, ierr := io.ReadAll(resp.Body)
	t.Log(string(body))
	assert.Equal(string(body), "hello world")
}

func TestReqAtClient(t *testing.T) {
	var ierr error
	defer then(&ierr, nil, func() {
		t.Error(ierr)
	})
	cmd := exec.Command("node", "link_test.js", "server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	ierr = cmd.Start()
	defer cmd.Process.Kill()

	time.Sleep(2 * time.Second)

	cfg := clientCfg
	tun, ierr := vtun.CreateTUN("xhe", 2400-80)
	cfg.GoTun = tun
	dev, ierr := xhe.Run(cfg)
	defer dev.Close()

	stats, ierr := dev.IpcGet()
	fmt.Println(stats)

	time.Sleep(3 * time.Second)

	client := newClient(tun)
	resp, ierr := client.Get(fmt.Sprintf("http://[%s]/", key1.ip))
	body, ierr := io.ReadAll(resp.Body)
	t.Log(string(body))
	assert.Equal(string(body), "hello world")
}

func TestReqBrowserAtServer(t *testing.T) {
	var ierr error
	defer then(&ierr, nil, func() {
		t.Error(ierr)
	})
	cfg := serverCfg
	tun, ierr := vtun.CreateTUN("xhe", 2400-80)
	cfg.GoTun = tun
	dev, ierr := xhe.Run(cfg)
	defer dev.Close()

	{
		opts := chromedp.DefaultExecAllocatorOptions[:]
		// opts = append(opts, chromedp.Flag("headless", false))
		ctx := context.Background()
		ctx, cancel := chromedp.NewExecAllocator(ctx, opts...)
		defer cancel()
		ctx, cancel = chromedp.NewContext(ctx)
		defer cancel()
		tasks := []chromedp.Action{
			chromedp.Navigate(fmt.Sprintf("http://%s/testdata/", signalerAddr)),
		}
		ierr = chromedp.Run(ctx, tasks...)
	}

	time.Sleep(5 * time.Second)

	client := newClient(tun)
	resp, ierr := client.Get(fmt.Sprintf("http://[%s]/hello.txt", key2.ip))
	body, ierr := io.ReadAll(resp.Body)
	t.Log(body)
	assert.Equal(string(body), "hello world")
}

func TestReqBrowserAtClient(t *testing.T) {
	var ierr error
	defer then(&ierr, nil, func() {
		t.Error(ierr)
	})

	{
		opts := chromedp.DefaultExecAllocatorOptions[:]
		// opts = append(opts, chromedp.Flag("headless", false))
		ctx := context.Background()
		ctx, cancel := chromedp.NewExecAllocator(ctx, opts...)
		defer cancel()
		ctx, cancel = chromedp.NewContext(ctx)
		defer cancel()
		tasks := []chromedp.Action{
			chromedp.Navigate(fmt.Sprintf("http://%s/testdata/?server=1", signalerAddr)),
		}
		ierr = chromedp.Run(ctx, tasks...)
	}

	time.Sleep(3 * time.Second)

	cfg := clientCfg
	tun, ierr := vtun.CreateTUN("xhe", 2400-80)
	cfg.GoTun = tun
	dev, ierr := xhe.Run(cfg)
	defer dev.Close()

	time.Sleep(3 * time.Second)

	client := newClient(tun)
	resp, ierr := client.Get(fmt.Sprintf("http://[%s]/hello.txt", key1.ip))
	body, ierr := io.ReadAll(resp.Body)
	t.Log(string(body))
	assert.Equal(string(body), "hello world")
}

func newClient(tun vtun.GetStack) *http.Client {
	stack := tun.GetStack()
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, ierr error) {
				d, ierr := netip.ParseAddrPort(addr)
				fa, pn := convertToFullAddr(tun.NIC(), d)
				return gonet.DialContextTCP(ctx, stack, fa, pn)
			},
		},
	}
	return client
}
