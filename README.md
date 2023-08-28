# 简介

暴露浏览器中的接口到 xhe wireguard vpn 网络中

# 如何使用

参考 [link_test.js](./link_test.js) 和 [testdata/index.html](./testdata/index.html)

# 在 UserScript 中使用

```js
// ==UserScript==
// @name        Xhe Link - B站测试
// @namespace   Violentmonkey Scripts
// @match       https://www.bilibili.com/account/history
// @version     1.0
// @author      -
// @require     https://unpkg.com/@remoon.net/xhe-link@0.0.4/dist/xhe-link.umd.js
// @description 02/06/2023, 02:52:30
// @grant none
// ==/UserScript==

XheLinkLib.Init()
  .then(async () => {
    const xwg = await XheLink({
      log_level: "debug",
      private_key: "SA7wvbecJtRXtb9ATH9h7Vu+GLq4qoOVPg/SrxIGP0w=",
      links: ["https://xhe.remoon.net"],
      peers: [
        "peer://8066d0db32b6dda61541d4513a431504599cb296b250f0b6855c7c30bcaab862",
      ],
    });

    const server = await xwg.ListenTCP(80);
    server.Serve().catch((err) => {
      console.err(err);
    });

    await server.ReverseProxy("/user/", "https://api.bilibili.com");

    console.log("反向代理成功");
  })
  .catch((err) => console.error(err));
```

```sh
xhe --vtun --export 10808 --log debug -k oKL7+pbuh/kJvD1pleelYM5r/F5i/G5iCZ7fNqPT8lU= -p 'https://xhe.remoon.net?peer=81dea2c5c077bf78b34a518eda9851cfbe718656fdc470970bde057cbceef23e&keepalive=15'
# another shell
curl -x socks5://127.0.0.1:10808 -i http://[fdd9:f800:e85c:5789:b094:ac0b:adb0:13d6]/user/x/web-interface/nav
```
 