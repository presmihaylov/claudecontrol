# Discord Integration Limitations

## Current Status: ⚠️ Limited Implementation

The Discord integration currently supports **HTTP interactions only** (slash commands, buttons, modals) and **does not support regular message events** (including bot mentions).

## Why This Limitation Exists

### Discord vs Slack Architecture
- **Slack**: Uses HTTP webhooks for all events including message events
- **Discord**: Uses WebSocket gateway for message events, HTTP only for interactions

### Technical Implications
- **Message Events**: Require persistent WebSocket connections to Discord's gateway
- **Bot Mentions**: Cannot be detected without WebSocket gateway connection
- **Job Creation**: Current workflow assumes HTTP webhooks that Discord doesn't provide for messages
- **Server Architecture**: Our HTTP-only architecture is incompatible with Discord's WebSocket requirements

## What Currently Works

✅ **Discord HTTP Interactions**
- Slash commands via `/discord/interactions` endpoint
- Ed25519 signature verification
- Guild integration lookup
- Basic interaction processing

## What Doesn't Work

❌ **Message Event Processing**
- Bot mentions in Discord messages
- Regular message-based job creation
- Message reaction handling
- Thread creation from messages

## Future Implementation Options

### Option 1: Slash Commands Only (Recommended)
- **Pros**: Works with existing HTTP architecture, production-ready
- **Cons**: Users must use `/` commands instead of mentioning the bot
- **Implementation**: Complete the slash command handlers in the current Discord events handler

### Option 2: WebSocket Gateway Integration
- **Pros**: Full Discord functionality including message events
- **Cons**: Requires significant architectural changes, persistent connections, complex scaling
- **Implementation**: Requires WebSocket client, connection management, and event handling

### Option 3: Hybrid Approach
- **Pros**: Best of both worlds
- **Cons**: Most complex, requires both WebSocket and HTTP handling
- **Implementation**: Maintain WebSocket connections for events, use HTTP for responses

## Recommendation

For production use, implement **slash commands only** to maintain architectural consistency with the rest of the system. This provides a clean, scalable Discord integration that works within the existing HTTP-based infrastructure.

## Implementation Status

- ✅ Basic Discord handler structure
- ✅ Signature verification
- ✅ Integration lookup
- ❌ Slash command processing
- ❌ Job creation from interactions
- ❌ Agent assignment workflow
- ❌ Message sending functionality