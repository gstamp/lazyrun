import blessed from "neo-blessed";
import type { Widgets } from "neo-blessed";
import type { Branch, WorkflowRun, Job, AppState, Panel, LogEntry } from "./types";

export interface UIComponents {
  screen: Widgets.Screen;
  branchesList: Widgets.ListElement;
  runsList: Widgets.ListElement;
  detailsBox: Widgets.BoxElement;
  detailsContent: Widgets.ScrollableTextElement;
  statusBar: Widgets.BoxElement;
  helpBox: Widgets.BoxElement;
}

function truncate(s: string, max: number): string {
  if (s.length <= max) return s;
  return s.slice(0, max - 1) + "…";
}

function formatRelativeTime(isoDate: string): string {
  const now = Date.now();
  const then = new Date(isoDate).getTime();
  const diffSec = Math.floor((now - then) / 1000);

  if (diffSec < 60) return "now";
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m`;
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h`;
  return `${Math.floor(diffSec / 86400)}d`;
}

function getRunIcon(run: WorkflowRun | { status: string; conclusion: string | null }): string {
  if (run.status === "queued" || run.status === "pending" || run.status === "waiting") {
    return "◌";
  }
  if (run.status === "in_progress" || run.status === "requested") {
    return "●";
  }
  // completed
  switch (run.conclusion) {
    case "success":
      return "✓";
    case "failure":
    case "startup_failure":
      return "✗";
    case "cancelled":
      return "⊘";
    case "timed_out":
      return "⚠";
    case "skipped":
      return "–";
    case "neutral":
      return "◇";
    case "stale":
      return "★";
    case "action_required":
      return "◆";
    default:
      return "?";
  }
}

function getStatusColor(status: string, conclusion: string | null): string {
  if (status === "in_progress" || status === "requested") return "cyan";
  if (status === "queued" || status === "pending" || status === "waiting") return "yellow";
  if (conclusion === "success") return "green";
  if (conclusion === "failure" || conclusion === "startup_failure") return "red";
  if (conclusion === "cancelled") return "grey";
  if (conclusion === "timed_out") return "yellow";
  return "white";
}

const COLORS = {
  border: "#565f89",
  borderFocus: "#7dcfff",
  bg: "#1a1b26",
  bgAlt: "#1f2335",
  bgHighlight: "#2f354a",
  fg: "#a9b1d6",
  fgDim: "#565f89",
  fgBright: "#c0caf5",
  accent: "#7dcfff",
  green: "#9ece6a",
  red: "#f7768e",
  yellow: "#e0af68",
  cyan: "#73daca",
  blue: "#7aa2f7",
  title: "#2ac3de",
};

function makeListStyle(): any {
  return {
    fg: COLORS.fg,
    bg: COLORS.bg,
    border: { fg: COLORS.border },
    label: { fg: COLORS.title },
    selected: {
      fg: COLORS.fgBright,
      bg: COLORS.bgHighlight,
      bold: true,
    },
    item: {
      fg: COLORS.fg,
      bg: COLORS.bg,
    },
  };
}

function makeScrollbar(): any {
  return {
    ch: "░",
    track: { bg: COLORS.bgAlt },
    style: { bg: COLORS.fgDim },
  };
}

/**
 * Create all UI components and return them.
 */
