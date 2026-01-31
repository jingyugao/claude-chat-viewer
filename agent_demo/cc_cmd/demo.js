const pty = require('node-pty');
const os = require('os');

const shell = 'bash'; 

const ptyProcess = pty.spawn(shell, [], {
  name: 'xterm-color',
  cols: 80,
  rows: 30,
  cwd: process.cwd(),
  env: process.env
});

let buffer = '';

ptyProcess.on('data', function(data) {
  process.stdout.write(data);
  buffer += data;
});

const delay = (ms) => new Promise(resolve => setTimeout(resolve, ms));

/**
 * Waits until the output buffer contains the shell prompt.
 * Note: specific to the current user's prompt ending in "$ ".
 */
async function waitForPrompt() {
  return new Promise((resolve) => {
    const checkInterval = setInterval(() => {
      // Check for the standard non-root shell prompt ending
      if (buffer.includes('$ ')) {
        clearInterval(checkInterval);
        // Clear buffer so we can wait for the next prompt cleanly
        buffer = ''; 
        resolve();
      }
    }, 100);
  });
}

async function runDemo() {
  console.log("--- Starting Bash Pty Session ---");
  
  // Wait for initial prompt
  await waitForPrompt();

  // First interaction
  console.log("\n--- Round 1: Sending 'Hello' ---");
  ptyProcess.write('claude -p "Hello, who are you? (Answer in 1 sentence)"\r');
  
  // Wait for the command to echo and the prompt to return
  // We add a small delay to ensure we don't catch the prompt we just typed if echo is fast,
  // though clearing buffer in waitForPrompt usually handles this.
  await delay(500); 
  await waitForPrompt();

  // Second interaction (using -c to continue context)
  console.log("\n--- Round 2: Asking follow-up ---");
  ptyProcess.write('claude -c -p "What was the last word of your previous answer?"\r');
  
  await delay(500);
  await waitForPrompt();

  console.log("\n--- Exiting ---");
  ptyProcess.kill();
  process.exit();
}

runDemo();
