#!/usr/bin/env node
import { readFileSync } from 'node:fs';

const threshold = Number(process.argv[2] ?? 90);
const coverageFile = process.argv[3] ?? 'coverage/web/coverage-final.json';
const coverage = JSON.parse(readFileSync(coverageFile, 'utf8'));

let covered = 0;
let total = 0;

for (const fileCoverage of Object.values(coverage)) {
  for (const count of Object.values(fileCoverage.s ?? {})) {
    total += 1;
    if (count > 0) {
      covered += 1;
    }
  }
}

const percentage = total === 0 ? 100 : (covered / total) * 100;
if (percentage < threshold) {
  console.error(`frontend statement coverage ${percentage.toFixed(1)}% is below ${threshold.toFixed(1)}%`);
  process.exit(1);
}

console.log(`frontend statement coverage ${percentage.toFixed(1)}% meets ${threshold.toFixed(1)}%`);
