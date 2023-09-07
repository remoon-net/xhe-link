# Introduction

ReverseProxy browser fetch api to xhe wireguard vpn network

# How to use

you can see [link_test.js](./link_test.js) and [testdata/index.html](./testdata/index.html)

# the real example by use UserScript

before connect browser and linux, we need generate key1, pubkey1(hex) and key2, pubkey2(hex)

```sh
echo -n 'key1: ' > /tmp/key1.txt; wg genkey >> /tmp/key1.txt
echo -n 'key2: ' > /tmp/key2.txt; wg genkey >> /tmp/key2.txt
echo -n 'pubkey1: ' > /tmp/pubkey1.txt; tail -c +7 /tmp/key1.txt | wg pubkey | base64 -d | xxd -p -c 32 >> /tmp/pubkey1.txt
echo -n 'pubkey2: ' > /tmp/pubkey2.txt; tail -c +7 /tmp/key2.txt | wg pubkey | base64 -d | xxd -p -c 32 >> /tmp/pubkey2.txt
cat /tmp/key1.txt /tmp/pubkey1.txt /tmp/key2.txt /tmp/pubkey2.txt | tee xhe-link-keys.txt
# open xhe-link-keys.txt, the below steps will use those keys
```

twitter has enable csp, it will reject any script soruce is not from twitter.
so we need disable csp to allow UserScript execute, by use extenstion like [CSP Unblock](https://chrome.google.com/webstore/detail/csp-unblock/lkbelpgpclajeekijigjffllhigbhobd)

```js
// ==UserScript==
// @name        twitter spider - Xhe Link
// @namespace   Violentmonkey Scripts
// @match       *://twitter.com/*
// @version     1.0
// @author      -
// @require     https://unpkg.com/@remoon.net/xhe-link@0.0.6/dist/xhe-link.umd.js
// @run-at      document-start
// @description 02/06/2023, 02:52:30
// @grant none
// ==/UserScript==

// twitter service worker also enable csp, so disable sw is required
navigator.serviceWorker.register = (...args) => {
  console.log("sw: try regsiter", args);
  return new Promise((rl, rj) => {
    rj("disable sw");
  });
};

XheLinkLib.Init()
  .then(async () => {
    const xwg = await XheLink({
      log_level: "debug",
      private_key: "{key1}",
      links: ["https://xhe.remoon.net"],
      peers: ["peer://{pubkey2}"],
    });

    const server = await xwg.ListenTCP(80);
    server.Serve().catch((err) => {
      console.err(err);
    });

    let fetch = globalThis.fetch;
    globalThis.fetch = async function hookedFetch(u, init = {}) {
      if (init.credentials === "include") {
        let h = new Headers(init?.headers);
        let csrf = await cookieStore.get("ct0").then((c) => c.value);
        h.set("x-csrf-token", csrf);
        init.headers = h;
      }
      return fetch(u, init).then((res) => {
        let u = new URL(res.url);
        let hostname = u.hostname;
        if (hostname.endsWith("twitter.com")) {
          let rh = new Headers(res.headers);
          // twitter api response content-length header is not equal the real content body length
          rh.delete("content-length");
          return new Response(res.body, {
            headers: rh,
            status: res.status,
            statusText: res.statusText,
          });
        }
        return res;
      });
    };

    await server.ReverseProxy("/", "https://twitter.com");

    console.log("reverse proxy successful");
  })
  .catch((err) => console.error(err));
```

get xhe from <https://github.com/remoon-net/xhe>

```sh
xhe --vtun --export 10808 \
  -k {key2} \
  -p 'https://xhe.remoon.net?peer={pubkey1}&keepalive=15' \
  --log debug

# another shell
curl -x socks5://127.0.0.1:10808 \
  -H 'authorization: Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA' \
  -H 'js.fetch.credentials: include' \
  'http://twitter.com/i/api/graphql/_pnlqeTOtnpbIL9o-fS_pg/ProfileSpotlightsQuery?variables=%7B%22screen_name%22%3A%22shynome%22%7D' \
  # ip fdd9:f800:1325:6416:ba5a:cfba:f495:2536 is from `xhe ip pubkey1`
  --resolve twitter.com:80:fdd9:f800:1325:6416:ba5a:cfba:f495:2536
```
