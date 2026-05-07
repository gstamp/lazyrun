import { $ } from "bun";
import type { Branch, WorkflowRun, Job, LogEntry, Step } from "./types";

/**
 * Run a gh command and return stdout as string.
 * Throws on non-zero exit with stderr in the message.
 */
async function gh(...args: string[]): Promise<string> {
  const cmd = ["gh", ...args];
  try {
    const proc = Bun.spawnSync(cmd, {});
    if (proc.exitCode !== 0) {
      const stderr = proc.stderr.toString().trim();
      throw new Error(stderr || `gh command failed: ${cmd.join(" ")}`);
    }
    return proc.stdout.toString().trim();
  } catch (e: unknown) {
    if (e instanceof Error) throw e;
    throw new Error(`gh command failed: ${cmd.join(" ")}`);
  }
}

/**
 * Run a gh api command and parse JSON.
 * Handles paginated responses (one JSON object per line).
 */
async function ghApi<T>(endpoint: string, flags: string[] = []): Promise<T> {
  const result = await gh("api", endpoint, ...flags, "--jq", ".");
  // Paginated responses produce multiple JSON objects, one per line.
  // Try parsing as a single JSON first, then try per-line.
  try {
    return JSON.parse(result) as T;
  } catch {
    // May be paginated — each line is a separate JSON document.
    const lines = result.split("\n").filter((l) => l.trim());
    if (lines.length <= 1) throw new Error(`Could not parse API response: ${result.slice(0, 200)}`);
    return lines.map((l) => JSON.parse(l)) as T;
  }
}

/**
 * Detect owner/repo from the git remote in the current directory.
 */
export async function detectOwnerRepo(): Promise<{
  owner: string;
  repo: string;
}> {
  const proc = Bun.spawnSync([
    "git",
    "remote",
    "get-url",
    "origin",
  ]);
  if (proc.exitCode !== 0) {
    throw new Error("No git remote 'origin' found");
  }
  const remote = proc.stdout.toString().trim();

  // Parse GitHub remote URLs
  // git@github.com:owner/repo.git
  // https://github.com/owner/repo
  const match = remote.match(/github\.com[:\/]([^\/]+)\/([^\/]+?)(?:\.git)?$/);
  if (!match) {
    throw new Error(`Not a GitHub remote: ${remote}`);
  }
  const owner = match[1]!;
  const repo = match[2]!.replace(/\.git$/, "");
  return { owner, repo };
}

/**
 * Fetch branches for a repo.
 */
export async function getBranches(
  owner: string,
  repo: string,
): Promise<Branch[]> {
  const data = await ghApi<{ name: string; protected: boolean }[]>(
    `/repos/${owner}/${repo}/branches?per_page=100`,
    ["--paginate"],
  );

  // Get the default branch separately
  const repoInfo = await ghApi<{ default_branch: string }>(
    `/repos/${owner}/${repo}`,
  );
  const defaultBranch = repoInfo.default_branch;

  // Flatten if paginated (array of arrays)
  const branches = Array.isArray(data)
    ? Array.isArray((data as unknown[])[0])
      ? (data as unknown as { name: string; protected: boolean }[][]).flat()
      : (data as { name: string; protected: boolean }[])
    : [];

  return branches.map((b) => ({
    name: b.name,
    isDefault: b.name === defaultBranch,
    isProtected: b.protected,
  }));
}

interface StepResponse {
  name: string;
  status: string;
  conclusion: string | null;
  number: number;
  started_at: string | null;
  completed_at: string | null;
}

interface JobResponse {
  id: number;
  name: string;
  status: string;
  conclusion: string | null;
  started_at: string | null;
  completed_at: string | null;
  steps: StepResponse[] | null;
  runner_name: string | null;
}

interface RunsResponse {
  workflow_runs: Array<{
    id: number;
    name: string;
    display_title: string;
    status: string;
    conclusion: string | null;
    head_branch: string;
    head_sha: string;
    created_at: string;
    updated_at: string;
    html_url: string;
    event: string;
    actor: { login: string } | null;
    run_number: number;
    workflow_name?: string;
  }>;
}

/**
 * Fetch workflow runs for a repo, optionally filtered by branch.
 */
