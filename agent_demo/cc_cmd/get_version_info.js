const pty = require('node-pty');

// 运行 claude -v 来获取类似 /status 的信息，这样它不会进入 TUI 模式，方便解析
const ptyProcess = pty.spawn('claude', ['-v'], {
  name: 'xterm-color',
  cols: 80,
  rows: 30,
  cwd: process.cwd(),
  env: process.env
});

let output = '';

ptyProcess.on('data', function(data) {
  // 简单的 ANSI 清洗
  const clean = data.toString().replace(/[\u001b\u009b][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]/g, '');
  output += clean;
  process.stdout.write(data);
});

ptyProcess.on('exit', () => {
  console.log("\n\n--- 捕获的纯文本总结 ---");
  console.log(output.trim());
  console.log("------------------------");
});

