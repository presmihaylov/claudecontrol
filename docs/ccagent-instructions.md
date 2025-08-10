# CLAUDE.local.md

This file contains specific instructions for Claude Code instances operating in autonomous mode with minimal human supervision.

## Task Classification and Response Strategy

### Distinguishing Questions vs Tasks

Before taking any action, determine the nature of the user's request:

#### **Questions/Information Requests**
- User asks "How does X work?"
- User asks "What is the current state of Y?"
- User asks "Can you explain Z?"
- User requests documentation, explanations, or analysis

**Response**: Answer directly with information, explanations, or analysis. Do NOT initiate the full task completion workflow.

#### **Task Assignments**  
- User asks to "implement X"
- User asks to "fix Y"
- User asks to "add Z feature"
- User asks to "refactor W"
- User provides work to be completed

**Response**: Follow the complete autonomous workflow outlined below.

### Task Planning and Clarification (PREREQUISITE)

When a task is identified, BEFORE starting implementation:

#### 1. Create Execution Plan
- Analyze the task requirements
- Break down into specific steps
- Identify potential implementation approaches
- Consider dependencies and integration points

#### 2. Identify Ambiguities and Gaps
Based on your execution plan, identify:
- Missing technical specifications
- Unclear requirements or scope
- Ambiguous implementation details
- Integration points that need clarification
- Dependencies on external systems or data

#### 3. Single Clarification Request
Consolidate ALL questions into ONE comprehensive request that asks for:
- Any missing technical details
- Clarification of ambiguous requirements  
- Confirmation of scope boundaries
- Any constraints or preferences
- Expected behavior in edge cases

**Format Example**:
```
I've analyzed your request to [task summary]. Before implementing, I need clarification on a few points:

Execution Plan:
1. [Step 1]
2. [Step 2] 
3. [Step 3]

Questions:
- [Specific question about requirement X]
- [Clarification needed for ambiguous point Y]
- [Technical detail needed for implementation Z]

Once you provide these details, I'll proceed with the complete autonomous implementation.
```

#### 4. Wait for Clarification
- Do NOT proceed with implementation until all questions are answered
- Once clarified, acknowledge and begin the autonomous workflow
- **Exception**: If the task is completely clear with no ambiguities, you may skip the clarification step and proceed directly to implementation

## Autonomous Operation Guidelines (For Confirmed Tasks Only)

**IMPORTANT**: The following workflow steps apply ONLY after a task is clearly defined and all clarifying questions have been answered.

### Task Completion Requirements

Every task must be completed to full working state before considering it done. This means:

1. **Code Implementation**: Write the requested functionality following project patterns
2. **Build Verification**: Ensure all modules build successfully
3. **Test Coverage**: Fix all broken tests and ensure test suite passes
4. **Code Quality**: Ensure linting passes without errors
5. **Code Review Integration**: Apply high-impact improvements from code review

### Mandatory Workflow Steps

#### 1. Implementation Phase
- Write the requested code changes following existing patterns
- Use TodoWrite tool to track implementation progress
- Follow service architecture patterns from CLAUDE.md

#### 2. Build Verification (REQUIRED)
Always verify builds succeed after making changes:
```bash
cd ccbackend && make build     # Backend must build
cd ccagent && make build       # Agent must build  
cd ccfrontend && bun run build # Frontend must build
```
If any build fails, fix the issues before proceeding.

#### 3. Code Review Integration (REQUIRED)
When main implementation is complete:
- Use the CodeReviewer subagent to analyze all changes
- Review the suggestions and identify high-impact improvements
- Apply the most valuable suggestions that improve:
  - Code quality and maintainability
  - Performance optimizations
  - Security enhancements
  - Better adherence to project conventions
- This must happen before running tests to ensure code quality improvements are tested

#### 4. Test Management (REQUIRED)
**ALWAYS use the TestFixer subagent for all test-related operations:**
- Use TestFixer subagent after code review improvements are applied
- TestFixer will run tests and fix any failures
- Never run test commands directly - always delegate to TestFixer
- If TestFixer reports database issues, inform user that `supabase start` is needed

#### 5. Linting Compliance (REQUIRED)
Ensure code passes linting standards:
```bash
cd ccbackend && make lint      # Backend linting
cd ccagent && make lint        # Agent linting
cd ccfrontend && bun run lint  # Frontend linting
```
Fix any linting issues before considering task complete.

### Error Handling Protocol

#### Build Failures
- Fix compilation errors immediately
- Ensure all imports are correct
- Verify type compatibility
- Check for missing dependencies

#### Test Failures
- Use TestFixer subagent to diagnose and fix issues
- If database connection errors occur, report to user
- Retry flaky tests 2-3 times before investigating further
- Focus on business logic tests, avoid infrastructure failure testing

#### Linting Failures
- Address all linting violations
- Follow project code style conventions
- Use auto-fix commands when available (`make lint-fix`, `bun run lint:fix`)

### Quality Assurance Standards

#### Code Standards
- Follow existing patterns and conventions from the codebase
- Use proper error handling and logging patterns
- Implement proper validation in service layers
- Use context-aware database operations

#### Service Layer Requirements
- All service functions must take `ctx context.Context` as first parameter
- Include structured logging with 📋 emoji for entry/exit
- Validate inputs before database operations
- Use proper error wrapping with context

#### Database Operations
- Use real PostgreSQL test schema for testing
- Implement proper cleanup in test `defer` blocks
- Scope all entities to user context for security
- Use sqlx context-aware functions

### Completion Checklist

Before considering any task complete, verify:

- [ ] **Task Classification**: Determined if request is a question (answer directly) or task (follow workflow)
- [ ] **Clarification**: For tasks, created execution plan and asked all clarifying questions in single request
- [ ] **Requirements Confirmed**: All ambiguities resolved and task clearly defined
- [ ] **Implementation**: Code is written and follows project patterns
- [ ] **Builds**: All modules (`ccbackend`, `ccagent`, `ccfrontend`) build successfully
- [ ] **Code Review**: CodeReviewer suggestions have been reviewed and high-impact ones applied
- [ ] **Tests**: TestFixer subagent has been used and all tests pass (after code review improvements)
- [ ] **Linting**: All linting checks pass without errors
- [ ] **Final Verification**: All steps completed in correct order with working, tested, and linted code

### Communication Protocol

- Use TodoWrite tool to track progress and provide visibility
- Report any blockers immediately (database not started, missing dependencies, etc.)
- Provide clear status updates when using subagents
- Summarize what was accomplished when task is complete

### Autonomous Decision Making

When faced with implementation choices:
- Prefer existing patterns over introducing new approaches
- Choose the most maintainable solution
- Follow the principle of least surprise
- Prioritize code clarity and readability
- Use established libraries and utilities already in the project

This autonomous mode is designed for complete task ownership - from initial implementation through final delivery of working, tested, and reviewed code.