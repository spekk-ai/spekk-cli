import { test, describe } from 'node:test';
import assert from 'node:assert';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const projectRoot = path.join(__dirname, '../../..');

describe('reuses-coach-prompt-as-base', () => {
  test('no standalone meeting-processor agent prompt exists', () => {
    const standalonePrompt = path.join(projectRoot, 'specs/meeting-notes-to-specs/meeting-notes-to-specs.prompt.md');
    assert.ok(
      !fs.existsSync(standalonePrompt),
      'Standalone meeting-processor prompt file should not exist'
    );
  });

  test('no meeting-processor-agent registered in PromptResolver', async () => {
    const { PromptResolver } = await import('../../cli/prompt-resolver.js');
    const resolver = new PromptResolver();

    const hasStandaloneAgent = resolver.promptFiles.some(
      p => p.name === 'meeting-processor-agent'
    );
    assert.ok(
      !hasStandaloneAgent,
      'PromptResolver should not have a meeting-processor-agent entry'
    );
  });

  test('coach prompt is the base for meeting processing', async () => {
    const coachPrompt = fs.readFileSync(
      path.join(projectRoot, 'specs/coach-agent/coach-agent.prompt.md'),
      'utf8'
    );

    // Coach prompt should contain meeting-related skill triggers
    assert.ok(
      coachPrompt.includes('meeting notes') || coachPrompt.includes('Meeting Notes'),
      'Coach prompt should reference meeting notes skill'
    );
    assert.ok(
      coachPrompt.includes('meeting transcript') || coachPrompt.includes('Meeting Transcript'),
      'Coach prompt should reference meeting transcript triggers'
    );
  });

  test('meeting-processing skill adds behavior on top of coach capabilities', async () => {
    // The meeting skill should be a coach skill (extends Skill interface)
    const { MeetingNotesToSpecs } = await import('../meeting-notes-to-specs.js');
    const skill = new MeetingNotesToSpecs();

    // Verify it implements the skill interface
    assert.ok(typeof skill.getId === 'function', 'Should have getId');
    assert.ok(typeof skill.getName === 'function', 'Should have getName');
    assert.ok(typeof skill.shouldTrigger === 'function', 'Should have shouldTrigger');
    assert.ok(typeof skill.getQuestions === 'function', 'Should have getQuestions');
    assert.ok(typeof skill.processResponses === 'function', 'Should have processResponses');

    // Verify it's registered in the skill registry
    const { skillRegistry } = await import('../skills/index.js');
    const registeredSkill = skillRegistry.getSkill('meeting-notes-to-specs');
    assert.ok(registeredSkill, 'Meeting skill should be registered in the skill registry');
  });

  test('spec format definitions are NOT duplicated in a standalone prompt', () => {
    // No standalone prompt should exist that duplicates YAML frontmatter, kebab-case,
    // priority, or status definitions that the coach already knows
    const meetingSpecDir = path.join(projectRoot, 'specs/meeting-notes-to-specs');
    const files = fs.readdirSync(meetingSpecDir);

    // There should be no .prompt.md file in the meeting-notes-to-specs directory
    const promptFiles = files.filter(f => f.endsWith('.prompt.md'));
    assert.strictEqual(
      promptFiles.length,
      0,
      'No standalone prompt files should exist in meeting-notes-to-specs spec directory'
    );
  });

  test('no standalone meeting-processor CLI module exists', () => {
    const standaloneCliDir = path.join(projectRoot, 'src/meeting-processor');
    assert.ok(
      !fs.existsSync(standaloneCliDir),
      'Standalone meeting-processor CLI module should not exist'
    );
  });

  test('coach CLI routes meeting subcommand through coach agent', () => {
    const coachCli = fs.readFileSync(
      path.join(projectRoot, 'src/coach/cli.js'),
      'utf8'
    );

    // Coach CLI should use launchAgentWithPrompt('coach-agent'), not 'meeting-processor-agent'
    assert.ok(
      coachCli.includes("launchAgentWithPrompt('coach-agent')"),
      'Coach CLI should launch coach-agent, not a separate meeting agent'
    );
    assert.ok(
      !coachCli.includes("launchAgentWithPrompt('meeting-processor-agent')"),
      'Coach CLI should NOT launch a standalone meeting-processor-agent'
    );
    assert.ok(
      coachCli.includes("subcommand === 'meeting'"),
      'Coach CLI should handle meeting as a subcommand'
    );
  });

  test('meeting skill follows extensible skills framework pattern', async () => {
    const { MeetingNotesToSpecs } = await import('../meeting-notes-to-specs.js');
    const { Skill } = await import('../skill-interface.js');

    const skill = new MeetingNotesToSpecs();

    // Should extend the Skill base class
    assert.ok(
      skill instanceof Skill,
      'MeetingNotesToSpecs should extend the Skill base class'
    );
  });
});
