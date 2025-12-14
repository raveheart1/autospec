#!/bin/bash
# Mock Claude command for testing workflow scripts
# Simulates Claude Code CLI behavior

COMMAND="$1"
MOCK_BEHAVIOR="${MOCK_CLAUDE_BEHAVIOR:-success}"

# Parse command to determine which speckit command was called
if echo "$COMMAND" | grep -q "/autospec.specify"; then
    if [ "$MOCK_BEHAVIOR" = "success" ]; then
        echo "Creating specification..."
        echo "Specification created successfully"
        exit 0
    else
        echo "Error creating specification"
        exit 1
    fi

elif echo "$COMMAND" | grep -q "/autospec.plan"; then
    if [ "$MOCK_BEHAVIOR" = "success" ]; then
        echo "Creating implementation plan..."
        echo "Plan created successfully"
        exit 0
    else
        echo "Error creating plan"
        exit 1
    fi

elif echo "$COMMAND" | grep -q "/autospec.tasks"; then
    if [ "$MOCK_BEHAVIOR" = "success" ]; then
        echo "Generating task breakdown..."
        echo "Tasks created successfully"
        exit 0
    else
        echo "Error generating tasks"
        exit 1
    fi

elif echo "$COMMAND" | grep -q "/autospec.implement"; then
    if [ "$MOCK_BEHAVIOR" = "success" ]; then
        echo "Implementing feature..."
        echo "Implementation in progress..."
        exit 0
    else
        echo "Error during implementation"
        exit 1
    fi

else
    echo "Unknown command: $COMMAND"
    exit 1
fi
