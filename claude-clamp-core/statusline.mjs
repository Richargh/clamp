#!/usr/bin/env node

// Read JSON input from stdin
let input = '';
for await (const chunk of process.stdin) {
  input += chunk;
}

// Output the status line
console.log('Clamped');
