import { test, describe, before, after } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'url';
import { buildSkillActivationMessage, resolveSkillContent, SKILL_MAP } from '../cli.js';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.join(__dirname, '../../..');


describe('Coach CLI', () => {
  let tempDir;

  before(() => {
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
    const spekkJs = fs.readFileSync(path.join(projectRoot, 'bin/spekk.js'), 'utf8');

    assert.ok(!spekkJs.includes("case 'meeting':"), 'Should not have top-level meeting command');
    assert.ok(spekkJs.includes('launchCoachAgent(args.slice(1))'),
      'Coach should pass subcommand args through');
  });

  test('coach CLI handles meeting subcommand', async () => {
    const coachCli = fs.readFileSync(path.join(projectRoot, 'src/coach/cli.js'), 'utf8');

    assert.ok(coachCli.includes("SKILL_MAP[subcommand]"),
      'Coach CLI should check SKILL_MAP for subcommand');
  });

  describe('Skill content inlining', () => {
    test('SKILL_MAP contains meeting subcommand', () => {
      assert.ok(SKILL_MAP.meeting, 'meeting should be mapped in SKILL_MAP');
      assert.strictEqual(SKILL_MAP.meeting, 'meeting-notes-to-specs-skill.md');
    });

    test('resolveSkillContent returns null for unknown subcommands', () => {
      const result = resolveSkillContent('nonexistent-subcommand');
      assert.strictEqual(result, null, 'Should return null for unmapped subcommand');
    });

    test('resolveSkillContent reads the meeting skill file', () => {
      const content = resolveSkillContent('meeting');
      assert.ok(content, 'Should return skill file content');
      assert.ok(content.includes('## Workflow'),
        'Skill content should include ## Workflow heading');
      assert.ok(content.includes('Extract and categorize into three types'),
        'Skill content should include workflow step about categorization');
    });

    test('buildSkillActivationMessage includes full skill content for meeting subcommand', () => {
      const baseMessage = 'You are the Coach Agent.';
      const message = buildSkillActivationMessage(baseMessage, 'meeting', ['meeting']);

      // Should include the base message
      assert.ok(message.includes(baseMessage),
        'Should include the base activation message');

      // Should include skill content delimiters
      assert.ok(message.includes('<skill-content>'),
        'Should include skill-content opening tag');
      assert.ok(message.includes('</skill-content>'),
        'Should include skill-content closing tag');

      // Should include actual workflow content from the skill file
      assert.ok(message.includes('## Workflow'),
        'Should include ## Workflow heading from skill file');
      assert.ok(message.includes('Extract and categorize into three types'),
        'Should include categorization step from skill file');
      assert.ok(message.includes('## Validation'),
        'Should include ## Validation heading from skill file');

      // Should include the activation instruction
      assert.ok(message.includes('Follow the inlined skill workflow below immediately'),
        'Should instruct the coach to follow the inlined workflow');
    });

    test('buildSkillActivationMessage returns base message for unknown subcommands', () => {
      const baseMessage = 'You are the Coach Agent.';
      const message = buildSkillActivationMessage(baseMessage, 'unknown', ['unknown']);
      assert.strictEqual(message, baseMessage,
        'Should return base message unchanged for unknown subcommand');
    });

    test('buildSkillActivationMessage includes transcript content when file provided', () => {
      const transcriptFile = path.join(tempDir, 'test-transcript.txt');
      fs.writeFileSync(transcriptFile, 'We discussed feature X.');

      const baseMessage = 'You are the Coach Agent.';
      const message = buildSkillActivationMessage(baseMessage, 'meeting', ['meeting', transcriptFile]);

      assert.ok(message.includes('<transcript>'),
        'Should include transcript opening tag');
      assert.ok(message.includes('We discussed feature X.'),
        'Should include transcript content');
      assert.ok(message.includes('Process this transcript now.'),
        'Should instruct to process transcript');
    });

    test('buildSkillActivationMessage throws for missing transcript file', () => {
      const baseMessage = 'You are the Coach Agent.';
      const fakePath = path.join(tempDir, 'nonexistent.txt');

      assert.throws(
        () => buildSkillActivationMessage(baseMessage, 'meeting', ['meeting', fakePath]),
        /Transcript file not found/,
        'Should throw when transcript file does not exist'
      );
    });

    test('no hardcoded workflow steps in cli.js', () => {
      const coachCli = fs.readFileSync(path.join(projectRoot, 'src/coach/cli.js'), 'utf8');

      // The old hardcoded strings should be gone
      assert.ok(!coachCli.includes('Activate your meeting-notes-to-specs skill immediately'),
        'Should not contain hardcoded meeting skill activation text');
      assert.ok(!coachCli.includes('meeting-processing skill active via'),
        'Should not contain hardcoded skill activation description');
    });

    test('workflow content comes from markdown files, not JS strings', () => {
      // Verify the skill file exists and has real content
      const skillPath = path.join(projectRoot, 'specs', 'coach-skills-system', 'meeting-notes-to-specs-skill.md');
      assert.ok(fs.existsSync(skillPath), 'Skill markdown file should exist');

      const skillContent = fs.readFileSync(skillPath, 'utf8');
      const message = buildSkillActivationMessage('base', 'meeting', ['meeting']);

      // The message should contain the actual file content
      assert.ok(message.includes(skillContent),
        'Activation message should contain the full skill file content');
    });
  });
});
