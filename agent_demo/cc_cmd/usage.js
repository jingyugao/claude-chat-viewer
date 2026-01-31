const pty = require('node-pty');

const ptyProcess = pty.spawn('claude', [], {
  name: 'xterm-color',
  cols: 80,
  rows: 30,
  cwd: process.cwd(),
  env: process.env
});

ptyProcess.on('data', function(data) {
  process.stdout.write(data);
});

const delay = (ms) => new Promise(resolve => setTimeout(resolve, ms));

async function runUsageDemo() {
  console.log("--- 正在启动 Claude 交互会话 ---");
  await delay(5000); 

  console.log("\n--- 正在发送 /usage 命令 ---");
  ptyProcess.write('/usage\r');

  await delay(5000); 

  console.log("\n--- 演示结束 ---");
  ptyProcess.kill();
  process.exit();
}

runUsageDemo();