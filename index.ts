#!/usr/bin/env bun
/**
 * lazyrun — A TUI for GitHub Actions workflow runs.
 *
 * Usage:
 *   bun run index.ts          # Auto-detect from git remote
 *   bun run index.ts owner/repo  # Specify repo
 */

import { App } from "./src/app";

async function main() {
  let owner: string | undefined;
  let repo: string | undefined;

  const arg = process.argv[2];
  if (arg && arg.includes("/")) {
    const parts = arg.split("/");
    owner = parts[0];
    repo = parts[1];
    if (repo?.endsWith(".git")) {
      repo = repo.slice(0, -4);
    }
  }

  try {
    const app = await App.create(owner, repo);
    await app.start();
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    console.error(`\n  ✗ Error: ${msg}\n`);
    console.error("  Make sure you're in a git repo with a GitHub remote,");
    console.error("  or pass the repo as an argument: lazyrun owner/repo\n");
    process.exit(1);
  }
}

main();