export async function getRuns(
  owner: string,
  repo: string,
  branch?: string,
  perPage: number = 50,
): Promise<WorkflowRun[]> {
  let endpoint = `/repos/${owner}/${repo}/actions/runs?per_page=${perPage}`;
  if (branch) {
    endpoint += `&branch=${encodeURIComponent(branch)}`;
  }

  const data = await ghApi<RunsResponse[] | RunsResponse>(endpoint, [
    "--paginate",
  ]);

  // Handle both single and paginated responses
  const pages = Array.isArray(data) ? data : [data];
  const runsList = pages.flatMap((p) => (p as RunsResponse).workflow_runs ?? []);

  return runsList.map((r) => ({
    id: r.id,
    displayTitle: r.display_title || r.name || "",
    workflowName: r.workflow_name || "Unknown",
    status: r.status as WorkflowRun["status"],
    conclusion: r.conclusion as WorkflowRun["conclusion"],
    branch: r.head_branch,
    headSha: r.head_sha,
    createdAt: r.created_at,
    updatedAt: r.updated_at,
    htmlUrl: r.html_url,
    event: r.event,
    actor: r.actor?.login ?? "unknown",
    runNumber: r.run_number,
  }));
}

/**
 * Fetch jobs for a specific workflow run.
 */
export async function getJobs(
  owner: string,
  repo: string,
  runId: number,
): Promise<Job[]> {
  const data = await ghApi<{ jobs: JobResponse[] }[] | { jobs: JobResponse[] }>(
    `/repos/${owner}/${repo}/actions/runs/${runId}/jobs?per_page=100`,
    ["--paginate"],
  );

  const pages = Array.isArray(data) ? data : [data];
  const jobsList = pages.flatMap((p) => p.jobs ?? []);

  return jobsList.map((j) => ({
    id: j.id,
    name: j.name,
    status: j.status as Job["status"],
    conclusion: j.conclusion as Job["conclusion"],
    startedAt: j.started_at,
    completedAt: j.completed_at,
    steps: (j.steps ?? []).map(
      (s: StepResponse): Step => ({
        name: s.name,
        status: s.status as Step["status"],
        conclusion: s.conclusion as Step["conclusion"],
        number: s.number,
        startedAt: s.started_at,
        completedAt: s.completed_at,
      }),
    ),
    runnerName: j.runner_name,
  }));
}

/**
 * Fetch logs for a specific job.
 */
export async function getJobLogs(
  owner: string,
  repo: string,
  jobId: number,
): Promise<LogEntry[]> {
  try {
    const proc = Bun.spawnSync([
      "gh",
      "api",
      `/repos/${owner}/${repo}/actions/jobs/${jobId}/logs`,
    ]);

    if (proc.exitCode !== 0) {
      return [
        {
          content: `Failed to fetch logs (exit ${proc.exitCode})`,
          jobName: "",
          stepName: "",
        },
      ];
    }

    const text = proc.stdout.toString();
    const lines = text.split("\n");

    return lines.map((line) => {
      const match = line.match(/^\S+\s+\[([^\]]+)\]\s+(.*)$/);
      return {
        content: line,
        jobName: "",
        stepName: match?.[1] ?? "",
      };
    });
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e);
    return [
      { content: `Error fetching logs: ${msg}`, jobName: "", stepName: "" },
    ];
  }
}

/**
 * Fetch logs for all jobs in a run.
 */
export async function getAllJobLogs(
  owner: string,
  repo: string,
  runId: number,
): Promise<LogEntry[]> {
  const jobs = await getJobs(owner, repo, runId);
  const allLogs: LogEntry[] = [];

  for (const job of jobs) {
    allLogs.push({ content: "", jobName: job.name, stepName: "" });
    allLogs.push({
      content: `━━━ Job: ${job.name} (${job.status}${job.conclusion ? ` / ${job.conclusion}` : ""}) ━━━`,
      jobName: job.name,
      stepName: "",
    });

    const logs = await getJobLogs(owner, repo, job.id);
    allLogs.push(...logs);
    allLogs.push({ content: "", jobName: "", stepName: "" });
  }

  return allLogs;
}