export function createUI(): UIComponents {
  const screen = blessed.screen({
    smartCSR: true,
    title: "lazyrun",
    cursor: { artificial: true, shape: "underline", blink: true, color: "white" } as any,
    dockBorders: true,
    fullUnicode: true,
    terminal: "xterm-256color",
  });

  // ── Branches Panel ──
  const branchesList = blessed.list({
    top: 0,
    left: 0,
    width: "30%",
    height: "70%",
    label: " Branches ",
    border: { type: "line" },
    style: makeListStyle(),
    keys: true,
    vi: true,
    mouse: true,
    scrollbar: makeScrollbar(),
    tags: true,
  });

  // ── Runs Panel ──
  const runsList = blessed.list({
    top: 0,
    left: "30%",
    width: "70%",
    height: "70%",
    label: " Runs ",
    border: { type: "line" },
    style: makeListStyle(),
    keys: true,
    vi: true,
    mouse: true,
    scrollbar: makeScrollbar(),
    tags: true,
  });

  // ── Details Panel ──
  const detailsContent = blessed.scrollabletext({
    top: 0,
    left: 0,
    width: "100%",
    height: "100%",
    content: "",
    style: {
      fg: COLORS.fg,
      bg: COLORS.bg,
    } as any,
    scrollbar: makeScrollbar(),
    mouse: true,
    keys: true,
    vi: true,
  });

  const detailsBox = blessed.box({
    bottom: 0,
    left: 0,
    width: "100%",
    height: "30%",
    label: " Details ",
    border: { type: "line" },
    style: {
      fg: COLORS.fg,
      bg: COLORS.bg,
      border: { fg: COLORS.border },
      label: { fg: COLORS.title },
    } as any,
  });

  detailsBox.append(detailsContent);

  // ── Status Bar ──
  const statusBar = blessed.box({
    bottom: 0,
    left: 0,
    width: "100%",
    height: 1,
    content:
      "  [↑↓] navigate  [Tab] cycle panels  [Enter] select  [r] refresh  [q] quit",
    style: {
      fg: COLORS.fgDim,
      bg: COLORS.bgAlt,
    } as any,
  });

  // ── Help hint overlay ──
  const helpBox = blessed.box({
    bottom: 1,
    right: 1,
    width: "shrink",
    height: "shrink",
    content: " press ? for help ",
    style: {
      fg: COLORS.fgDim,
      bg: COLORS.bgAlt,
    } as any,
    hidden: true,
  });

  // Assemble
  screen.append(branchesList);
  screen.append(runsList);
  screen.append(detailsBox);
  screen.append(statusBar);
  screen.append(helpBox);

  return {
    screen,
    branchesList,
    runsList,
    detailsBox,
    detailsContent,
    statusBar,
    helpBox,
  };
}

/**
 * Update the branches list items.
 */
export function updateBranchesList(
  list: Widgets.ListElement,
  branches: Branch[],
  selectedIndex: number,
  loading: boolean,
): void {
  if (loading) {
    list.setItems(["{yellow-fg}Loading...{/yellow-fg}"]);
    return;
  }

  const items =
    branches.length === 0
      ? ["{red-fg}No branches found{/red-fg}"]
      : branches.map((b, i) => {
          const prefix = b.isDefault ? "{cyan-fg}○{/cyan-fg}" : " ";
          const name = b.isProtected
            ? `{bold}${b.name}{/bold}{yellow-fg} 🔒{/yellow-fg}`
            : b.name;
          const marker = i === selectedIndex ? "{green-fg}▶{/green-fg} " : "  ";
          return `${marker}${prefix} ${name}`;
        });

  list.setItems(items);
  list.select(selectedIndex);
}

/**
 * Update the runs list items.
 */
export function updateRunsList(
  list: Widgets.ListElement,
  runs: WorkflowRun[],
  selectedIndex: number,
  loading: boolean,
  branchName: string,
): void {
  if (loading) {
    list.setLabel(` Runs (${branchName}) `);
    list.setItems(["{yellow-fg}Loading...{/yellow-fg}"]);
    return;
  }

  list.setLabel(` Runs — {cyan-fg}${truncate(branchName, 25)}{/cyan-fg} `);

  if (runs.length === 0) {
    list.setItems(["{yellow-fg}No runs found{/yellow-fg}"]);
    return;
  }

  const items = runs.slice(0, 50).map((r, i) => {
    const color = getStatusColor(r.status, r.conclusion);
    const icon = getRunIcon(r);
    const timeAgo = formatRelativeTime(r.updatedAt);
    const branch = truncate(r.branch, 15);
    const name = truncate(r.displayTitle || r.workflowName, 30);
    const marker = i === selectedIndex ? "{green-fg}▶{/green-fg} " : "  ";
    return `${marker}{${color}-fg}${icon}{/${color}-fg} #{cyan-fg}${r.runNumber}{/cyan-fg} {bold}${name}{/bold} ${branch} ${timeAgo}`;
  });

  list.setItems(items);
  list.select(selectedIndex);
}

/**
 * Build the details text for a run.
 */
