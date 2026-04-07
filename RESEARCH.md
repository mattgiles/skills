Yes. Your current “clone elsewhere, then copy into dot-directories” workflow is the clunkiest common pattern, and people are already moving to better ones. The big picture is: **the skill file format is starting to standardize, but installation and update management are still fragmented.** Claude Code and Codex both use `SKILL.md`-based skills, and Codex explicitly recommends skills for repeatable workflows; Gemini CLI has gone in a parallel direction with first-class **extensions** and built-in install/update/link commands. ([Claude API Docs][1])

What looks like the closest thing to an emergent standard right now is **not** “copy the files,” but one of these:

1. **Symlink from a canonical local repo**
2. **Use a git submodule or subtree inside a project**
3. **Use a small installer/manager CLI that tracks provenance and can update**
4. **Separate project-wide instructions from reusable skills** using things like `AGENTS.md` for repo policy and `SKILL.md` for reusable capabilities. ([GitHub][2])

### What other people seem to be doing

A lot of public repos now describe one of two patterns:

* **One central config repo, symlinked into multiple tools.** For example, `cdbattags/ai` literally describes itself as “one repo, symlinked everywhere,” with separate directories for cross-tool skills plus tool-specific config. ([GitHub][2])
* **Git submodules for team/project distribution.** Repos like `Cortexa-LLC/ai-pack` explicitly recommend a submodule workflow for teams, with `git submodule update --remote` for updates. Another public repo describes its skills collection as “organized as git submodules.” ([GitHub][3])

There is also a newer wave of **skill/package managers** trying to smooth over the rough edges. Examples include:

* `skills` / Vercel-style multi-agent installers that auto-detect agents and symlink into expected directories, as described by Viget’s public repo. ([GitHub][4])
* `openskills`, which presents itself as a universal loader with `install`, `update`, `list`, and a “universal” `.agent/skills/` location to avoid per-agent duplication. ([GitHub][5])
* `skilld`, which is pushing an npm-style dependency model and automatic sync via a `prepare` script. ([GitHub][6])
* `agentspec`, a newer “universal agent skill and sub-agent manager” that focuses on linking, validation, and cross-tool management. ([GitHub][7])

### Is there an actual standard yet?

There is **an emergent content standard**, but not yet a fully settled install/update standard. The strongest common layer is the **Agent Skills format** around `SKILL.md`, which agentskills.io describes as an open format meant to be reused across products. Codex also uses `SKILL.md`, and Claude Code’s official docs define skills the same basic way. ([Agent Skills][8])

But the **distribution story is still unsettled**:

* Claude Code officially supports local skills in the expected directory structure, but its docs do not prescribe a package manager; symlinking clearly exists in practice, and Anthropic even fixed a bug involving `~/.claude` being symlinked. ([Claude API Docs][1])
* Codex is standardizing around **Skills + AGENTS.md**, and has deprecated older “custom prompts” in favor of skills, but again does not define a universal installer. ([OpenAI Developers][9])
* Gemini is furthest along operationally: it has `gemini extensions install`, `list`, `update`, `link`, and release-channel guidance for branches/tags. ([GitHub][10])

So the honest answer is: **the standard is emerging at the file format level, while install/update tooling is still in the “multiple competing conventions” phase.** ([Agent Skills][8])

### The best practical way for you

Given what you described, I would switch to this:

**For personal use across agents:** keep a single local repo like `~/agent-skills`, and symlink each skill into the agent-specific directories instead of copying. That gives you instant updates on `git pull` with no recopying. This pattern is explicitly recommended or demonstrated in several public repos, and Gemini’s official extension system even has a built-in `link` command for local development. ([GitHub][2])

**For project/team use:** put skills in the repo via **git submodule** or sometimes **git subtree**. Submodule is the more visible pattern in current skill repos because it preserves upstream provenance and makes updates explicit. ([GitHub][3])

**For Codex specifically:** use `AGENTS.md` for repository-specific behavior and policies, and reserve `SKILL.md` directories for reusable capabilities you want discoverable across tasks. That separation is directly aligned with OpenAI’s docs. ([OpenAI Developers][11])

### Concretely, a better workflow than yours

A simple manual version:

```bash
mkdir -p ~/src/agent-skills
cd ~/src/agent-skills
git clone git@github.com:someone/skill-repo.git

mkdir -p ~/.claude/skills
ln -s ~/src/agent-skills/skill-repo/some-skill ~/.claude/skills/some-skill

mkdir -p ~/.codex/skills
ln -s ~/src/agent-skills/skill-repo/some-skill ~/.codex/skills/some-skill
```

Then updates are just:

```bash
cd ~/src/agent-skills/skill-repo
git pull
```

No delete-and-recopy cycle. This is basically the pattern public repos are converging on when they are not using a dedicated manager. ([GitHub][2])

### My recommendation

For you, I would rank the options like this:

**Best now:**
**Canonical repo + symlinks** for personal setup across Claude/Codex/Gemini.

**Best for teams:**
**Git submodule** in the project repo.

**Most promising but still early:**
A dedicated manager like **skills**, **openskills**, **skilld**, or **agentspec**. These are where the ecosystem may be heading, but I would still treat them as emerging tooling rather than a settled standard. ([GitHub][4])

So: **yes, there is a better way than copying; no, there is not yet one universally accepted best practice; the de facto best practice today is “single source of truth + symlinks or submodules,” with the `SKILL.md` format itself becoming the most important cross-tool standard.** ([Agent Skills][8])

I can sketch a concrete directory layout and a tiny install/update script for Claude + Codex + Gemini on your machine.

[1]: https://docs.anthropic.com/en/docs/claude-code/skills "Extend Claude with skills - Claude Code Docs"
[2]: https://github.com/cdbattags/ai "GitHub - cdbattags/ai: User & workspace level rules and skills for consistent AI agent behavior across projects · GitHub"
[3]: https://github.com/Cortexa-LLC/ai-pack "GitHub - Cortexa-LLC/ai-pack: Claude Workbench & AI Workflows · GitHub"
[4]: https://github.com/vigetlabs/viget-agent-skills?utm_source=chatgpt.com "vigetlabs/viget-agent-skills"
[5]: https://github.com/numman-ali/openskills?utm_source=chatgpt.com "numman-ali/openskills: Universal skills loader for AI coding ..."
[6]: https://github.com/harlan-zw/skilld?utm_source=chatgpt.com "harlan-zw/skilld: Generate AI agent skills from your NPM ..."
[7]: https://github.com/urmzd/agentspec?utm_source=chatgpt.com "urmzd/agentspec: Universal agent skill and sub- ..."
[8]: https://agentskills.io/home?utm_source=chatgpt.com "Agent Skills: Overview"
[9]: https://developers.openai.com/codex/custom-prompts/ "Custom Prompts – Codex | OpenAI Developers"
[10]: https://github.com/google-gemini/gemini-cli/blob/main/docs/extensions/reference.md "gemini-cli/docs/extensions/reference.md at main · google-gemini/gemini-cli · GitHub"
[11]: https://developers.openai.com/codex/guides/agents-md/ "Custom instructions with AGENTS.md – Codex | OpenAI Developers"
