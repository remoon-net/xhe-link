// @ts-check
const { Init } = require(".");

const server = require("http")
  .createServer((req, res) => {
    res.end("hello world");
  })
  .listen(0);

// @ts-ignore
let port = server.address().port;

let p = Init().then(async () => {
  const isServer = process.argv[2] === "server";
  console.log("server?", isServer);
  let xhe = isServer
    ? await XheLink({
        log_level: "debug",
        private_key: "SA7wvbecJtRXtb9ATH9h7Vu+GLq4qoOVPg/SrxIGP0w=",
        links: ["https://xhe.remoon.net"],
        peers: [
          "peer://8066d0db32b6dda61541d4513a431504599cb296b250f0b6855c7c30bcaab862",
        ],
      })
    : await XheLink({
        log_level: "debug",
        private_key: "oKL7+pbuh/kJvD1pleelYM5r/F5i/G5iCZ7fNqPT8lU=",
        peers: [
          "https://xhe.remoon.net?peer=81dea2c5c077bf78b34a518eda9851cfbe718656fdc470970bde057cbceef23e&keepalive=15",
        ],
      });
  let server = await xhe.ListenTCP();
  server.Serve().catch(() => {
    // donothing
  });
  if (!server.ServeReady()) {
    throw new Error("server is not ready");
  }
  await server.ReverseProxy("/", `http://127.0.0.1:${port}/`);
  await server.HandleEval("/xhe-eval");
});
p.catch((err) => {
  console.error(err);
  process.exit(1);
});

// hanlde performance.markResourceTiming is not a function
process.on("uncaughtException", (err) => {
  // console.error(err);
});
