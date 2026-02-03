/**
 * Shared JSDoc type definitions for Claude Code hooks.
 *
 * @fileoverview Type definitions for hook input data and output responses.
 */

// ============================================================================
// Hook Input Types
// ============================================================================

/**
 * Common fields present in all hook input data.
 * @typedef {Object} HookDataBase
 * @property {string} session_id - Unique identifier for the current session
 * @property {string} transcript_path - Path to the session transcript file
 * @property {string} cwd - Current working directory
 * @property {string} permission_mode - Permission mode (e.g., "default")
 * @property {string} hook_event_name - Name of the hook event (e.g., "PreToolUse")
 */

/**
 * Tool input for Bash commands.
 * @typedef {Object} BashToolInput
 * @property {string} command - The bash command to execute
 * @property {string} [description] - Description of what the command does
 * @property {number} [timeout] - Optional timeout in milliseconds
 */

/**
 * Tool input for file operations (Read, Edit, Write).
 * @typedef {Object} FileToolInput
 * @property {string} file_path - The absolute path to the file
 * @property {string} [old_string] - For Edit: the text to replace
 * @property {string} [new_string] - For Edit: the replacement text
 * @property {string} [content] - For Write: the content to write
 * @property {number} [offset] - For Read: line offset
 * @property {number} [limit] - For Read: line limit
 */

/**
 * Tool input for WebFetch operations.
 * @typedef {Object} WebFetchToolInput
 * @property {string} url - The URL to fetch
 * @property {string} prompt - The prompt to run on the fetched content
 */

/**
 * Hook data for PreToolUse events with Bash tool.
 * @typedef {HookDataBase & { tool_name: 'Bash', tool_input: BashToolInput, tool_use_id: string }} BashHookData
 */

/**
 * Hook data for PreToolUse events with file tools (Read, Edit, Write).
 * @typedef {HookDataBase & { tool_name: 'Read' | 'Edit' | 'Write', tool_input: FileToolInput, tool_use_id: string }} FileHookData
 */

/**
 * Hook data for PreToolUse events with WebFetch tool.
 * @typedef {HookDataBase & { tool_name: 'WebFetch', tool_input: WebFetchToolInput, tool_use_id: string }} WebFetchHookData
 */

/**
 * Generic hook data for PreToolUse events.
 * @typedef {Object} PreToolUseHookData
 * @property {string} session_id - Unique identifier for the current session
 * @property {string} transcript_path - Path to the session transcript file
 * @property {string} cwd - Current working directory
 * @property {string} permission_mode - Permission mode (e.g., "default")
 * @property {string} hook_event_name - Name of the hook event
 * @property {string} tool_name - Name of the tool being used
 * @property {Object} tool_input - Tool-specific input parameters
 * @property {string} tool_use_id - Unique identifier for this tool use
 */

// ============================================================================
// Hook Output Types
// ============================================================================

/**
 * Response to block a tool use request.
 * @typedef {Object} BlockResponse
 * @property {'block'} decision - The decision to block the request
 * @property {string} reason - Human-readable reason for blocking
 */

/**
 * Response to allow a tool use request (with optional modifications).
 * @typedef {Object} AllowResponse
 * @property {'allow'} decision - The decision to allow the request
 * @property {Object} [updatedInput] - Optional modified tool input
 */

/**
 * Full hook output structure for PreToolUse hooks.
 * @typedef {Object} PreToolUseOutput
 * @property {boolean} [continue] - Whether to continue processing
 * @property {string} [stopReason] - Reason for stopping
 * @property {boolean} [suppressOutput] - Whether to suppress output
 * @property {string} [systemMessage] - System message to add
 * @property {Object} [hookSpecificOutput] - Hook-specific output
 * @property {'PreToolUse'} hookSpecificOutput.hookEventName - Event name
 * @property {'allow' | 'deny' | 'ask'} [hookSpecificOutput.permissionDecision] - Permission decision
 * @property {string} [hookSpecificOutput.permissionDecisionReason] - Reason for decision
 * @property {Object} [hookSpecificOutput.updatedInput] - Modified tool input
 * @property {string} [hookSpecificOutput.additionalContext] - Additional context for Claude
 */

// Export empty object to make this a module (for ES modules compatibility)
export {};
