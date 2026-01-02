#!/bin/bash

# Test script to verify the release name template
# This simulates what chart-releaser-action does with the release_name_template

set -e

CHART_YAML="charts/gobackup-operator/Chart.yaml"
TEMPLATE="v{{ .Version }}"

if [ ! -f "$CHART_YAML" ]; then
  echo "Error: Chart.yaml not found at $CHART_YAML"
  exit 1
fi

# Extract version from Chart.yaml
VERSION=$(grep "^version:" "$CHART_YAML" | sed 's/version: *//' | sed 's/"//g' | sed "s/'//g" | sed 's/^[[:space:]]*//' | sed 's/[[:space:]]*$//')

if [ -z "$VERSION" ]; then
  echo "Error: Could not extract version from Chart.yaml"
  exit 1
fi

# Apply the template (replace {{ .Version }} with actual version)
RELEASE_NAME=$(echo "$TEMPLATE" | sed "s/{{ .Version }}/$VERSION/g")

echo "Chart version from Chart.yaml: $VERSION"
echo "Release name template: $TEMPLATE"
echo "Generated release name: $RELEASE_NAME"
echo ""
echo "Expected format: v0.1.0-alpha"
echo "Actual format:   $RELEASE_NAME"
echo ""

if [ "$RELEASE_NAME" = "v$VERSION" ]; then
  echo "✅ Release name format is correct!"
else
  echo "❌ Release name format mismatch!"
  exit 1
fi

