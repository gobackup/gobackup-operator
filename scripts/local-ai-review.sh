#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Configuration
ARVAN_API_KEY="${ARVAN_API_KEY:-dcaa2cdd-658b-5d43-8a10-c68bf577acf1}"
API_ENDPOINT="https://arvancloudai.ir/gateway/models/GPT-5/dLeqgnGfz1C2zftmNeS4Nd9Q0Fb33Qx5JRqplYEFa4yWAFRC8saj2LMJiyMAjrDOedL4qNXb_PMDMjCSnEFqSCQkgVFnCx3T4Qtu-sruzmH-wRNhYvxHSI34EjG_nfnoOe9djzeqGnhfCJlQ-Z3zUowQQYYeg510o8t8FtAWu35poiK93SAFBYWkU_bbOCe_ZU3gg8dBzzFwpXgD2z4BWzYhz96DR1jTXOanCnI7gw4/v1/chat/completions"

# Help message
show_help() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Run AI code review locally on your git changes"
    echo ""
    echo "Options:"
    echo "  -b, --base BRANCH     Base branch to compare against (default: main)"
    echo "  -h, --head BRANCH     Head branch to review (default: current branch)"
    echo "  -s, --staged          Review only staged changes"
    echo "  -u, --unstaged        Review only unstaged changes"
    echo "  --help                Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0                    # Review current branch against main"
    echo "  $0 -b develop         # Review current branch against develop"
    echo "  $0 -s                 # Review staged changes only"
    echo "  $0 -b main -h feature # Review feature branch against main"
    exit 0
}

# Parse arguments
BASE_BRANCH="main"
HEAD_BRANCH=$(git rev-parse --abbrev-ref HEAD)
STAGED_ONLY=false
UNSTAGED_ONLY=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -b|--base)
            BASE_BRANCH="$2"
            shift 2
            ;;
        -h|--head)
            HEAD_BRANCH="$2"
            shift 2
            ;;
        -s|--staged)
            STAGED_ONLY=true
            shift
            ;;
        -u|--unstaged)
            UNSTAGED_ONLY=true
            shift
            ;;
        --help)
            show_help
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            ;;
    esac
done

echo -e "${GREEN}🤖 Local AI Code Review${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Get the diff
if [ "$STAGED_ONLY" = true ]; then
    echo -e "${YELLOW}Reviewing staged changes...${NC}"
    DIFF=$(git diff --cached)
elif [ "$UNSTAGED_ONLY" = true ]; then
    echo -e "${YELLOW}Reviewing unstaged changes...${NC}"
    DIFF=$(git diff)
else
    echo -e "${YELLOW}Comparing ${HEAD_BRANCH} against ${BASE_BRANCH}...${NC}"
    DIFF=$(git diff ${BASE_BRANCH}...${HEAD_BRANCH})
fi

# Check if there are any changes
if [ -z "$DIFF" ]; then
    echo -e "${RED}No changes found to review.${NC}"
    exit 1
fi

# Save diff to temp file
TEMP_DIR=$(mktemp -d)
DIFF_FILE="${TEMP_DIR}/diff.txt"
echo "$DIFF" > "$DIFF_FILE"

# Check diff size
DIFF_SIZE=$(wc -c < "$DIFF_FILE")
echo -e "Diff size: ${DIFF_SIZE} bytes"

if [ $DIFF_SIZE -gt 15000 ]; then
    echo -e "${YELLOW}Diff is large (>15KB), summarizing...${NC}"
    if [ "$STAGED_ONLY" = true ]; then
        git diff --cached --stat > "$DIFF_FILE"
    elif [ "$UNSTAGED_ONLY" = true ]; then
        git diff --stat > "$DIFF_FILE"
    else
        git diff ${BASE_BRANCH}...${HEAD_BRANCH} --stat > "$DIFF_FILE"
    fi
    echo -e "\n\n---\nNote: Full diff was too large, showing summary only." >> "$DIFF_FILE"
    DIFF=$(cat "$DIFF_FILE")
fi

# Prepare the prompt
read -r -d '' PROMPT << EOM || true
You are an expert code reviewer. Please review the following code changes and provide:
1. A summary of the changes
2. Potential issues or bugs
3. Security concerns if any
4. Performance considerations
5. Best practices and suggestions for improvement
6. Positive aspects of the code

Focus on Go code, Kubernetes manifests, and Helm charts.

Code diff:
${DIFF}
EOM

echo -e "${YELLOW}Calling AI API...${NC}"

# Escape the prompt for JSON
PROMPT_JSON=$(echo "$PROMPT" | jq -Rs .)

# Call ArvanCloud AI API
RESPONSE=$(curl -s -X POST "$API_ENDPOINT" \
    -H "Authorization: Bearer $ARVAN_API_KEY" \
    -H "Content-Type: application/json" \
    -d "{
      \"model\": \"gpt-5\",
      \"messages\": [
        {
          \"role\": \"system\",
          \"content\": \"You are an expert code reviewer specializing in Go, Kubernetes, and cloud-native applications.\"
        },
        {
          \"role\": \"user\",
          \"content\": $PROMPT_JSON
        }
      ],
      \"temperature\": 0.3,
      \"max_tokens\": 2000
    }")

# Check for errors
if echo "$RESPONSE" | jq -e '.error' > /dev/null 2>&1; then
    ERROR_MSG=$(echo "$RESPONSE" | jq -r '.error.message // "Unknown error"')
    echo -e "${RED}API Error: $ERROR_MSG${NC}"
    rm -rf "$TEMP_DIR"
    exit 1
fi

# Extract the review
REVIEW=$(echo "$RESPONSE" | jq -r '.choices[0].message.content // "Error: Unable to get AI review"')

# Display the review
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}📝 AI Review Results:${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "$REVIEW"
echo ""
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Cleanup
rm -rf "$TEMP_DIR"

echo -e "\n${GREEN}✅ Review complete!${NC}"
