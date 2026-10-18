// Same-origin website/API server, bound only to this PC.
const http = require('node:http');
const fs = require('node:fs');
const path = require('node:path');
const root = path.join(__dirname, '..', 'build', 'web');
const mime = {'.html':'text/html; charset=utf-8','.js':'text/javascript; charset=utf-8','.css':'text/css','.png':'image/png','.svg':'image/svg+xml','.ico':'image/x-icon','.json':'application/json'};
http.createServer((req, res) => {
  const url = new URL(req.url, 'http://localhost');
  if (url.pathname.startsWith('/api/')) {
    const proxy = http.request({hostname:'127.0.0.1',port:8647,path:req.url.slice(4),method:req.method,headers:{...req.headers,host:'127.0.0.1:8647'}}, upstream => {res.writeHead(upstream.statusCode,upstream.headers); upstream.pipe(res);});
    proxy.on('error',()=>{res.writeHead(502);res.end('The local API is not running. Start Pokedextracker again.');});
    req.pipe(proxy); return;
  }
  if (url.pathname === '/version') {const versionFile=path.join(root,'version');res.end(fs.existsSync(versionFile)?fs.readFileSync(versionFile,'utf8').trim():'personal');return;}
  let file;
  try {file=path.resolve(root,'.'+decodeURIComponent(url.pathname));} catch {res.writeHead(400);res.end();return;}
  if (file !== root && !file.startsWith(root+path.sep)) {res.writeHead(403);res.end();return;}
  if (!fs.existsSync(file) || !fs.statSync(file).isFile()) file=path.join(root,'index.html');
  res.setHeader('Content-Type',mime[path.extname(file)] || 'application/octet-stream');
  res.setHeader('Cache-Control','no-cache');
  fs.createReadStream(file).on('error',()=>{res.statusCode=500;res.end('Website build is missing.');}).pipe(res);
}).listen(9898,'127.0.0.1',()=>console.log('Pokedextracker: http://localhost:9898'));
