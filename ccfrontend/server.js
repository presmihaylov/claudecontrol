const { createServer } = require("node:https");
const { parse } = require("node:url");
const next = require("next");
const fs = require("node:fs");
const path = require("node:path");

const dev = process.env.NODE_ENV !== "production";
const app = next({ dev });
const handle = app.getRequestHandler();

const httpsOptions = {
	key: fs.readFileSync(path.join(__dirname, "certificates/localhost-key.pem")),
	cert: fs.readFileSync(path.join(__dirname, "certificates/localhost.pem")),
};

app.prepare().then(() => {
	createServer(httpsOptions, (req, res) => {
		const parsedUrl = parse(req.url, true);
		handle(req, res, parsedUrl);
	}).listen(3000, (err) => {
		if (err) throw err;
		console.log("> Ready on https://localhost:3000");
	});
});
