export interface Branch {
  name: string;
  isDefault: boolean;
  isProtected: boolean;
}

export interface WorkflowRun {
  id: number;
  displayTitle: string;
  workflowName: string;
  status: "queued" | "in_progress" | "completed" | "waiting" | "requested" | "pending";
  conclusion:
    | "success"
    | "failure"
    | "cancelled"
    | "timed_out"
    | "action_required"
    | "neutral"
    | "skipped"
    | "stale"
    | "startup_failure"
    | null;
  branch: string;
  headSha: string;
  createdAt: string;
  updatedAt: string;
  htmlUrl: string;
  event: string;
  actor: string;
  runNumber: number;
}

export interface Step {
  name: string;
  status: "queued" | "in_progress" | "completed" | "pending";
  conclusion: string | null;
  number: number;
  startedAt: string | null;
  completedAt: string | null;
}

export interface Job {
  id: number;
  name: string;
  status: "queued" | "in_progress" | "completed" | "pending" | "waiting";
  conclusion: string | null;
  startedAt: string | null;
  completedAt: string | null;
  steps: Step[];
  runnerName: string | null;
}

export type Panel = "branches" | "runs" | "details";
export type View = "main" | "logs";

export interface PollState {
  branchesPoller: ReturnType<typeof setInterval> | null;
  runsPoller: ReturnType<typeof setInterval> | null;
}

export interface LogEntry {
  content: string;
  jobName: string;
  stepName: string;
}

export interface AppState {
  owner: string;
  repo: string;
  branches: Branch[];
  runs: WorkflowRun[];
  jobs: Job[];
  logEntries: LogEntry[];

  selectedBranchIndex: number;
  selectedRunIndex: number;
  selectedJobIndex: number;

  activePanel: Panel;
  currentView: View;

  statusMessage: string;
  errorMessage: string | null;

  loading: {
    branches: boolean;
    runs: boolean;
    jobs: boolean;
    logs: boolean;
  };

  pollIntervalMs: number;
}
