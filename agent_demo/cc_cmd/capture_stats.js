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

async function captureStats() {
  // 等待启动
  await new Promise(r => setTimeout(r, 6000));
  
  console.log("\n[发送 /stats 命令...]");
  ptyProcess.write('/stats\r');
  
  // 给它足够的时间来渲染统计表格
  await new Promise(r => setTimeout(r, 8000));
  
  console.log("\n[捕获完成]");
  ptyProcess.kill();
  process.exit();
}

captureStats();

