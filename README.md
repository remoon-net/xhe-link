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
// @require     https://unpkg.com/@remoon.cn/xhe-link@0.0.6/dist/xhe-link.umd.js
// @description 02/06/2023, 02:52:30
// @grant none
// ==/UserScript==

XheLinkLib.Init()
  .then(async () => {
    const xwg = await await XheLink({
      private_key: "sM3la8r44RHb6IDVA4BQeJUrmPEzARLH3ixpJev80GQ=",
      links: [
        "https://lighthouse.remoon.cn/?subnet=fdd9:f800:0:1:0:2:0:201/120&node={node}&key={key}",
      ],
      // log_level: "debug", // for debug
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
xhe --vtun --export 10808 -c 'https://lighthouse.remoon.cn/?ip=fdd9:f800:0:1:0:2:0:202/120&node={node}&sharedkey={key}' --log debug
# another shell
curl -x socks5://127.0.0.1:10808 -i http://[fdd9:f800:0:1:0:2:0:201]/user/x/web-interface/nav
```
