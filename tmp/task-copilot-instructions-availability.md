# Task: Make copilot-instructions.md Available Across All Branches

## Problem
The `.github/copilot-instructions.md` file only exists on the `dev` branch, causing agents to miss critical branching workflow instructions when working on `main` or `feature/*` branches. This leads to repeated violations of the branching strategy (committing directly to `dev` instead of creating feature branches).

## Current State
- File exists: `.github/copilot-instructions.md` (only on `dev` branch)
- File contains: Critical branching workflow, coding conventions, anti-patterns
- Impact: Agents working on feature branches don't see these instructions

## Proposed Solutions

### Option 1: Commit to main and upstream (RECOMMENDED)
**Pros:**
- File naturally available in all branches via inheritance
- No special git configuration needed
- Survives rebases automatically
- Aligns with upstream contribution

**Cons:**
- Requires upstream PR acceptance
- May contain fork-specific workflow details

**Implementation:**
1. Create feature branch from main: `git checkout main && git checkout -b feature/copilot-instructions`
2. Copy file: `git checkout dev -- .github/copilot-instructions.md`
3. Remove/modify any fork-specific sections if needed
4. Commit and push feature branch
5. Create PR to upstream
6. After merge upstream, all branches get it automatically

### Option 2: Symlink to version-controlled copy in root
**Pros:**
- Works immediately without upstream coordination
- Single source of truth

**Cons:**
- Symlinks in git can be problematic
- May not work well across different systems

**Implementation:**
1. Move to root: `mv .github/copilot-instructions.md copilot-instructions.md`
2. Create symlink: `ln -s ../copilot-instructions.md .github/copilot-instructions.md`
3. Commit both to main

### Option 3: Git attributes with merge strategy
**Pros:**
- Technical solution using git features
- Keeps file where it is

**Cons:**
- Complex, non-standard workflow
- Requires all developers to understand configuration
- May cause confusion

**Implementation:**
1. Add to `.gitattributes`: `.github/copilot-instructions.md merge=ours`
2. Configure merge driver in main
3. Cherry-pick from dev to main
4. Every feature branch inherits from main

## Recommended Action
**Use Option 1** - Commit to main and create upstream PR. This is the cleanest long-term solution. If fork-specific content exists:
- Create a separate `FORK-WORKFLOW.md` for fork-specific details
- Keep `.github/copilot-instructions.md` general enough for upstream

## Files to Modify
- `.github/copilot-instructions.md` (review for fork-specific content)
- Potentially create new `FORK-WORKFLOW.md` if needed

## Testing
After implementation, verify on a new feature branch:
```bash
git checkout main
git pull
git checkout -b feature/test-instructions
cat .github/copilot-instructions.md  # Should show content
```

## Success Criteria
- [ ] File exists and is readable from `main` branch
- [ ] File exists and is readable from `feature/*` branches
- [ ] File survives rebases from upstream
- [ ] No duplicate/conflicting instructions
