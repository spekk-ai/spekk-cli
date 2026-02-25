import { test, describe, before, after } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);


describe('Coach CLI', () => {
  let tempDir;

  before(() => {
    const projectRoot = path.join(__dirname, '../../..');
    const tmpBase = path.join(projectRoot, '.tmp');
    if (!fs.existsSync(tmpBase)) {
      fs.mkdirSync(tmpBase, { recursive: true });
    }
    tempDir = fs.mkdtempSync(path.join(tmpBase, 'temp-test-'));
  });

  after(() => {
    if (fs.existsSync(tempDir)) {
      fs.rmSync(tempDir, { recursive: true, force: true });
    }
  });

  test('no top-level meeting command in CLI routing', async () => {
    const projectRoot = path.join(__dirname, '../../..');
    const spekkJs = fs.readFileSync(path.join(projectRoot, 'bin/spekk.js'), 'utf8');

    assert.ok(!spekkJs.includes("case 'meeting':"), 'Should not have top-level meeting command');
    assert.ok(spekkJs.includes('launchCoachAgent(args.slice(1))'),
      'Coach should pass subcommand args through');
  });

  test('coach CLI handles meeting subcommand with transcript file', async () => {
    const transcriptFile = path.join(tempDir, 'test-transcript.txt');
    fs.writeFileSync(transcriptFile, 'We discussed feature X.');

    const projectRoot = path.join(__dirname, '../../..');
    const coachCli = fs.readFileSync(path.join(projectRoot, 'src/coach/cli.js'), 'utf8');

    assert.ok(coachCli.includes("subcommand === 'meeting'"),
      'Coach CLI should handle meeting subcommand');
  });
});
