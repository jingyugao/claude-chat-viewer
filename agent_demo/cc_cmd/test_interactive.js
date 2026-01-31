const pty = require('node-pty');
const fs = require('fs');

const ptyProcess = pty.spawn('claude', [], {
  name: 'xterm-color',
  cols: 80,
  rows: 30,
  cwd: process.cwd(),
  env: process.env
});

let output = '';
ptyProcess.on('data', function(data) {
  output += data;
  process.stdout.write(data);
});

async function test() {
  await new Promise(r => setTimeout(r, 5000));
  console.log("\n--- Sending /help ---");
  ptyProcess.write('/help\r');
  await new Promise(r => setTimeout(r, 5000));
  ptyProcess.kill();
  process.exit();
}
test();

