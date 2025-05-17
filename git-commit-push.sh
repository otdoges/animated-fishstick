#!/bin/bash

# Git commit and push script for Supabase-in-Go project
# Usage: ./git-commit-push.sh "Your commit message"

# Check if commit message is provided
if [ $# -eq 0 ]; then
  echo "Error: Please provide a commit message"
  echo "Usage: ./git-commit-push.sh \"Your commit message\""
  exit 1
fi

# Store the commit message
COMMIT_MESSAGE="$1"

# Navigate to the project root directory
cd "$(dirname "$0")"

# Make sure we're in the project root
if [ ! -d "backend" ] || [ ! -d "front-end" ]; then
  echo "Error: Not in the project root directory"
  exit 1
fi

# Check if git is initialized
if [ ! -d ".git" ]; then
  echo "Initializing git repository..."
  git init
  
  # Check if .gitignore exists
  if [ ! -f ".gitignore" ]; then
    echo "Creating default .gitignore..."
    cat > .gitignore << EOL
# Dependencies
node_modules
/front-end/.pnp
/front-end/.pnp.js
/front-end/.yarn/install-state.gz

# Testing
/front-end/coverage

# Next.js
/front-end/.next/
/front-end/out/

# Production
/front-end/build

# Debug
npm-debug.log*
yarn-debug.log*
yarn-error.log*# Go to project root

# Environment variables
.env
.env*.local

# Vercel
.vercel

# Go binaries
*.exe
*.exe~
*.dll
*.so
*.dylib

# Go test binary
*.test

# Go coverage output
*.out

# Go workspace files
go.work

# Local development files
tmp/
EOL
  fi
fi

# Check if remote origin exists
if ! git remote | grep -q "origin"; then
  echo "No remote 'origin' found. Would you like to add it now? (y/n)"
  read -r response
  if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    echo "Please enter the GitHub repository URL (e.g. https://github.com/username/repo.git):"
    read -r repo_url
    git remote add origin "$repo_url"
  fi
fi

# Stage all files
echo "Staging all files..."
git add .

# Commit with the given message
echo "Committing with message: '$COMMIT_MESSAGE'"
git commit -m "$COMMIT_MESSAGE"

# Push to the remote repository
echo "Pushing to remote repository..."
current_branch=$(git symbolic-ref --short HEAD)
if ! git branch -vv | grep -q "\* $current_branch.*\[origin/$current_branch\]"; then
  git push --set-upstream origin "$current_branch"
else
  git push
fi

echo "Completed successfully!"