export function buildRunDetailsText(
  run: WorkflowRun | null,
  jobs: Job[],
  loading: boolean,
): string {
  if (!run) {
    return "{yellow-fg}Select a workflow run to see details{/yellow-fg}";
  }
  if (loading) {
    return "{yellow-fg}Loading...{/yellow-fg}";
  }

  const lines: string[] = [];

  const runColor = getStatusColor(run.status, run.conclusion);
  const runIcon = getRunIcon(run);
  lines.push(
    ` {bold}#${run.runNumber}{/bold}: ${run.displayTitle || run.workflowName}  {${runColor}-fg}${runIcon} ${run.status}${run.conclusion ? ` / ${run.conclusion}` : ""}{/${runColor}-fg}`,
  );
  lines.push(
    ` {cyan-fg}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━{/cyan-fg}`,
  );
  lines.push(` Branch: {bold}${run.branch}{/bold}`);
  lines.push(` Event: ${run.event}  Actor: ${run.actor}`);
  lines.push(` Created: ${new Date(run.createdAt).toLocaleString()}`);
  lines.push("");

  if (jobs.length === 0) {
    lines.push(" {yellow-fg}No jobs found{/yellow-fg}");
  } else {
    for (const job of jobs) {
      const jobColor = getStatusColor(job.status, job.conclusion);
      const jobIcon = getRunIcon(job);

      const duration = formatDuration(job.startedAt, job.completedAt);
      lines.push(
        ` {${jobColor}-fg}${jobIcon}{/${jobColor}-fg} {bold}${job.name}{/bold}  {${jobColor}-fg}${job.status}${job.conclusion ? ` / ${job.conclusion}` : ""}{/${jobColor}-fg}  {grey-fg}${duration}{/grey-fg}`,
      );

      for (const step of job.steps) {
        const sColor = getStatusColor(step.status, step.conclusion);
        const sIcon = getRunIcon(step);
        const sDuration = formatDuration(step.startedAt, step.completedAt);
        lines.push(
          `   {${sColor}-fg}${sIcon}{/${sColor}-fg} ${step.name}  {grey-fg}${sDuration}{/grey-fg}`,
        );
      }
      lines.push("");
    }
  }

  lines.push(" {grey-fg}[l] view full logs  [r] refresh{/grey-fg}");

  return lines.join("\n");
}

function formatDuration(
  startedAt: string | null,
  completedAt: string | null,
): string {
  if (!startedAt) return "--";
  const start = new Date(startedAt).getTime();
  const end = completedAt ? new Date(completedAt).getTime() : Date.now();
  const diffSec = Math.floor((end - start) / 1000);
  if (diffSec < 60) return `${diffSec}s`;
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ${diffSec % 60}s`;
  return `${Math.floor(diffSec / 3600)}h ${Math.floor((diffSec % 3600) / 60)}m`;
}

/**
 * Set the focus indicator on a panel.
 */
export function setActivePanel(
  branchesList: Widgets.ListElement,
  runsList: Widgets.ListElement,
  detailsBox: Widgets.BoxElement,
  panel: Panel,
): void {
  const defaultBorder = COLORS.border;
  const focusBorder = COLORS.borderFocus;

  (branchesList as any).style.border = {
    fg: panel === "branches" ? focusBorder : defaultBorder,
  };
  (runsList as any).style.border = {
    fg: panel === "runs" ? focusBorder : defaultBorder,
  };
  (detailsBox as any).style.border = {
    fg: panel === "details" ? focusBorder : defaultBorder,
  };

  branchesList.setLabel(
    panel === "branches"
      ? ` {bold}{green-fg}▶{/green-fg} Branches `
      : " Branches ",
  );
  runsList.setLabel(
    panel === "runs"
      ? ` {bold}{green-fg}▶{/green-fg} Runs `
      : " Runs ",
  );
  detailsBox.setLabel(
    panel === "details"
      ? ` {bold}{green-fg}▶{/green-fg} Details `
      : " Details ",
  );
}

/**
 * Update the status bar message.
 */
export function updateStatusBar(
  bar: Widgets.BoxElement,
  message: string,
): void {
  bar.setContent(`  ${message}`);
}

/**
 * Open the full log view - reuses the details panel full-screen.
 */
export function enterLogView(
  detailsBox: Widgets.BoxElement,
  detailsContent: Widgets.ScrollableTextElement,
): void {
  (detailsBox as any).height = "100%";
  (detailsBox as any).top = 0;
  detailsBox.setLabel(" Logs (press Esc to go back) ");
  detailsContent.setScrollPerc(0);
}

export function exitLogView(detailsBox: Widgets.BoxElement): void {
  (detailsBox as any).height = "30%";
  (detailsBox as any).top = undefined;
  (detailsBox as any).bottom = 1;
  detailsBox.setLabel(" Details ");
}

export function setLogContent(
  content: Widgets.ScrollableTextElement,
  logs: LogEntry[],
): void {
  if (logs.length === 0) {
    content.setContent("{yellow-fg}No logs available{/yellow-fg}");
    return;
  }

  const text = logs
    .map((l) => {
      if (l.jobName && !l.stepName && !l.content) {
        return `\n{cyan-fg}━━━ Job: ${l.jobName} ━━━{/cyan-fg}\n`;
      }
      return l.content;
    })
    .join("\n");

  content.setContent(text);
  content.setScrollPerc(0);
}
