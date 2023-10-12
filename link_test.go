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
	"github.com/shynome/err0/try"
	"github.com/stretchr/testify/assert"
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

	key1.key = try.To1(base64.StdEncoding.DecodeString("SA7wvbecJtRXtb9ATH9h7Vu+GLq4qoOVPg/SrxIGP0w="))
	key2.key = try.To1(base64.StdEncoding.DecodeString("oKL7+pbuh/kJvD1pleelYM5r/F5i/G5iCZ7fNqPT8lU="))

	try.To(exec.Command("npm", "run", "build").Run())

	func() {
		pubkey := wgtypes.Key(key1.key).PublicKey()
		key1.pubkey = pubkey[:]
		ip := try.To1(xhe.GetIP(key1.pubkey))
		key1.ip = ip.Addr().String()
		return
	}()

	func() {
		pubkey := wgtypes.Key(key2.key).PublicKey()
		key2.pubkey = pubkey[:]
		ip := try.To1(xhe.GetIP(key2.pubkey))
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
	try.To(caddy.Start())
	defer caddy.Process.Kill()

	m.Run()
}

func TestReqAtServer(t *testing.T) {
	cfg := serverCfg
	tun := try.To1(vtun.CreateTUN("xhe", 2400-80))
	cfg.GoTun = tun
	dev := try.To1(xhe.Run(cfg))
	defer dev.Close()

	cmd := exec.Command("node", "link_test.js")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	try.To(cmd.Start())
	defer cmd.Process.Kill()

	time.Sleep(5 * time.Second)

	client := newClient(tun)
	{
		resp := try.To1(client.Get(fmt.Sprintf("http://[%s]/", key2.ip)))
		body := try.To1(io.ReadAll(resp.Body))
		t.Log(string(body))
		assert.Equal(t, "hello world", string(body))
	}
	{
		resp := try.To1(client.Get(fmt.Sprintf("http://[%s]:7070/", key2.ip)))
		body := try.To1(io.ReadAll(resp.Body))
		t.Log(string(body))
		assert.Equal(t, "hello hono", string(body))
	}
}

func TestReqAtClient(t *testing.T) {
	cmd := exec.Command("node", "link_test.js", "server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	try.To(cmd.Start())
	defer cmd.Process.Kill()

	time.Sleep(2 * time.Second)

	cfg := clientCfg
	tun := try.To1(vtun.CreateTUN("xhe", 2400-80))
	cfg.GoTun = tun
	dev := try.To1(xhe.Run(cfg))
	defer dev.Close()

	stats := try.To1(dev.IpcGet())
	fmt.Println(stats)

	time.Sleep(3 * time.Second)

	client := newClient(tun)
	resp := try.To1(client.Get(fmt.Sprintf("http://[%s]/", key1.ip)))
	body := try.To1(io.ReadAll(resp.Body))
	t.Log(string(body))
	assert.Equal(t, "hello world", string(body))
}

func TestReqBrowserAtServer(t *testing.T) {
	cfg := serverCfg
	tun := try.To1(vtun.CreateTUN("xhe", 2400-80))
	cfg.GoTun = tun
	dev := try.To1(xhe.Run(cfg))
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
		try.To(chromedp.Run(ctx, tasks...))
	}

	time.Sleep(5 * time.Second)

	client := newClient(tun)
	resp := try.To1(client.Get(fmt.Sprintf("http://[%s]/hello.txt", key2.ip)))
	body := try.To1(io.ReadAll(resp.Body))
	t.Log(body)
	assert.Equal(t, "hello world", string(body))
}

func TestReqBrowserAtClient(t *testing.T) {
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
		try.To(chromedp.Run(ctx, tasks...))
	}

	time.Sleep(3 * time.Second)

	cfg := clientCfg
	tun := try.To1(vtun.CreateTUN("xhe", 2400-80))
	cfg.GoTun = tun
	dev := try.To1(xhe.Run(cfg))
	defer dev.Close()

	time.Sleep(3 * time.Second)

	client := newClient(tun)
	resp := try.To1(client.Get(fmt.Sprintf("http://[%s]/hello.txt", key1.ip)))
	body := try.To1(io.ReadAll(resp.Body))
	t.Log(string(body))
	assert.Equal(t, "hello world", string(body))
}

func newClient(tun vtun.GetStack) *http.Client {
	stack := tun.GetStack()
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (conn net.Conn, err error) {
				d, err := netip.ParseAddrPort(addr)
				if err != nil {
					return
				}
				fa, pn := convertToFullAddr(tun.NIC(), d)
				return gonet.DialContextTCP(ctx, stack, fa, pn)
			},
		},
	}
	return client
}
