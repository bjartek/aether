# Debugging Aether

This guide explains how to debug Aether and report issues effectively.

## Debug Logging

Aether includes a debug mode that captures detailed internal logs to help diagnose issues.

### Enable Debug Mode

Run Aether with the `--debug` or `-d` flag:

```bash
aether --debug
```

or

```bash
aether -d
```

This will:
1. Create a file named `aether-debug.log` in the current directory
2. Capture detailed debug-level logs from Aether's internal operations
3. Overwrite any existing `aether-debug.log` file

### What Gets Logged

The debug log captures:
- Configuration loading and validation
- Internal state changes
- UI component lifecycle events
- Error details and stack traces
- Timing information for operations

### Debug Log Location

The debug log is always created in the **current working directory** where you run the `aether` command, not in the project directory.

## Reporting Issues

When reporting a bug or issue with Aether, please include:

### 1. Debug Log

Run Aether with debug logging enabled and reproduce the issue:

```bash
aether --debug
# Reproduce the issue
# Exit Aether
```

Then attach the `aether-debug.log` file to your issue report.

### 2. Configuration File

If you're using a custom `aether.yaml` configuration, include it in your report (remove any sensitive information first).

### 3. Environment Information

Include:
- Operating system (macOS, Linux, Windows)
- Aether version (run `aether --version` if available, or note the commit/release)
- Flow CLI version (run `flow version`)
- Go version (run `go version`)

### 4. Steps to Reproduce

Provide clear steps to reproduce the issue:

1. What command did you run?
2. What did you expect to happen?
3. What actually happened?
4. Can you reproduce it consistently?

### 5. Screenshots (if applicable)

For UI issues, screenshots or terminal recordings can be very helpful.

## Example Issue Report

```markdown
**Description:**
Aether crashes when filtering transactions with special characters.

**Steps to Reproduce:**
1. Start aether with `aether --debug`
2. Navigate to Transactions tab
3. Press `/` to open filter
4. Type `@#$%` in the filter
5. Aether crashes

**Environment:**
- OS: macOS 14.1
- Aether: commit abc123
- Flow CLI: v1.10.0
- Go: 1.21.5

**Debug Log:**
See attached aether-debug.log

**Configuration:**
Using default configuration (no custom aether.yaml)
```

## Common Issues

### Debug Log Not Created

If `aether-debug.log` is not created:
- Check you have write permissions in the current directory
- Verify you're using the `--debug` or `-d` flag
- Check for error messages when starting Aether

### Debug Log Too Large

Debug logs can grow large during long sessions. If you need to share a large log:
1. Compress it: `gzip aether-debug.log`
2. Upload to a file sharing service
3. Link to it in your issue report

### Sensitive Information

Before sharing debug logs:
- Review for any sensitive information (API keys, private data)
- Redact sensitive information if present
- Note in your issue if you've redacted content

## Additional Debugging

### Configuration Validation

To check if your configuration is valid without starting Aether:

```bash
# This will validate and show the effective configuration
aether --config aether.yaml --debug
# Check aether-debug.log for configuration details
```

### Network Issues

If you're having network connectivity issues:
1. Enable debug logging
2. Check the debug log for connection errors
3. Verify your network configuration in `aether.yaml`
4. Check firewall settings for required ports

### Performance Issues

For performance problems:
1. Enable debug logging
2. Note the specific operation that's slow
3. Check debug log for timing information
4. Include system resource usage (CPU, memory) in your report

## Getting Help

- **GitHub Issues**: https://github.com/bjartek/aether/issues
- **Discussions**: https://github.com/bjartek/aether/discussions

When in doubt, include the debug log - it's the most helpful information for diagnosing issues!
