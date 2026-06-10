package skills

const xelyonBuiltinSkillCreatorName = "skill-creator"

func xelyonBuiltinSkills() []ParsedSkill {
	return []ParsedSkill{
		xelyonBuiltinSkillCreator(),
	}
}

func xelyonBuiltinSkillCreator() ParsedSkill {
	return ParsedSkill{
		Name:        xelyonBuiltinSkillCreatorName,
		Description: "Create or update XELYON Agent Skills. Use when the user asks to make a skill, turn a repeated workflow into a skill, improve skill triggers, or scaffold SKILL.md/resources under .agents/skills or ~/.agents/skills.",
		Body:        xelyonBuiltinSkillCreatorBody,
		Directory:   "xelyon://skills/skill-creator",
		SkillPath:   "xelyon://skills/skill-creator/SKILL.md",
		Source:      SourceXelyon,
		Routing: &RoutingMetadata{
			Version:    XelyonRoutingMetadataVersion,
			Intents:    []string{"skill-authoring"},
			Role:       RoutingRoleAuthoring,
			ReadOnly:   false,
			Modes:      []string{"authoring"},
			Triggers:   []string{"skill", "skills", "スキル", "スキルズ"},
			Conflicts:  []string{"runtime-execution"},
			Activation: RoutingActivationHint,
		},
	}
}

const xelyonBuiltinSkillCreatorBody = `# XELYON Skill Creator

Use this skill when the user asks to create or update a XELYON Agent Skill.

## Rules

- Do not read, copy, or vendor Codex system skills. This is XELYON-owned authoring guidance.
- Prefer project-local skills at .agents/skills/<name> for repository workflows.
- Use ~/.agents/skills/<name> only when the user explicitly wants a personal/global skill.
- Normalize new skill names to lowercase hyphen-case.
- SKILL.md frontmatter must contain name and description. Keep description trigger-focused because it is shown in prompt metadata.
- Do not add XELYON-specific fields to SKILL.md frontmatter. Keep SKILL.md portable for other skills-compatible agents.
- When the authoring context is clear enough, create or update agents/xelyon.yaml with XELYON routing metadata.
- Keep SKILL.md concise and procedural. Put detailed reference material in references/, deterministic repeated operations in scripts/, and reusable output assets/templates in assets/ only when needed.
- Do not add README, installation guides, changelogs, or other extra docs unless the user directly asks for them.
- After writing or editing a skill, run /skills doctor --routing when available, or manually verify that frontmatter parses, agents/xelyon.yaml uses version: 1, and no duplicate skill name shadows the intended skill.

## Minimal SKILL.md

~~~markdown
---
name: example-skill
description: Use when XELYON should handle a specific repeated workflow, including the concrete trigger conditions.
---

# Example Skill

Use this workflow when the task matches the description.

1. Inspect the task-local context first.
2. Load only the references or scripts needed for this request.
3. Execute the workflow and report the verification performed.
~~~

## Optional agents/xelyon.yaml

Create this file when the routing intent is clear:

~~~yaml
version: 1
intents:
  - skill-authoring
role: authoring
read_only: false
modes:
  - authoring
triggers:
  - skill
  - スキル
conflicts:
  - runtime-execution
activation: hint
~~~
`
