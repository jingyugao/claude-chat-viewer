const pty = require('node-pty');
const fs = require('fs');

const ptyProcess = pty.spawn('claude', [], {
  name: 'xterm-color',
  cols: 80,
  rows: 30,
  cwd: process.cwd(),
  env: process.env
});

const stripAnsi = (str) => {
  return str.replace(/[\u001b\u009b][[()#;?]*(?:[0-9]{1,4}(?:;[0-9]{0,4})*)?[0-9A-ORZcf-nqry=><]/g, '');
};

let rawOutput = '';

ptyProcess.on('data', function(data) {
  rawOutput += data;
  process.stdout.write(data);
});

async function inspectStats() {
  console.log("\n--- 启动会话 ---");
  await new Promise(r => setTimeout(r, 6000));

  console.log("\n--- 发送 /status ---");
  ptyProcess.write('/status\r\r');

  // 等待渲染
  await new Promise(r => setTimeout(r, 8000));
  
  console.log("\n--- 分析捕获的数据 ---");
  
  fs.writeFileSync('raw_capture.log', rawOutput);
  console.log("原始数据已保存到 raw_capture.log");

  const cleanText = stripAnsi(rawOutput);
  console.log("\n====== 清洗后的纯文本内容 ======\n");
  
  // 只打印最后 2000 个字符，因为前面的主要是欢迎语
  console.log(cleanText.slice(-2000)); 
  console.log("\n==============================\n");

  ptyProcess.kill();
  process.exit();
}

inspectStats();
