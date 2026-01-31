const pty = require('node-pty');

const ptyProcess = pty.spawn('claude', [], {
  name: 'xterm-color',
  cols: 100,
  rows: 40,
  cwd: process.cwd(),
  env: process.env
});

let output = '';
ptyProcess.on('data', function(data) {
  output += data;
  process.stdout.write(data);
});

async function run() {
  // Wait for startup
  await new Promise(r => setTimeout(r, 6000));
  
  console.log("\n[Sending /stats command...]");
  ptyProcess.write('/stats\r\r');
  
  // Wait for response
  await new Promise(r => setTimeout(r, 8000));
  
  console.log("\n[Session captured]");
  ptyProcess.kill();
  process.exit();
}

run();

